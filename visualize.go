package main

import (
	"image/color"
)

func matchMaskColor() color.RGBA {
	return color.RGBA{R: 7, G: 12, B: 24, A: 255}
}

func TestDensityColor(tests, total int) color.RGBA {
	if total <= 0 || tests <= 0 {
		return color.RGBA{R: 8, G: 12, B: 26, A: 255}
	}
	t := Clamp01(float64(tests) / float64(total))
	low := Vec3{X: 0.05, Y: 0.09, Z: 0.24}
	mid := Vec3{X: 0.10, Y: 0.62, Z: 0.78}
	high := Vec3{X: 0.98, Y: 0.72, Z: 0.20}
	var c Vec3
	if t < 0.5 {
		c = low.Lerp(mid, t*2)
	} else {
		c = mid.Lerp(high, (t-0.5)*2)
	}
	return color.RGBA{
		R: uint8(c.X*255 + .5),
		G: uint8(c.Y*255 + .5),
		B: uint8(c.Z*255 + .5),
		A: 255,
	}
}

func mismatchMaskColor(kind MismatchKind) color.RGBA {
	switch kind {
	case KindMissed:
		return color.RGBA{R: 255, G: 38, B: 74, A: 255}
	case KindFalseHit:
		return color.RGBA{R: 255, G: 176, B: 32, A: 255}
	case KindWrongTriangle:
		return color.RGBA{R: 232, G: 64, B: 255, A: 255}
	case KindCountMismatch:
		return color.RGBA{R: 32, G: 226, B: 255, A: 255}
	case KindDepthMismatch:
		return color.RGBA{R: 80, G: 120, B: 255, A: 255}
	default:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
}
