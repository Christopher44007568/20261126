package main

import (
	"image"
	"image/color"
	"math"
)

var (
	colInk       = color.RGBA{R: 7, G: 13, B: 24, A: 255}
	colPanel     = color.RGBA{R: 16, G: 27, B: 45, A: 238}
	colPanelEdge = color.RGBA{R: 54, G: 78, B: 108, A: 255}
	colWhite     = color.RGBA{R: 235, G: 242, B: 250, A: 255}
	colMuted     = color.RGBA{R: 139, G: 158, B: 180, A: 255}
	colCyan      = color.RGBA{R: 45, G: 212, B: 232, A: 255}
	colAmber     = color.RGBA{R: 255, G: 178, B: 58, A: 255}
	colMagenta   = color.RGBA{R: 255, G: 72, B: 151, A: 255}
	colGreen     = color.RGBA{R: 77, G: 224, B: 142, A: 255}
)

func blendPixel(dst *image.RGBA, x, y int, src color.RGBA) {
	if !image.Pt(x, y).In(dst.Bounds()) {
		return
	}
	if src.A == 0 {
		return
	}
	off := dst.PixOffset(x, y)
	if src.A == 255 {
		dst.Pix[off+0] = src.R
		dst.Pix[off+1] = src.G
		dst.Pix[off+2] = src.B
		dst.Pix[off+3] = 255
		return
	}

	sa := uint32(src.A)
	da := uint32(dst.Pix[off+3])
	inv := 255 - sa
	outA := sa*255 + da*inv
	if outA == 0 {
		return
	}
	blendChannel := func(sc, dc uint8) uint8 {
		v := uint32(sc)*sa*255 + uint32(dc)*da*inv
		return uint8((v + outA/2) / outA)
	}
	dst.Pix[off+0] = blendChannel(src.R, dst.Pix[off+0])
	dst.Pix[off+1] = blendChannel(src.G, dst.Pix[off+1])
	dst.Pix[off+2] = blendChannel(src.B, dst.Pix[off+2])
	dst.Pix[off+3] = uint8((outA + 127) / 255)
}

func fillRect(dst *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			blendPixel(dst, px, py, c)
		}
	}
}

func drawRect(dst *image.RGBA, x, y, w, h, thickness int, c color.RGBA) {
	if thickness < 1 {
		thickness = 1
	}
	fillRect(dst, x, y, w, thickness, c)
	fillRect(dst, x, y+h-thickness, w, thickness, c)
	fillRect(dst, x, y, thickness, h, c)
	fillRect(dst, x+w-thickness, y, thickness, h, c)
}

func drawLine(dst *image.RGBA, a, b Vec2, thickness int, c color.RGBA) {
	dx, dy := b.X-a.X, b.Y-a.Y
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(a.X + dx*t))
		y := int(math.Round(a.Y + dy*t))
		fillRect(dst, x-thickness/2, y-thickness/2, thickness, thickness, c)
	}
}

func drawDashedLine(dst *image.RGBA, a, b Vec2, thickness int, dash, gap float64, c color.RGBA) {
	delta := b.sub(a)
	length := delta.length()
	if length < 1 {
		return
	}
	dir := delta.scale(1 / length)
	for start := 0.0; start < length; start += dash + gap {
		end := math.Min(start+dash, length)
		drawLine(dst, a.add(dir.scale(start)), a.add(dir.scale(end)), thickness, c)
	}
}

func drawArrow(dst *image.RGBA, from, to Vec2, thickness int, c color.RGBA) {
	drawLine(dst, from, to, thickness, c)
	delta := to.sub(from)
	length := delta.length()
	if length < 1 {
		return
	}
	dir := delta.scale(1 / length)
	perp := Vec2{X: -dir.Y, Y: dir.X}
	head := math.Min(18, length*0.32)
	base := to.sub(dir.scale(head))
	drawLine(dst, to, base.add(perp.scale(head*0.55)), thickness, c)
	drawLine(dst, to, base.sub(perp.scale(head*0.55)), thickness, c)
}

func fillCircle(dst *image.RGBA, cx, cy, radius int, c color.RGBA) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				blendPixel(dst, x, y, c)
			}
		}
	}
}

func drawCircle(dst *image.RGBA, cx, cy, radius, thickness int, c color.RGBA) {
	outer := radius * radius
	inner := (radius - thickness) * (radius - thickness)
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			d := dx*dx + dy*dy
			if d <= outer && d >= inner {
				blendPixel(dst, x, y, c)
			}
		}
	}
}

func scaleColor(c color.RGBA, factor float64) color.RGBA {
	clamp := func(v float64) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 255 {
			return 255
		}
		return uint8(v + 0.5)
	}
	return color.RGBA{
		R: clamp(float64(c.R) * factor),
		G: clamp(float64(c.G) * factor),
		B: clamp(float64(c.B) * factor),
		A: c.A,
	}
}

func mixColor(a, b color.RGBA, t float64) color.RGBA {
	clamp := func(v float64) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 255 {
			return 255
		}
		return uint8(v + 0.5)
	}
	return color.RGBA{
		R: clamp(float64(a.R)*(1-t) + float64(b.R)*t),
		G: clamp(float64(a.G)*(1-t) + float64(b.G)*t),
		B: clamp(float64(a.B)*(1-t) + float64(b.B)*t),
		A: clamp(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}

func dimImage(dst *image.RGBA, factor float64) {
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i+0] = uint8(float64(dst.Pix[i+0]) * factor)
		dst.Pix[i+1] = uint8(float64(dst.Pix[i+1]) * factor)
		dst.Pix[i+2] = uint8(float64(dst.Pix[i+2]) * factor)
	}
}
