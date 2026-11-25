package main

import (
	"image"
	"image/color"
	"math"
)

// Expose applies the EV gain in linear light, then tone-maps to sRGB
// 8-bit. A soft shoulder keeps window highlights from hard-clipping.
func Expose(f *Frame, ev float64) *image.NRGBA {
	gain := math.Exp2(ev)
	img := image.NewNRGBA(image.Rect(0, 0, f.W, f.H))
	for i := 0; i < f.W*f.H; i++ {
		r := toneMap(f.R[i] * gain)
		g := toneMap(f.G[i] * gain)
		b := toneMap(f.B[i] * gain)
		img.SetNRGBA(i%f.W, i/f.W, color.NRGBA{
			R: to8(r), G: to8(g), B: to8(b), A: 255,
		})
	}
	return img
}

// toneMap compresses linear light with a filmic-ish shoulder and
// applies the sRGB transfer curve.
func toneMap(x float64) float64 {
	if x < 0 {
		x = 0
	}
	// Soft shoulder: x/(1+x) above a knee keeps highlight detail.
	const knee = 0.8
	if x > knee {
		x = knee + (x-knee)/(1+(x-knee)*2.5)
		if x > 1 {
			x = 1
		}
	}
	return linearToSRGB(x)
}

func linearToSRGB(x float64) float64 {
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1.0/2.4) - 0.055
}

func to8(x float64) uint8 {
	if x < 0 {
		x = 0
	}
	if x > 1 {
		x = 1
	}
	return uint8(math.Round(x * 255))
}
