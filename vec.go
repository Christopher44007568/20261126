package main

import (
	"image/color"
	"math"
)

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Mul(b Vec3) Vec3 { return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }

func (a Vec3) Scale(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Len2() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.Len2()) }

func (a Vec3) Normalize() Vec3 {
	length := a.Len()
	if length == 0 || math.IsNaN(length) || math.IsInf(length, 0) {
		return V(0, 0, 0)
	}
	return a.Scale(1 / length)
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Scale(t))
}

func Clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func SmoothStep(edge0, edge1, x float64) float64 {
	t := Clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func ColorToRGBA(c Vec3) color.RGBA {
	c.X = Clamp01(c.X)
	c.Y = Clamp01(c.Y)
	c.Z = Clamp01(c.Z)

	// ACES-ish compact roll-off, then gamma. The formula is shared by both
	// render paths, so it cannot create an accelerator/reference mismatch.
	mapped := func(x float64) float64 {
		x = (x * (2.51*x + 0.03)) / (x*(2.43*x+0.59) + 0.14)
		return math.Pow(Clamp01(x), 1/2.2)
	}
	return color.RGBA{
		R: uint8(mapped(c.X)*255 + 0.5),
		G: uint8(mapped(c.Y)*255 + 0.5),
		B: uint8(mapped(c.Z)*255 + 0.5),
		A: 255,
	}
}

func HeatColor(t float64) color.RGBA {
	t = Clamp01(t)
	type stop struct {
		at float64
		c  Vec3
	}
	stops := []stop{
		{0, V(0.03, 0.05, 0.12)},
		{0.25, V(0.10, 0.28, 0.90)},
		{0.50, V(0.00, 0.82, 0.88)},
		{0.75, V(1.00, 0.86, 0.20)},
		{1.00, V(0.92, 0.12, 0.24)},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			f := (t - stops[i-1].at) / (stops[i].at - stops[i-1].at)
			c := stops[i-1].c.Lerp(stops[i].c, f)
			return color.RGBA{R: uint8(c.X*255 + .5), G: uint8(c.Y*255 + .5), B: uint8(c.Z*255 + .5), A: 255}
		}
	}
	return color.RGBA{R: 235, G: 31, B: 61, A: 255}
}

func CountColor(n int) color.RGBA {
	switch {
	case n <= 0:
		return color.RGBA{R: 10, G: 14, B: 30, A: 255}
	case n == 1:
		return color.RGBA{R: 31, G: 119, B: 180, A: 255}
	case n == 2:
		return color.RGBA{R: 32, G: 180, B: 190, A: 255}
	case n == 3:
		return color.RGBA{R: 80, G: 220, B: 120, A: 255}
	case n == 4:
		return color.RGBA{R: 240, G: 220, B: 70, A: 255}
	case n == 5:
		return color.RGBA{R: 245, G: 145, B: 40, A: 255}
	default:
		return color.RGBA{R: 220, G: 45, B: 55, A: 255}
	}
}
