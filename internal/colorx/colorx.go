// Package colorx implements scene-referred to display-referred conversion:
// Reinhard tone mapping followed by the IEC 61966-2-1 sRGB transfer curve.
// Everything here is plain math and uses only the standard library.
package colorx

import "math"

// SRGBEncode maps one linear-light channel in [0,1] to an sRGB-encoded
// byte value in [0,255]. The ~2.4 power curve is applied once here; writing
// the matching gAMA/cHRM metadata into the PNG tells viewers not to apply
// gamma a second time.
func SRGBEncode(linear float64) uint8 {
	var v float64
	switch {
	case linear <= 0:
		v = 0
	case linear <= 0.0031308:
		v = 12.92 * linear
	default:
		v = 1.055*math.Pow(linear, 1.0/2.4) - 0.055
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

// SRGBDecode inverts SRGBEncode: encoded [0,1] value back to linear light.
func SRGBDecode(enc float64) float64 {
	if enc <= 0.04045 {
		return enc / 12.92
	}
	return math.Pow((enc+0.055)/1.055, 2.4)
}

// Reinhard applies global Reinhard tone mapping with an exposure scale.
// exposure > 1 brightens midtones; the roll-off compresses highlights above
// SDR white (linear 1.0) instead of hard clipping them.
func Reinhard(linear float64, exposure float64) float64 {
	x := linear * exposure
	return x / (1 + x)
}

// TonemapTriplet maps a linear HDR triplet to an 8-bit sRGB triplet and also
// returns the post-tonemap linear value of the red channel range check.
func TonemapTriplet(r, g, b float64, exposure float64) (uint8, uint8, uint8) {
	return SRGBEncode(Reinhard(r, exposure)),
		SRGBEncode(Reinhard(g, exposure)),
		SRGBEncode(Reinhard(b, exposure))
}

// MaxCodeValue tonemaps a single peak value and reports its resulting byte.
// It makes the "maximum code value" panel number reproducible from the
// actual scene peak rather than hard-coded 255.
func MaxCodeValue(peakLinear, exposure float64) uint8 {
	return SRGBEncode(Reinhard(peakLinear, exposure))
}
