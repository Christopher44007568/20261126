package main

import "unicode"

var tinyFont = map[rune][5]string{
	'A': {".X.", "X.X", "XXX", "X.X", "X.X"},
	'B': {"XX.", "X.X", "XX.", "X.X", "XX."},
	'C': {".XX", "X..", "X..", "X..", ".XX"},
	'D': {"XX.", "X.X", "X.X", "X.X", "XX."},
	'E': {"XXX", "X..", "XX.", "X..", "XXX"},
	'F': {"XXX", "X..", "XX.", "X..", "X.."},
	'G': {".XX", "X..", "X.X", "X.X", ".XX"},
	'H': {"X.X", "X.X", "XXX", "X.X", "X.X"},
	'I': {"XXX", ".X.", ".X.", ".X.", "XXX"},
	'J': {"..X", "..X", "..X", "X.X", ".X."},
	'K': {"X.X", "XX.", "X..", "XX.", "X.X"},
	'L': {"X..", "X..", "X..", "X..", "XXX"},
	'M': {"X.X", "XXX", "X.X", "X.X", "X.X"},
	'N': {"X.X", "XXX", "X.X", "X.X", "X.X"},
	'O': {".XX", "X.X", "X.X", "X.X", ".XX"},
	'P': {"XX.", "X.X", "XX.", "X..", "X.."},
	'Q': {".XX", "X.X", "X.X", ".XX", "..X"},
	'R': {"XX.", "X.X", "XX.", "X.X", "X.X"},
	'S': {"XXX", "X..", ".XX", "..X", "XXX"},
	'T': {"XXX", ".X.", ".X.", ".X.", ".X."},
	'U': {"X.X", "X.X", "X.X", "X.X", "XXX"},
	'V': {"X.X", "X.X", "X.X", "X.X", ".X."},
	'W': {"X.X", "X.X", "X.X", "XXX", "X.X"},
	'X': {"X.X", "X.X", ".X.", "X.X", "X.X"},
	'Y': {"X.X", "X.X", ".X.", ".X.", ".X."},
	'Z': {"XXX", "..X", ".X.", "X..", "XXX"},
	'0': {".XX", "X.X", "X.X", "X.X", ".XX"},
	'1': {".X.", "XX.", ".X.", ".X.", "XXX"},
	'2': {"XX.", "..X", ".X.", "X..", "XXX"},
	'3': {"XX.", "..X", ".XX", "..X", "XX."},
	'4': {"X.X", "X.X", "XXX", "..X", "..X"},
	'5': {"XXX", "X..", "XXX", "..X", "XXX"},
	'6': {".XX", "X..", "XXX", "X.X", ".XX"},
	'7': {"XXX", "..X", ".X.", ".X.", ".X."},
	'8': {".XX", "X.X", ".X.", "X.X", ".XX"},
	'9': {".XX", "X.X", ".XX", "..X", ".XX"},
	' ': {"...", "...", "...", "...", "..."},
	':': {"...", ".X.", "...", ".X.", "..."},
	'/': {"..X", "..X", ".X.", "X..", "X.."},
	'-': {"...", "...", "XXX", "...", "..."},
	'.': {"...", "...", "...", "...", ".X."},
	'(': {".X.", "X..", "X..", "X..", ".X."},
	')': {".X.", "..X", "..X", "..X", ".X."},
	'!': {".X.", ".X.", ".X.", "...", ".X."},
	'?': {"XX.", "..X", ".X.", "...", ".X."},
}

func TextWidth(text string, scale int) int {
	return len(text) * 4 * scale
}

func (c *Canvas) DrawText(x, y, scale int, text string, color Vec3) {
	cursor := x
	for _, r := range text {
		r = unicode.ToUpper(r)
		glyph, ok := tinyFont[r]
		if !ok {
			glyph = tinyFont['?']
		}
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if glyph[row][col] == 'X' {
					c.FillRect(cursor+col*scale, y+row*scale, scale, scale, color, 1)
				}
			}
		}
		cursor += 4 * scale
	}
}
