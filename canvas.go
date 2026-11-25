package main

import (
	"image"
	"image/color"
	"math"
)

func newCanvas(w, h int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	return img
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	i := img.PixOffset(x, y)
	a := float64(c.A) / 255 * alpha
	img.Pix[i] = uint8(float64(img.Pix[i])*(1-a) + float64(c.R)*a)
	img.Pix[i+1] = uint8(float64(img.Pix[i+1])*(1-a) + float64(c.G)*a)
	img.Pix[i+2] = uint8(float64(img.Pix[i+2])*(1-a) + float64(c.B)*a)
	img.Pix[i+3] = 255
}

func fillCircle(img *image.RGBA, p Vec2, radius float64, c color.RGBA) {
	r := int(math.Ceil(radius))
	for y := int(math.Floor(p.Y)) - r; y <= int(math.Ceil(p.Y))+r; y++ {
		for x := int(math.Floor(p.X)) - r; x <= int(math.Ceil(p.X))+r; x++ {
			dx := float64(x) - p.X
			dy := float64(y) - p.Y
			d := math.Hypot(dx, dy)
			if d <= radius-0.5 {
				img.SetRGBA(x, y, c)
			} else if d <= radius+0.5 {
				blendPixel(img, x, y, c, radius+0.5-d)
			}
		}
	}
}

func strokeCircle(img *image.RGBA, p Vec2, radius, width float64, c color.RGBA) {
	outer, inner := radius+width/2, radius-width/2
	r := int(math.Ceil(outer + 1))
	for y := int(math.Floor(p.Y)) - r; y <= int(math.Ceil(p.Y))+r; y++ {
		for x := int(math.Floor(p.X)) - r; x <= int(math.Ceil(p.X))+r; x++ {
			d := math.Hypot(float64(x)-p.X, float64(y)-p.Y)
			if d <= outer && d >= inner {
				img.SetRGBA(x, y, c)
			} else if math.Abs(d-radius) <= width/2+0.7 {
				blendPixel(img, x, y, c, math.Max(0, 1-math.Abs(d-radius)/(width/2+0.7)))
			}
		}
	}
}

func drawLine(img *image.RGBA, a, b Vec2, width float64, c color.RGBA) {
	dist := math.Hypot(b.X-a.X, b.Y-a.Y)
	steps := int(math.Ceil(dist / math.Max(1, width*0.35)))
	if steps < 1 {
		steps = 1
	}
	r := width / 2
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		fillCircle(img, Vec2{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}, r, c)
	}
}

func drawCross(img *image.RGBA, p Vec2, size, width float64, c color.RGBA) {
	drawLine(img, Vec2{X: p.X - size, Y: p.Y - size}, Vec2{X: p.X + size, Y: p.Y + size}, width, c)
	drawLine(img, Vec2{X: p.X - size, Y: p.Y + size}, Vec2{X: p.X + size, Y: p.Y - size}, width, c)
}

func drawPlus(img *image.RGBA, p Vec2, size, width float64, c color.RGBA) {
	drawLine(img, Vec2{X: p.X - size, Y: p.Y}, Vec2{X: p.X + size, Y: p.Y}, width, c)
	drawLine(img, Vec2{X: p.X, Y: p.Y - size}, Vec2{X: p.X, Y: p.Y + size}, width, c)
}

func heatColor(t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	stops := []struct {
		at  float64
		rgb [3]float64
	}{
		{0.0, [3]float64{28, 32, 92}},
		{0.25, [3]float64{0, 168, 220}},
		{0.5, [3]float64{48, 210, 120}},
		{0.75, [3]float64{255, 220, 48}},
		{1.0, [3]float64{220, 38, 38}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a := (t - stops[i-1].at) / (stops[i].at - stops[i-1].at)
			return color.RGBA{
				R: uint8(stops[i-1].rgb[0]*(1-a) + stops[i].rgb[0]*a),
				G: uint8(stops[i-1].rgb[1]*(1-a) + stops[i].rgb[1]*a),
				B: uint8(stops[i-1].rgb[2]*(1-a) + stops[i].rgb[2]*a),
				A: 255,
			}
		}
	}
	return color.RGBA{R: 220, G: 38, B: 38, A: 255}
}
