package aliaslab

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// RGB is one opaque PPM pixel. Keeping channels as bytes makes the valid and
// reference renders directly comparable without any tolerance.
type RGB struct {
	R, G, B uint8
}

// Image is a tightly packed RGB image.
type Image struct {
	Width  int
	Height int
	Pix    []RGB
}

func NewImage(width, height int) *Image {
	return &Image{Width: width, Height: height, Pix: make([]RGB, width*height)}
}

func (img *Image) Clone() *Image {
	out := NewImage(img.Width, img.Height)
	copy(out.Pix, img.Pix)
	return out
}

func (img *Image) At(x, y int) RGB {
	return img.Pix[y*img.Width+x]
}

func (img *Image) Set(x, y int, c RGB) {
	img.Pix[y*img.Width+x] = c
}

func (img *Image) Equal(other *Image) bool {
	if img.Width != other.Width || img.Height != other.Height {
		return false
	}
	if len(img.Pix) != len(other.Pix) {
		return false
	}
	for i := range img.Pix {
		if img.Pix[i] != other.Pix[i] {
			return false
		}
	}
	return true
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerp8(a, b uint8, t float64) uint8 {
	t = clamp01(t)
	return clamp8(int(float64(a)*(1-t) + float64(b)*t + 0.5))
}

func mixRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: lerp8(a.R, b.R, t),
		G: lerp8(a.G, b.G, t),
		B: lerp8(a.B, b.B, t),
	}
}

func addRGB(a, b RGB) RGB {
	return RGB{
		R: clamp8(int(a.R) + int(b.R)),
		G: clamp8(int(a.G) + int(b.G)),
		B: clamp8(int(a.B) + int(b.B)),
	}
}

func scaleRGB(c RGB, factor float64) RGB {
	return RGB{
		R: clamp8(int(float64(c.R) * factor)),
		G: clamp8(int(float64(c.G) * factor)),
		B: clamp8(int(float64(c.B) * factor)),
	}
}

// WritePPM writes a binary P6 PPM via a temporary file and rename.
func WritePPM(path string, img *Image) error {
	if img.Width <= 0 || img.Height <= 0 || len(img.Pix) != img.Width*img.Height {
		return fmt.Errorf("invalid image %dx%d with %d pixels", img.Width, img.Height, len(img.Pix))
	}
	var body bytes.Buffer
	fmt.Fprintf(&body, "P6\n%d %d\n255\n", img.Width, img.Height)
	for _, c := range img.Pix {
		body.WriteByte(c.R)
		body.WriteByte(c.G)
		body.WriteByte(c.B)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ppm-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(body.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
