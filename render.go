package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// ssFactor is the supersampling factor used for anti-aliasing.
const ssFactor = 4

// canvas is a straight-alpha pixel buffer.
type canvas struct {
	w, h int
	pix  []color.NRGBA
}

func newCanvas(w, h int) *canvas {
	return &canvas{w: w, h: h, pix: make([]color.NRGBA, w*h)}
}

func (c *canvas) fillAll(col color.NRGBA) {
	for i := range c.pix {
		c.pix[i] = col
	}
}

// blendPixel composites src over the destination (src-over, straight alpha).
func (c *canvas) blendPixel(x, y int, src color.NRGBA) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h || src.A == 0 {
		return
	}
	i := y*c.w + x
	dst := c.pix[i]
	sa := float64(src.A) / 255
	da := float64(dst.A) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		c.pix[i] = color.NRGBA{}
		return
	}
	blend := func(sc, dc uint8) uint8 {
		v := (float64(sc)*sa + float64(dc)*da*(1-sa)) / outA
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v + 0.5)
	}
	c.pix[i] = color.NRGBA{
		R: blend(src.R, dst.R),
		G: blend(src.G, dst.G),
		B: blend(src.B, dst.B),
		A: uint8(outA*255 + 0.5),
	}
}

// fillPolygon fills a polygon (non-zero winding) with a solid color.
func (c *canvas) fillPolygon(pts []Point, col color.NRGBA) {
	if len(pts) < 3 || col.A == 0 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	y0 := int(minY)
	y1 := int(maxY) + 1
	if y0 < 0 {
		y0 = 0
	}
	if y1 > c.h {
		y1 = c.h
	}
	var xs []float64
	for y := y0; y < y1; y++ {
		yc := float64(y) + 0.5
		xs = xs[:0]
		for i, p := range pts {
			q := pts[(i+1)%len(pts)]
			if (p.Y <= yc && q.Y > yc) || (q.Y <= yc && p.Y > yc) {
				t := (yc - p.Y) / (q.Y - p.Y)
				xs = append(xs, p.X+t*(q.X-p.X))
			}
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			xa := int(xs[k] + 0.5)
			xb := int(xs[k+1] + 0.5)
			if xa < 0 {
				xa = 0
			}
			if xb > c.w {
				xb = c.w
			}
			for x := xa; x < xb; x++ {
				c.blendPixel(x, y, col)
			}
		}
	}
}

// strokePolyline strokes each segment of a closed polyline by filling
// a quad of the given width around it (butt caps, miter-free joins).
func (c *canvas) strokePolyline(pts []Point, width float64, col color.NRGBA) {
	if len(pts) < 2 || width <= 0 || col.A == 0 {
		return
	}
	half := width / 2
	n := len(pts)
	for i := 0; i < n; i++ {
		p := pts[i]
		q := pts[(i+1)%n]
		dx, dy := q.X-p.X, q.Y-p.Y
		l := math.Hypot(dx, dy)
		if l == 0 {
			continue
		}
		nx, ny := -dy/l*half, dx/l*half
		quad := []Point{
			{X: p.X + nx, Y: p.Y + ny},
			{X: q.X + nx, Y: q.Y + ny},
			{X: q.X - nx, Y: q.Y - ny},
			{X: p.X - nx, Y: p.Y - ny},
		}
		c.fillPolygon(quad, col)
	}
}

// downsample averages ss x ss blocks into the final image.
func downsample(src *canvas, ss int) *image.NRGBA {
	w, h := src.w/ss, src.h/ss
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	n := ss * ss
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					p := src.pix[(y*ss+dy)*src.w+(x*ss+dx)]
					r += uint32(p.R)
					g += uint32(p.G)
					b += uint32(p.B)
					a += uint32(p.A)
				}
			}
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / uint32(n)),
				G: uint8(g / uint32(n)),
				B: uint8(b / uint32(n)),
				A: uint8(a / uint32(n)),
			})
		}
	}
	return out
}
