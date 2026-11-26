package main

import (
	"fmt"
	"strings"
)

const (
	fontGlyphW  = 5
	fontGlyphH  = 7
	fontAdvance = 6
)

// Hex parses a #RRGGBB colour (alpha defaults to 255).
func Hex(s string) colorRGBA {
	var r, g, b int
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return colorRGBA{R: 255, G: 0, B: 255, A: 255}
	}
	return colorRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func HexA(s string, a uint8) colorRGBA {
	c := Hex(s)
	c.A = a
	return c
}

func glyphFor(r rune) Glyph {
	if g, ok := fontGlyphs[r]; ok {
		return g
	}
	if g, ok := fontGlyphs['?']; ok {
		return g
	}
	return Glyph{}
}

// DrawText renders s with the bitmap font. Coordinates are top-left.
func (c *Canvas) DrawText(s string, x, top, scale int, col colorRGBA, tracking int) int {
	s = strings.ToUpper(s)
	pen := x
	step := fontAdvance*scale + tracking
	for _, r := range s {
		g := glyphFor(r)
		for row := 0; row < fontGlyphH; row++ {
			line := g[row]
			for colI := 0; colI < len(line) && colI < fontGlyphW; colI++ {
				if line[colI] != '#' {
					continue
				}
				c.FillRect(pen+colI*scale, top+row*scale, scale, scale, col)
			}
		}
		pen += step
	}
	return pen
}

// Measure returns the advance width of s at the given scale/tracking.
func Measure(s string, scale, tracking int) int {
	n := len([]rune(strings.ToUpper(s)))
	if n == 0 {
		return 0
	}
	return n*fontAdvance*scale + (n-1)*tracking
}

// Ellipsize shortens s so its width fits maxW, adding "..." when truncated.
func Ellipsize(s string, scale, tracking, maxW int) string {
	s = strings.ToUpper(s)
	if Measure(s, scale, tracking) <= maxW {
		return s
	}
	ell := "..."
	ellW := Measure(ell, scale, tracking)
	runes := []rune(s)
	for n := len(runes); n > 0; n-- {
		cand := string(runes[:n])
		if Measure(cand, scale, tracking)+ellW <= maxW {
			return cand + ell
		}
	}
	return ell
}

// DrawTextFit draws s truncated with an ellipsis to never exceed maxW.
func (c *Canvas) DrawTextFit(s string, x, top, scale int, col colorRGBA, tracking, maxW int) int {
	return c.DrawText(Ellipsize(s, scale, tracking, maxW), x, top, scale, col, tracking)
}

// DrawTextRight draws s ending at rightX.
func (c *Canvas) DrawTextRight(s string, rightX, top, scale int, col colorRGBA, tracking int) int {
	w := Measure(s, scale, tracking)
	return c.DrawText(s, rightX-w, top, scale, col, tracking)
}
