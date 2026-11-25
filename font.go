package main

import (
	"image"
	"image/color"
	"strings"
)

var glyphs = buildGlyphs()

func buildGlyphs() map[rune][]string {
	out := make(map[rune][]string)
	for key, rows := range letterGlyphs {
		out[key] = rows
	}
	for key, rows := range numberGlyphs {
		out[key] = rows
	}
	return out
}

func textWidth(text string, scale int) int {
	if text == "" {
		return 0
	}
	advance := 6 * scale
	return len([]rune(text))*advance - scale
}

func drawText(dst *image.RGBA, x, y int, text string, ink color.RGBA, scale int) int {
	if scale < 1 {
		scale = 1
	}
	cursor := x
	for _, r := range strings.ToUpper(text) {
		rows, ok := glyphs[r]
		if !ok {
			rows = glyphs['?']
		}
		if r != ' ' {
			drawGlyph(dst, cursor, y, rows, ink, scale)
		}
		cursor += 6 * scale
	}
	return cursor - x
}

func drawGlyph(dst *image.RGBA, x, y int, rows []string, ink color.RGBA, scale int) {
	b := dst.Bounds()
	for gy, row := range rows {
		for gx, bit := range row {
			if bit != '1' {
				continue
			}
			px := x + gx*scale
			py := y + gy*scale
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					xx, yy := px+sx, py+sy
					if image.Pt(xx, yy).In(b) {
						dst.SetRGBA(xx, yy, ink)
					}
				}
			}
		}
	}
}

func drawTextRight(dst *image.RGBA, right, y int, text string, ink color.RGBA, scale int) {
	x := right - textWidth(text, scale)
	if x < dst.Bounds().Min.X {
		x = dst.Bounds().Min.X
	}
	drawText(dst, x, y, text, ink, scale)
}

func drawTextCentered(dst *image.RGBA, left, right, y int, text string, ink color.RGBA, scale int) {
	x := left + (right-left-textWidth(text, scale))/2
	if x < left {
		x = left
	}
	drawText(dst, x, y, text, ink, scale)
}

func fillRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(dst.Bounds())
	drawRect(dst, r, c)
}

func drawRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

func drawLine(dst *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(dst.Bounds()) {
			dst.SetRGBA(x0, y0, c)
		}
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

func strokeRect(dst *image.RGBA, r image.Rectangle, c color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	for i := 0; i < thickness; i++ {
		rr := r.Inset(-i)
		drawLine(dst, rr.Min.X, rr.Min.Y, rr.Max.X-1, rr.Min.Y, c)
		drawLine(dst, rr.Max.X-1, rr.Min.Y, rr.Max.X-1, rr.Max.Y-1, c)
		drawLine(dst, rr.Max.X-1, rr.Max.Y-1, rr.Min.X, rr.Max.Y-1, c)
		drawLine(dst, rr.Min.X, rr.Max.Y-1, rr.Min.X, rr.Min.Y, c)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
