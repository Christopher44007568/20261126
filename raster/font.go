package raster

import "strings"

// The bitmap font is a hand-authored 5x7 grid. Glyph data lives in
// font_glyphs.go / font_lower.go as readable ASCII art and is visually
// validated by TestFontPreview plus the rendered preview PNGs.

const (
	glyphW = 5
	glyphH = 7
)

var glyphSets = []map[rune][glyphH]string{glyphDigits, glyphUpper, glyphPunct, glyphLower}

func lookupGlyph(r rune) ([glyphH]string, bool) {
	for _, set := range glyphSets {
		if g, ok := set[r]; ok {
			return g, true
		}
	}
	return [glyphH]string{}, false
}

// TextWidth returns the pixel width of s at integer scale.
func TextWidth(s string, scale int) int {
	if scale < 1 {
		scale = 1
	}
	if len([]rune(s)) == 0 {
		return 0
	}
	return len([]rune(s)) * (glyphW + 1) * scale
}

// TextHeight returns the line height at scale.
func TextHeight(scale int) int { return (glyphH + 2) * scale }

// Text draws s with top-left at (x,y) and returns the x after the last glyph.
func (c *Canvas) Text(s string, x, y, scale int, col Color) int {
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, r := range s {
		g, ok := lookupGlyph(r)
		if r == ' ' {
			cx += (glyphW + 1) * scale
			continue
		}
		if !ok {
			g = missingGlyph
		}
		for row := 0; row < glyphH; row++ {
			line := g[row]
			for colI := 0; colI < len(line) && colI < glyphW; colI++ {
				if line[colI] == '#' {
					c.FillRect(cx+colI*scale, y+row*scale, scale, scale, col)
				}
			}
		}
		cx += (glyphW + 1) * scale
	}
	return cx
}

// TextRight draws s ending at rightX.
func (c *Canvas) TextRight(s string, rightX, y, scale int, col Color) {
	c.Text(s, rightX-TextWidth(s, scale), y, scale, col)
}

// Wrap performs word-aware greedy wrapping. Long tokens (notably paths) that do
// not fit on one line are hard-broken on rune boundaries. Explicit "\n" starts
// a new line.
func Wrap(s string, scale, maxWidth int) []string {
	if maxWidth <= 0 {
		return strings.Split(s, "\n")
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := ""
		for _, word := range words {
			for TextWidth(word, scale) > maxWidth {
				if line != "" {
					lines = append(lines, line)
					line = ""
				}
				word = emitHardChunk(word, scale, maxWidth, &lines)
			}
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if TextWidth(candidate, scale) > maxWidth && line != "" {
				lines = append(lines, line)
				line = word
			} else {
				line = candidate
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// emitHardChunk consumes the longest rune prefix of word fitting maxWidth,
// appends it as a line, and returns the remainder.
func emitHardChunk(word string, scale, maxWidth int, lines *[]string) string {
	cur := ""
	for i, r := range word {
		if TextWidth(cur+string(r), scale) > maxWidth && cur != "" {
			*lines = append(*lines, cur)
			return word[i:]
		}
		cur += string(r)
	}
	*lines = append(*lines, cur)
	return ""
}

// TextWrapped draws wrapped text and returns the total height used.
func (c *Canvas) TextWrapped(s string, x, y, scale, maxWidth int, col Color) int {
	lines := Wrap(s, scale, maxWidth)
	lh := TextHeight(scale)
	for i, line := range lines {
		c.Text(line, x, y+i*lh, scale, col)
	}
	return len(lines) * lh
}

var missingGlyph = [glyphH]string{
	"#####",
	"#...#",
	"#.?##",
	"#..##",
	"#.#.#",
	".....",
	"..#..",
}
