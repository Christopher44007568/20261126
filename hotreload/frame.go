package hotreload

import (
	"image"
	"image/png"
	"io"
)

// Frame is a tightly packed RGB buffer shared by every render node.
type Frame struct {
	W, H int
	Pix  []byte
}

func NewFrame(w, h int) *Frame {
	return &Frame{W: w, H: h, Pix: make([]byte, w*h*3)}
}

func (f *Frame) index(x, y int) int {
	return (y*f.W + x) * 3
}

func (f *Frame) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < f.W && y < f.H
}

func (f *Frame) Set(x, y int, c RGB) {
	if !f.In(x, y) {
		return
	}
	i := f.index(x, y)
	f.Pix[i], f.Pix[i+1], f.Pix[i+2] = c.R, c.G, c.B
}

func (f *Frame) Get(x, y int) RGB {
	i := f.index(x, y)
	return RGB{f.Pix[i], f.Pix[i+1], f.Pix[i+2]}
}

func (f *Frame) Fill(c RGB) {
	for i := 0; i < len(f.Pix); i += 3 {
		f.Pix[i], f.Pix[i+1], f.Pix[i+2] = c.R, c.G, c.B
	}
}

func (f *Frame) FillRect(x0, y0, x1, y1 int, c RGB) {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > f.W {
		x1 = f.W
	}
	if y1 > f.H {
		y1 = f.H
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			f.Set(x, y, c)
		}
	}
}

func (f *Frame) Rect(x0, y0, x1, y1, thickness int, c RGB) {
	f.FillRect(x0, y0, x1, y0+thickness, c)
	f.FillRect(x0, y1-thickness, x1, y1, c)
	f.FillRect(x0, y0, x0+thickness, y1, c)
	f.FillRect(x1-thickness, y0, x1, y1, c)
}

// DashedRect draws a dashed stroke by skipping pixels on a fixed cadence.
func (f *Frame) DashedRect(x0, y0, x1, y1, thickness, dash, gap int, c RGB) {
	for d := 0; d < thickness; d++ {
		for x := x0; x < x1; x++ {
			if (x-x0)%(dash+gap) < dash {
				f.Set(x, y0+d, c)
				f.Set(x, y1-1-d, c)
			}
		}
		for y := y0; y < y1; y++ {
			if (y-y0)%(dash+gap) < dash {
				f.Set(x0+d, y, c)
				f.Set(x1-1-d, y, c)
			}
		}
	}
}

func (f *Frame) Line(x0, y0, x1, y1 int, c RGB) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		f.Set(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// FillTriangle rasterizes a triangle with per-scanline spans.
func (f *Frame) FillTriangle(a, b, c Pt, fill func(x, y int)) {
	pts := []Pt{a, b, c}
	for i := 1; i < 3; i++ {
		for j := i; j > 0 && pts[j-1].Y > pts[j].Y; j-- {
			pts[j-1], pts[j] = pts[j], pts[j-1]
		}
	}
	top, mid, bot := pts[0], pts[1], pts[2]
	denom := float64((bot.Y - top.Y))
	if denom == 0 {
		return
	}
	for y := top.Y; y <= bot.Y; y++ {
		t := float64(y-top.Y) / denom
		lx := float64(top.X) + t*float64(bot.X-top.X)
		var rx float64
		if y <= mid.Y && mid.Y != top.Y {
			rx = float64(top.X) + float64(y-top.Y)/float64(mid.Y-top.Y)*float64(mid.X-top.X)
		} else if bot.Y != mid.Y {
			rx = float64(mid.X) + float64(y-mid.Y)/float64(bot.Y-mid.Y)*float64(bot.X-mid.X)
		} else {
			rx = lx
		}
		if lx > rx {
			lx, rx = rx, lx
		}
		for x := int(lx); x <= int(rx+0.999); x++ {
			fill(x, y)
		}
	}
}

func (f *Frame) StrokeTriangle(a, b, c Pt, col RGB) {
	f.Line(a.X, a.Y, b.X, b.Y, col)
	f.Line(b.X, b.Y, c.X, c.Y, col)
	f.Line(c.X, c.Y, a.X, a.Y, col)
}

func (f *Frame) Copy(src *Frame, ox, oy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			f.Set(ox+x, oy+y, src.Get(x, y))
		}
	}
}

// EncodePNG writes the frame as a standard RGB PNG without external deps.
func (f *Frame) EncodePNG(w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	for i := 0; i < f.W*f.H; i++ {
		j := i * 3
		k := i * 4
		img.Pix[k], img.Pix[k+1], img.Pix[k+2] = f.Pix[j], f.Pix[j+1], f.Pix[j+2]
		img.Pix[k+3] = 255
	}
	return png.Encode(w, img)
}

// RGB is the only color model used by render nodes.
type RGB struct {
	R, G, B uint8
}

type Pt struct {
	X, Y int
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
