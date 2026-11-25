package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RGB struct {
	R, G, B byte
}

var (
	colInk      = RGB{232, 239, 248}
	colMuted    = RGB{128, 147, 176}
	colPanel    = RGB{16, 26, 48}
	colPanel2   = RGB{22, 35, 62}
	colGrid     = RGB{40, 59, 92}
	colGen1     = RGB{242, 126, 54}
	colGen2     = RGB{42, 195, 208}
	colGen3     = RGB{202, 92, 214}
	colGreen    = RGB{54, 210, 122}
	colYellow   = RGB{255, 215, 72}
	colRed      = RGB{248, 76, 92}
	colBg       = RGB{7, 12, 24}
	colBgFrame  = RGB{5, 9, 18}
	colBlue     = RGB{72, 132, 240}
	colDarkText = RGB{24, 32, 48}
)

type Canvas struct {
	W, H int
	Pix  []byte
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]byte, w*h*3)}
	c.Fill(bg)
	return c
}

func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
}

func (c *Canvas) Fill(col RGB) {
	for i := 0; i < len(c.Pix); i += 3 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
	}
}

func (c *Canvas) FillRect(x, y, w, h int, col RGB) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Set(xx, yy, col)
		}
	}
}

func (c *Canvas) DrawRect(x, y, w, h, thickness int, col RGB) {
	if thickness <= 0 {
		thickness = 1
	}
	c.FillRect(x, y, w, thickness, col)
	c.FillRect(x, y+h-thickness, w, thickness, col)
	c.FillRect(x, y, thickness, h, col)
	c.FillRect(x+w-thickness, y, thickness, h, col)
}

func (c *Canvas) BlendRect(x, y, w, h int, col RGB, alpha float64) {
	alpha = clamp01(alpha)
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx < 0 || yy < 0 || xx >= c.W || yy >= c.H {
				continue
			}
			i := (yy*c.W + xx) * 3
			c.Pix[i] = byte(float64(c.Pix[i])*(1-alpha) + float64(col.R)*alpha)
			c.Pix[i+1] = byte(float64(c.Pix[i+1])*(1-alpha) + float64(col.G)*alpha)
			c.Pix[i+2] = byte(float64(c.Pix[i+2])*(1-alpha) + float64(col.B)*alpha)
		}
	}
}

func (c *Canvas) Line(x0, y0, x1, y1, thickness int, col RGB) {
	if thickness <= 0 {
		thickness = 1
	}
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		c.FillRect(x0-thickness/2, y0-thickness/2, thickness, thickness, col)
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

func (c *Canvas) DashedLine(x0, y0, x1, y1, thickness, dash int, col RGB) {
	if dash <= 0 {
		dash = 6
	}
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	steps := int(1 + maxFloat(absFloat(dx), absFloat(dy)))
	for i := 0; i < steps; i++ {
		if (i/dash)%2 != 0 {
			continue
		}
		t := float64(i) / float64(steps)
		x := x0 + int(dx*t)
		y := y0 + int(dy*t)
		c.FillRect(x, y, thickness, thickness, col)
	}
}

func (c *Canvas) Circle(cx, cy, radius int, col RGB) {
	r2 := radius * radius
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= r2 {
				c.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (c *Canvas) Text(x, y, scale int, col RGB, text string) {
	if scale <= 0 {
		scale = 1
	}
	cursor := x
	for _, r := range strings.ToUpper(text) {
		glyph, ok := font5x7[byte(r)]
		if !ok {
			if r == ' ' {
				cursor += 6 * scale
				continue
			}
			glyph = font5x7['?']
		}
		for row := 0; row < 7; row++ {
			for colIndex := 0; colIndex < 5; colIndex++ {
				if glyph[row][colIndex] != '1' {
					continue
				}
				c.FillRect(cursor+colIndex*scale, y+row*scale, scale, scale, col)
			}
		}
		cursor += 6 * scale
	}
}

func textWidth(text string, scale int) int {
	return len(text) * 6 * scale
}

func writePPM(path string, c *Canvas) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	if _, err := w.Write(c.Pix); err != nil {
		return err
	}
	return w.Flush()
}

func writePixelsPPM(path string, width, height int, pix []byte) error {
	return writePPM(path, &Canvas{W: width, H: height, Pix: pix})
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

var font5x7 = map[byte][7]string{
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
	'.': {"00000", "00000", "00000", "00000", "00000", "01100", "01100"},
	':': {"00000", "01100", "01100", "00000", "01100", "01100", "00000"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'_': {"00000", "00000", "00000", "00000", "00000", "00000", "11111"},
	'/': {"00001", "00010", "00010", "00100", "01000", "01000", "10000"},
	'(': {"00010", "00100", "01000", "01000", "01000", "00100", "00010"},
	')': {"01000", "00100", "00010", "00010", "00010", "00100", "01000"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	'#': {"01010", "01010", "11111", "01010", "11111", "01010", "01010"},
	'<': {"00010", "00100", "01000", "10000", "01000", "00100", "00010"},
	'>': {"01000", "00100", "00010", "00001", "00010", "00100", "01000"},
	'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
	'%': {"11001", "11010", "00100", "01000", "10110", "00110", "00000"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"01110", "00100", "00100", "00100", "00100", "00100", "01110"},
	'J': {"00111", "00010", "00010", "00010", "00010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}
