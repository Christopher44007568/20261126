package render

import (
	"image"
	"image/color"
	"image/png"
	"io"
)

// Canvas is a simple RGB drawing surface backed by image.RGBA.
type Canvas struct {
	Img *image.RGBA
}

func New(w, h int) *Canvas {
	return &Canvas{Img: image.NewRGBA(image.Rect(0, 0, w, h))}
}

func (c *Canvas) W() int { return c.Img.Rect.Dx() }
func (c *Canvas) H() int { return c.Img.Rect.Dy() }

func (c *Canvas) Set(x, y int, col color.Color) {
	if image.Pt(x, y).In(c.Img.Rect) {
		c.Img.Set(x, y, col)
	}
}

func (c *Canvas) Fill(col color.Color) {
	for y := 0; y < c.H(); y++ {
		for x := 0; x < c.W(); x++ {
			c.Img.Set(x, y, col)
		}
	}
}

// FillRect paints an axis-aligned rectangle.
func (c *Canvas) FillRect(x0, y0, x1, y1 int, col color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.Set(x, y, col)
		}
	}
}

// RectStroke draws a one-pixel rectangle outline.
func (c *Canvas) RectStroke(x0, y0, x1, y1 int, col color.Color) {
	for x := x0; x <= x1; x++ {
		c.Set(x, y0, col)
		c.Set(x, y1, col)
	}
	for y := y0; y <= y1; y++ {
		c.Set(x0, y, col)
		c.Set(x1, y, col)
	}
}

// Line draws a line with the DDA algorithm.
func (c *Canvas) Line(x0, y0, x1, y1 int, col color.Color) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Set(x0, y0, col)
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

// Hatch draws diagonal magenta hatch lines over a hole mask region.
func (c *Canvas) Hatch(mask func(x, y int) bool, base color.Color) {
	w, h := c.W(), c.H()
	magenta := color.RGBA{R: 235, G: 24, B: 138, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask(x, y) {
				continue
			}
			if (x+y)%8 < 3 {
				c.Set(x, y, magenta)
			} else {
				c.Set(x, y, base)
			}
		}
	}
}

// WritePNG encodes the canvas.
func (c *Canvas) WritePNG(w io.Writer) error { return png.Encode(w, c.Img) }

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
