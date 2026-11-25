package main

import "strings"

var font3x5 = map[byte][5]uint8{
	' ': {0, 0, 0, 0, 0},
	'0': {0b01110, 0b10001, 0b10011, 0b10101, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b01110},
	'2': {0b01110, 0b10001, 0b00110, 0b01000, 0b11111},
	'3': {0b11110, 0b00001, 0b01110, 0b00001, 0b11110},
	'4': {0b00010, 0b00110, 0b01010, 0b11111, 0b00010},
	'5': {0b11111, 0b10000, 0b11110, 0b00001, 0b11110},
	'6': {0b00110, 0b01000, 0b11110, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000},
	'8': {0b01110, 0b10001, 0b01110, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b01111, 0b00001, 0b01110},
	'A': {0b01110, 0b10001, 0b11111, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b11110, 0b10001, 0b11110},
	'C': {0b01111, 0b10000, 0b10000, 0b10000, 0b01111},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b11110, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b11110, 0b10000, 0b10000},
	'G': {0b01111, 0b10000, 0b10111, 0b10001, 0b01111},
	'H': {0b10001, 0b10001, 0b11111, 0b10001, 0b10001},
	'I': {0b11111, 0b00100, 0b00100, 0b00100, 0b11111},
	'J': {0b00111, 0b00010, 0b00010, 0b10010, 0b01100},
	'K': {0b10001, 0b10010, 0b11100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001},
	'O': {0b01110, 0b10001, 0b10001, 0b10001, 0b01110},
	'P': {0b11110, 0b10001, 0b11110, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b11110, 0b10010, 0b10001},
	'S': {0b01111, 0b10000, 0b01110, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b01111},
	'V': {0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10101, 0b11011, 0b10001},
	'X': {0b10001, 0b01010, 0b00100, 0b01010, 0b10001},
	'Y': {0b10001, 0b01010, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00010, 0b00100, 0b01000, 0b11111},
	'-': {0b00000, 0b00000, 0b01110, 0b00000, 0b00000},
	'.': {0b00000, 0b00000, 0b00000, 0b00000, 0b00100},
	':': {0b00000, 0b00100, 0b00000, 0b00100, 0b00000},
	'/': {0b00001, 0b00010, 0b00100, 0b01000, 0b10000},
}

func drawText(canvas *Canvas, x, y int, text string, color RGB8) {
	text = strings.ToUpper(text)
	for i := 0; i < len(text); i++ {
		glyph, ok := font3x5[text[i]]
		if !ok {
			glyph = font3x5[' ']
		}
		for row := 0; row < 5; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row]&(1<<(4-col)) != 0 {
					canvas.Set(x+i*4+col, y+row, color)
				}
			}
		}
	}
}

func drawLine(canvas *Canvas, x0, y0, x1, y1 int, color RGB8) {
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
		canvas.Set(x0, y0, color)
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

func drawRect(canvas *Canvas, x, y, width, height int, color RGB8) {
	drawLine(canvas, x, y, x+width-1, y, color)
	drawLine(canvas, x, y+height-1, x+width-1, y+height-1, color)
	drawLine(canvas, x, y, x, y+height-1, color)
	drawLine(canvas, x+width-1, y, x+width-1, y+height-1, color)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func heatColor(value, max float64) RGB8 {
	t := value / max
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	if t < 0.5 {
		level := t * 2
		return RGB8{
			R: uint8(20 + level*220),
			G: uint8(45 + level*170),
			B: uint8(90 * (1 - level)),
		}
	}
	level := (t - 0.5) * 2
	return RGB8{
		R: 240,
		G: uint8(215 * (1 - level)),
		B: uint8(90 * (1 - level)),
	}
}
