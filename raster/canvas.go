// Package raster is a tiny zero-dependency RGB canvas with the primitives the
// audit posters need. It writes binary PPM (P6) directly and can also encode a
// PNG preview through the Go standard library.
package raster

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
)

// Color is an 8-bit RGB color.
type Color struct{ R, G, B uint8 }

// Hex builds a color from 0xRRGGBB.
func Hex(v uint32) Color {
	return Color{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}
}

var (
	Red = Color{255, 0, 0}
)

// Canvas is a top-left-origin RGB pixel buffer.
type Canvas struct {
	W, H int
	Pix  []byte
}

// New creates a canvas filled with bg.
func New(w, h int, bg Color) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]byte, w*h*3)}
	c.Clear(bg)
	return c
}

// Clear fills the whole canvas.
func (c *Canvas) Clear(col Color) {
	for i := 0; i < len(c.Pix); i += 3 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
	}
}

func (c *Canvas) set(x, y int, col Color) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
}

// At returns the color at a clipped location (zero if outside).
func (c *Canvas) At(x, y int) Color {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return Color{}
	}
	i := (y*c.W + x) * 3
	return Color{c.Pix[i], c.Pix[i+1], c.Pix[i+2]}
}

// FillRect fills w×h pixels at (x,y); clipping is handled per row.
func (c *Canvas) FillRect(x, y, w, h int, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, y0 := max(x, 0), max(y, 0)
	x1, y1 := min(x+w, c.W), min(y+h, c.H)
	for yy := y0; yy < y1; yy++ {
		i := (yy*c.W + x0) * 3
		for xx := x0; xx < x1; xx++ {
			c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
			i += 3
		}
	}
}

// FrameRect strokes a t-pixel-thick rectangle.
func (c *Canvas) FrameRect(x, y, w, h, t int, col Color) {
	c.FillRect(x, y, w, t, col)     // top
	c.FillRect(x, y+h-t, w, t, col) // bottom
	c.FillRect(x, y, t, h, col)     // left
	c.FillRect(x+w-t, y, t, h, col) // right
}

// Line draws a Bresenham line.
func (c *Canvas) Line(x0, y0, x1, y1 int, col Color) {
	c.LineWidth(x0, y0, x1, y1, 1, col)
}

// LineWidth draws a line with a square brush of radius ceil(w/2).
func (c *Canvas) LineWidth(x0, y0, x1, y1, width int, col Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	r := width / 2
	for {
		c.FillRect(x0-r, y0-r, width, width, col)
		if x0 == x1 && y0 == y1 {
			break
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

// DashedLine draws width-thick dashes along a line approximation.
func (c *Canvas) DashedLine(x0, y0, x1, y1, dash, gap, width int, col Color) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	ux, uy := dx/length, dy/length
	pos := 0.0
	on := true
	segLen := float64(dash)
	for pos < length {
		step := segLen
		if pos+step > length {
			step = length - pos
		}
		if on {
			sx := x0 + int(ux*pos+0.5)
			sy := y0 + int(uy*pos+0.5)
			ex := x0 + int(ux*(pos+step)+0.5)
			ey := y0 + int(uy*(pos+step)+0.5)
			c.LineWidth(sx, sy, ex, ey, width, col)
		}
		pos += segLen
		if on {
			segLen = float64(gap)
			on = false
		} else {
			segLen = float64(dash)
			on = true
		}
	}
}

// Disc fills a circle.
func (c *Canvas) Disc(cx, cy, r int, col Color) {
	for yy := -r; yy <= r; yy++ {
		for xx := -r; xx <= r; xx++ {
			if xx*xx+yy*yy <= r*r {
				c.set(cx+xx, cy+yy, col)
			}
		}
	}
}

// Circle strokes a circle outline of the given thickness.
func (c *Canvas) Circle(cx, cy, r, thickness int, col Color) {
	for yy := -r - thickness; yy <= r+thickness; yy++ {
		for xx := -r - thickness; xx <= r+thickness; xx++ {
			d2 := xx*xx + yy*yy
			outer := r + thickness/2
			inner := r - thickness/2
			if d2 <= outer*outer && d2 >= inner*inner {
				c.set(cx+xx, cy+yy, col)
			}
		}
	}
}

// EncodePPM writes binary P6 PPM.
func (c *Canvas) EncodePPM(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	_, err := w.Write(c.Pix)
	return err
}

// WritePPMFile writes a binary P6 PPM file.
func (c *Canvas) WritePPMFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.EncodePPM(f)
}

// ToRGBA exposes the canvas as a standard library image for PNG preview.
func (c *Canvas) ToRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 3
			o := img.PixOffset(x, y)
			img.Pix[o+0] = c.Pix[i+0]
			img.Pix[o+1] = c.Pix[i+1]
			img.Pix[o+2] = c.Pix[i+2]
			img.Pix[o+3] = 255
		}
	}
	return img
}

// WritePNGFile writes a PNG preview (standard library image/png).
func (c *Canvas) WritePNGFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.ToRGBA())
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
