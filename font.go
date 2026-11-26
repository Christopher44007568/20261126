package main

import (
	"image"
	"image/color"
)

const (
	glyphWidth  = 5
	glyphHeight = 7
)

var glyphs = map[rune][glyphHeight]string{}

func init() {
	for r, rows := range glyphsAM {
		glyphs[r] = rows
	}
	for r, rows := range glyphsNZ {
		glyphs[r] = rows
	}
	for r, rows := range glyphsDigits {
		glyphs[r] = rows
	}
}

func textWidth(text string, scale int) int {
	if text == "" {
		return 0
	}
	return (len([]rune(text))*glyphWidth + (len([]rune(text)) - 1)) * scale
}

func drawText(dst *image.RGBA, x, y int, text string, scale int, c color.RGBA) {
	if scale <= 0 {
		scale = 1
	}
	pen := x
	for _, r := range text {
		rows, ok := glyphs[r]
		if r == ' ' || !ok {
			pen += (glyphWidth + 1) * scale
			continue
		}
		for gy, row := range rows {
			for gx := 0; gx < glyphWidth && gx < len(row); gx++ {
				if row[gx] != '1' {
					continue
				}
				fillRect(dst, pen+gx*scale, y+gy*scale, scale, scale, c)
			}
		}
		pen += (glyphWidth + 1) * scale
	}
}

func drawTextCentered(dst *image.RGBA, cx, y int, text string, scale int, c color.RGBA) {
	drawText(dst, cx-textWidth(text, scale)/2, y, text, scale, c)
}
