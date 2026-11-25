package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
)

type Canvas struct {
	Image *image.RGBA
}

func NewCanvas(width, height int) Canvas {
	return Canvas{Image: image.NewRGBA(image.Rect(0, 0, width, height))}
}

func (c Canvas) Set(x, y int, col color.Color) {
	if image.Pt(x, y).In(c.Image.Rect) {
		c.Image.Set(x, y, col)
	}
}

func (c Canvas) Fill(col color.Color) {
	bounds := c.Image.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c.Image.Set(x, y, col)
		}
	}
}

func (c Canvas) FillRect(x, y, width, height int, col color.Color) {
	bounds := c.Image.Bounds()
	x0 := max(bounds.Min.X, x)
	y0 := max(bounds.Min.Y, y)
	x1 := min(bounds.Max.X, x+width)
	y1 := min(bounds.Max.Y, y+height)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			c.Image.Set(px, py, col)
		}
	}
}

func (c Canvas) DrawRect(x, y, width, height int, col color.Color) {
	c.FillRect(x, y, width, 1, col)
	c.FillRect(x, y+height-1, width, 1, col)
	c.FillRect(x, y, 1, height, col)
	c.FillRect(x+width-1, y, 1, height, col)
}

func (c Canvas) DrawLine(x0, y0, x1, y1 int, col color.Color) {
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
	errorValue := dx + dy
	for {
		c.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		error2 := 2 * errorValue
		if error2 >= dy {
			errorValue += dy
			x0 += sx
		}
		if error2 <= dx {
			errorValue += dx
			y0 += sy
		}
	}
}

func (c Canvas) DrawHatchedRect(x, y, width, height int, base, stripe color.Color) {
	c.FillRect(x, y, width, height, base)
	for offset := -height; offset < width+height; offset += 12 {
		x0, y0, x1, y1, ok := clipLine(x+offset, y+height, x+offset+height, y, x, y, x+width-1, y+height-1)
		if !ok {
			continue
		}
		c.DrawLine(x0, y0, x1, y1, stripe)
	}
	c.DrawRect(x, y, width, height, stripe)
}

func clipLine(x0, y0, x1, y1, minX, minY, maxX, maxY int) (int, int, int, int, bool) {
	const (
		left   = 1
		right  = 2
		bottom = 4
		top    = 8
	)
	code := func(x, y int) int {
		value := 0
		if x < minX {
			value |= left
		}
		if x > maxX {
			value |= right
		}
		if y < minY {
			value |= top
		}
		if y > maxY {
			value |= bottom
		}
		return value
	}

	code0 := code(x0, y0)
	code1 := code(x1, y1)
	for {
		if code0|code1 == 0 {
			return x0, y0, x1, y1, true
		}
		if code0&code1 != 0 {
			return 0, 0, 0, 0, false
		}

		outside := code0
		if outside == 0 {
			outside = code1
		}
		x, y := 0, 0
		if outside&top != 0 {
			x = x0 + (x1-x0)*(minY-y0)/(y1-y0)
			y = minY
		} else if outside&bottom != 0 {
			x = x0 + (x1-x0)*(maxY-y0)/(y1-y0)
			y = maxY
		} else if outside&right != 0 {
			y = y0 + (y1-y0)*(maxX-x0)/(x1-x0)
			x = maxX
		} else if outside&left != 0 {
			y = y0 + (y1-y0)*(minX-x0)/(x1-x0)
			x = minX
		}

		if outside == code0 {
			x0, y0, code0 = x, y, code(x, y)
		} else {
			x1, y1, code1 = x, y, code(x, y)
		}
	}
}

func (c Canvas) DrawText(x, y int, text string, scale int, col color.Color) {
	cursor := x
	for _, char := range strings.ToUpper(text) {
		glyph, ok := glyphs[char]
		if !ok {
			glyph = glyphs[' ']
		}
		for row, bits := range glyph {
			for column, bit := range bits {
				if bit == '1' {
					c.FillRect(cursor+column*scale, y+row*scale, scale, scale, col)
				}
			}
		}
		cursor += 6 * scale
	}
}

func (c Canvas) TextWidth(text string, scale int) int {
	return len(text) * 6 * scale
}

func (c Canvas) DrawTextCentered(x, y int, text string, scale int, col color.Color) {
	c.DrawText(x-c.TextWidth(text, scale)/2, y, text, scale, col)
}

func (c Canvas) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, c.Image)
}

func colorFromRGB(value RGB) color.RGBA {
	return color.RGBA{
		R: colorByte(value.R),
		G: colorByte(value.G),
		B: colorByte(value.B),
		A: 255,
	}
}

func colorByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}

func blend(dst, src color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{
		R: colorByte(float64(dst.R)/255*(1-alpha) + float64(src.R)/255*alpha),
		G: colorByte(float64(dst.G)/255*(1-alpha) + float64(src.G)/255*alpha),
		B: colorByte(float64(dst.B)/255*(1-alpha) + float64(src.B)/255*alpha),
		A: 255,
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
