// Package colorspace implements deterministic sRGB and linear sample conversion.
// It intentionally uses only the Go standard library.
package colorspace

import (
	"fmt"
	"math"
)

// Space describes the transfer encoding stored in texture texels.
type Space uint8

const (
	// Linear means texel values are linear color or unencoded data.
	Linear Space = 0
	// SRGB means red, green and blue texels use the sRGB transfer curve.
	SRGB Space = 1
)

// String returns the stable metadata name for a color space marker.
func (s Space) String() string {
	switch s {
	case Linear:
		return "linear"
	case SRGB:
		return "srgb"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(s))
	}
}

// Valid reports whether the marker is one of the supported values.
func (s Space) Valid() bool {
	return s == Linear || s == SRGB
}

// Parse converts a metadata flag such as "linear" or "srgb" to a Space.
func Parse(value string) (Space, error) {
	switch value {
	case "", "linear", "Linear", "LINEAR":
		return Linear, nil
	case "srgb", "sRGB", "SRGB":
		return SRGB, nil
	default:
		return 0, fmt.Errorf("colorspace: unsupported marker %q", value)
	}
}

// SRGBToLinear converts one encoded sRGB channel in [0, 255] to linear [0, 1].
func SRGBToLinear(channel uint8) float64 {
	c := float64(channel) / 255.0
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// LinearToSRGB converts a linear channel in [0, 1] to an encoded byte.
func LinearToSRGB(value float64) uint8 {
	if math.IsNaN(value) {
		return 0
	}
	value = math.Min(1, math.Max(0, value))
	var encoded float64
	if value <= 0.0031308 {
		encoded = value * 12.92
	} else {
		encoded = 1.055*math.Pow(value, 1.0/2.4) - 0.055
	}
	return uint8(math.Round(encoded * 255.0))
}

// LinearizeRGBA returns normalized linear material samples.
//
// Alpha is coverage and is never passed through the sRGB transfer curve.
func LinearizeRGBA(space Space, r, g, b, a uint8) [4]float64 {
	alpha := float64(a) / 255.0
	if space == SRGB {
		return [4]float64{
			SRGBToLinear(r),
			SRGBToLinear(g),
			SRGBToLinear(b),
			alpha,
		}
	}
	return [4]float64{
		float64(r) / 255.0,
		float64(g) / 255.0,
		float64(b) / 255.0,
		alpha,
	}
}

// Delta returns the maximum absolute RGB difference between two samples.
// Alpha is excluded because an sRGB tag must not change alpha interpretation.
func Delta(a, b [4]float64) float64 {
	return math.Max(math.Abs(a[0]-b[0]),
		math.Max(math.Abs(a[1]-b[1]), math.Abs(a[2]-b[2])))
}
