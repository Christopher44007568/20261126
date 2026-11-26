package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
)

type Canvas struct {
	W   int
	H   int
	Pix []byte
}

func NewCanvas(w, h int, bg Color3) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]byte, w*h*3)}
	c.Fill(bg)
	return c
}

func (c *Canvas) Fill(col Color3) {
	r, g, b := channel(col.R), channel(col.G), channel(col.B)
	for i := 0; i < len(c.Pix); i += 3 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = r, g, b
	}
}

func (c *Canvas) Set(x, y int, col Color3) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i] = channel(col.R)
	c.Pix[i+1] = channel(col.G)
	c.Pix[i+2] = channel(col.B)
}

func (c *Canvas) FillRect(x, y, w, h int, col Color3) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			c.Set(px, py, col)
		}
	}
}

func (c *Canvas) FillRoundedRect(x, y, w, h, radius int, col Color3) {
	if radius < 0 {
		radius = 0
	}
	r2 := radius * radius
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			dx, dy := 0, 0
			if px < x+radius {
				dx = x + radius - px
			} else if px >= x+w-radius {
				dx = px - (x + w - radius - 1)
			}
			if py < y+radius {
				dy = y + radius - py
			} else if py >= y+h-radius {
				dy = py - (y + h - radius - 1)
			}
			if dx == 0 || dy == 0 || dx*dx+dy*dy <= r2 {
				c.Set(px, py, col)
			}
		}
	}
}

func (c *Canvas) FillCircle(cx, cy, radius int, col Color3) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				c.Set(x, y, col)
			}
		}
	}
}

func (c *Canvas) Text(x, y int, text string, col Color3, scale int) int {
	if scale < 1 {
		scale = 1
	}
	cursor := x
	for _, ch := range strings.ToUpper(text) {
		if ch == '\n' {
			cursor = x
			y += 9 * scale
			continue
		}
		if ch > 127 {
			ch = '?'
		}
		glyph, ok := font5x7[byte(ch)]
		if !ok {
			glyph = font5x7['?']
		}
		for row := 0; row < 7; row++ {
			for colIndex := 0; colIndex < 5; colIndex++ {
				if glyph[row]&(1<<(4-colIndex)) == 0 {
					continue
				}
				c.FillRect(cursor+colIndex*scale, y+row*scale, scale, scale, col)
			}
		}
		cursor += 6 * scale
	}
	return cursor - x - scale
}

func (c *Canvas) TextRight(right, y int, text string, col Color3, scale int) {
	c.Text(right-c.TextWidth(text, scale), y, text, col, scale)
}

func (c *Canvas) TextWidth(text string, scale int) int {
	if len(text) == 0 {
		return 0
	}
	return len(text)*6*scale - scale
}

func (c *Canvas) PPM() []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "P6\n%d %d\n255\n", c.W, c.H)
	out.Write(c.Pix)
	return out.Bytes()
}

func (c *Canvas) PNG() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 3
			img.SetRGBA(x, y, color.RGBA{R: c.Pix[i], G: c.Pix[i+1], B: c.Pix[i+2], A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
