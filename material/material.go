// Package material reloads texture containers and evaluates their texels as
// linear material samples. It contains no rendering or GPU calls.
package material

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"

	"texturelab/colorspace"
	"texturelab/texcontainer"
)

// Material is the small subset of a render material needed for this pipeline.
type Material struct {
	Name   string
	Albedo *texcontainer.Texture
	Normal *texcontainer.Texture
}

// LoadFile reads and decodes one texture container path.
func LoadFile(path string) (*texcontainer.Texture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	texture, err := texcontainer.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return texture, nil
}

// LoadMaterial reloads albedo and normal containers from disk.
func LoadMaterial(name, albedoPath, normalPath string) (*Material, error) {
	albedo, err := LoadFile(albedoPath)
	if err != nil {
		return nil, err
	}
	normal, err := LoadFile(normalPath)
	if err != nil {
		return nil, err
	}
	return &Material{Name: name, Albedo: albedo, Normal: normal}, nil
}

// SampleLinear returns the material-space value for one texel and mip level.
func SampleLinear(texture *texcontainer.Texture, level, x, y int) ([4]float64, error) {
	if texture == nil {
		return [4]float64{}, fmt.Errorf("material: nil texture")
	}
	if level < 0 || level >= len(texture.Levels) {
		return [4]float64{}, fmt.Errorf("material: level %d out of range", level)
	}
	mip := texture.Levels[level]
	if x < 0 || y < 0 || x >= mip.Width || y >= mip.Height {
		return [4]float64{}, fmt.Errorf("material: texel (%d,%d) outside level %d %dx%d",
			x, y, level, mip.Width, mip.Height)
	}
	offset := (y*mip.Width + x) * texcontainer.BytesPerPixel
	return colorspace.LinearizeRGBA(texture.Space,
		mip.Pixels[offset+0], mip.Pixels[offset+1],
		mip.Pixels[offset+2], mip.Pixels[offset+3]), nil
}

// LinearFingerprint hashes every linearized sample in one level.
//
// Equal fingerprints before writing and after reloading prove that bytes and
// the sRGB/linear interpretation survived the container round trip.
func LinearFingerprint(texture *texcontainer.Texture, level int) (string, error) {
	if texture == nil || level < 0 || level >= len(texture.Levels) {
		return "", fmt.Errorf("material: invalid fingerprint level %d", level)
	}
	hash := sha256.New()
	mip := texture.Levels[level]
	for y := 0; y < mip.Height; y++ {
		for x := 0; x < mip.Width; x++ {
			sample, err := SampleLinear(texture, level, x, y)
			if err != nil {
				return "", err
			}
			var encoded [32]byte
			for channel, value := range sample {
				binary.LittleEndian.PutUint64(encoded[channel*8:(channel+1)*8], math.Float64bits(value))
			}
			hash.Write(encoded[:])
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// AllLinearFingerprints returns one stable material-space hash per mip level.
func AllLinearFingerprints(texture *texcontainer.Texture) ([]string, error) {
	result := make([]string, 0, len(texture.Levels))
	for level := range texture.Levels {
		fingerprint, err := LinearFingerprint(texture, level)
		if err != nil {
			return nil, err
		}
		result = append(result, fingerprint)
	}
	return result, nil
}
