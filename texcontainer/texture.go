// Package texcontainer implements a deterministic KTX-style binary texture
// container without bindings to a graphics API or SDK.
package texcontainer

import (
	"errors"
	"fmt"

	"texturelab/colorspace"
)

// Format identifies the only pixel layout implemented by this package.
type Format uint32

const (
	// FormatRGBA8 stores four bytes per texel in R, G, B, A order.
	FormatRGBA8 Format = 1
	// BytesPerPixel is the RGBA8 texel stride.
	BytesPerPixel = 4
)

// Level is one mip image. Level zero is the largest image.
type Level struct {
	Width  int
	Height int
	Pixels []byte
}

// Texture is the in-memory representation of a container.
type Texture struct {
	Format   Format
	Space    colorspace.Space
	Levels   []Level
	Metadata map[string]string
}

// Common validation errors.
var (
	ErrNoLevels       = errors.New("texcontainer: texture has no mip levels")
	ErrInvalidSize    = errors.New("texcontainer: invalid level dimensions")
	ErrInvalidPixels  = errors.New("texcontainer: pixel length does not match dimensions")
	ErrInvalidSpace   = errors.New("texcontainer: invalid color space marker")
	ErrInvalidFormat  = errors.New("texcontainer: unsupported format")
	ErrInvalidMipSize = errors.New("texcontainer: level is not the expected half-size mip")
)

// Validate checks structural invariants and exact RGBA8 payload lengths.
func (t *Texture) Validate() error {
	if t == nil {
		return errors.New("texcontainer: nil texture")
	}
	if t.Format != FormatRGBA8 {
		return fmt.Errorf("%w: %d", ErrInvalidFormat, t.Format)
	}
	if !t.Space.Valid() {
		return fmt.Errorf("%w: %d", ErrInvalidSpace, t.Space)
	}
	if len(t.Levels) == 0 {
		return ErrNoLevels
	}
	for i := range t.Levels {
		level := &t.Levels[i]
		if level.Width <= 0 || level.Height <= 0 {
			return fmt.Errorf("%w at level %d: %dx%d", ErrInvalidSize, i, level.Width, level.Height)
		}
		want := level.Width * level.Height * BytesPerPixel
		if len(level.Pixels) != want {
			return fmt.Errorf("%w at level %d: got %d bytes, want %d",
				ErrInvalidPixels, i, len(level.Pixels), want)
		}
	}
	return nil
}

// ValidateMipChain verifies that every level is an exact power-of-two halving
// of level zero. The requested sample chain is 512, 256 and 128 pixels.
func (t *Texture) ValidateMipChain() error {
	if err := t.Validate(); err != nil {
		return err
	}
	base := t.Levels[0]
	if base.Width != base.Height {
		return fmt.Errorf("%w: square base required, got %dx%d",
			ErrInvalidMipSize, base.Width, base.Height)
	}
	for i, level := range t.Levels {
		want := ExpectedMipDimensions(base.Width, base.Height, i)
		if level.Width != want[0] || level.Height != want[1] {
			return fmt.Errorf("%w at level %d: got %dx%d, want %dx%d",
				ErrInvalidMipSize, i, level.Width, level.Height, want[0], want[1])
		}
	}
	return nil
}

// ExpectedMipDimensions returns floor(width/2^level), stopping at one texel.
func ExpectedMipDimensions(baseWidth, baseHeight, level int) [2]int {
	width := baseWidth >> level
	height := baseHeight >> level
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return [2]int{width, height}
}

// Clone returns a deep copy so diagnostic retagging cannot mutate source pixels.
func (t *Texture) Clone() *Texture {
	if t == nil {
		return nil
	}
	clone := &Texture{
		Format: t.Format,
		Space:  t.Space,
		Levels: make([]Level, len(t.Levels)),
	}
	if t.Metadata != nil {
		clone.Metadata = make(map[string]string, len(t.Metadata))
		for key, value := range t.Metadata {
			clone.Metadata[key] = value
		}
	}
	for i := range t.Levels {
		clone.Levels[i] = Level{
			Width:  t.Levels[i].Width,
			Height: t.Levels[i].Height,
			Pixels: append([]byte(nil), t.Levels[i].Pixels...),
		}
	}
	return clone
}

// Retag returns a pixel-identical copy with a different color space marker.
func (t *Texture) Retag(space colorspace.Space) *Texture {
	clone := t.Clone()
	clone.Space = space
	if clone.Metadata != nil {
		clone.Metadata["colorspace"] = space.String()
	}
	return clone
}
