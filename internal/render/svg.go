// Package render turns lab documents and audit results into self-contained
// SVG diagrams using only the Go standard library.
package render

import (
	"fmt"
	"io"
	"strings"
)

const fontStack = "Inter, ui-sans-serif, system-ui, -apple-system, 'Noto Sans CJK SC', sans-serif"

// Palette colors shared by every diagram.
const (
	Ink       = "#0F172A"
	SubInk    = "#475569"
	Line      = "#CBD5E1"
	Panel     = "#FFFFFF"
	PageBg    = "#EEF2FF"
	PanelBg   = "#F8FAFC"
	ErrorPink = "#FF2E93"
	ErrorBg   = "#FFE3EE"
	GoodGreen = "#16A34A"
	Amber     = "#F59E0B"
)

// SVG is a minimal streaming SVG writer.
type SVG struct {
	body   strings.Builder
	w      io.Writer
	width  int
	height int
	title  string
	desc   string
}

// NewSVG starts an SVG document.
func NewSVG(w io.Writer, width, height int, title, desc string) *SVG {
	return &SVG{w: w, width: width, height: height, title: title, desc: desc}
}

// Close writes the full document to the underlying writer.
func (s *SVG) Close() error {
	header := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>`+
			`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="%s">`+
			"<title>%s</title><desc>%s</desc>",
		s.width, s.height, s.width, s.height, fontStack,
		xmlEscape(s.title), xmlEscape(s.desc))
	_, err := io.WriteString(s.w, header+
		`<defs>`+
		`<pattern id="error-stripes" width="10" height="10" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">`+
		`<rect width="10" height="10" fill="`+ErrorBg+`"/>`+
		`<line x1="0" y1="0" x2="0" y2="10" stroke="`+ErrorPink+`" stroke-width="3" opacity="0.55"/>`+
		`</pattern>`+
		`<pattern id="hidden-grid" width="8" height="8" patternUnits="userSpaceOnUse">`+
		`<rect width="8" height="8" fill="`+PanelBg+`"/>`+
		`<path d="M0 8 L8 0" stroke="`+Line+`" stroke-width="1"/>`+
		`</pattern>`+
		`</defs>`+
		`<rect width="100%" height="100%" fill="`+PageBg+`"/>`+
		s.body.String()+
		`</svg>`)
	return err
}

// Panel draws a rounded white panel.
func (s *SVG) Panel(x, y, w, h float64, title string) {
	s.f(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="18" fill="%s" stroke="#DBE4F0"/>`, x, y, w, h, Panel)
	if title != "" {
		s.Text(x+18, y+28, title, 16, Ink, 800)
	}
}

// Header draws the page title and subtitle.
func (s *SVG) Header(title, subtitle string) {
	s.Text(48, 58, title, 28, Ink, 800)
	s.wrapText(48, 88, subtitle, 13, SubInk, 1180)
}

// Text writes a single text element.
func (s *SVG) Text(x, y float64, text string, size float64, fill string, weight int) {
	s.f(`<text x="%.1f" y="%.1f" font-size="%.1f" fill="%s" font-weight="%d">%s</text>`,
		x, y, size, fill, weight, xmlEscape(text))
}

// Rect draws a rectangle.
func (s *SVG) Rect(x, y, w, h float64, fill string, stroke string, strokeWidth float64, rx float64) {
	strokeAttr := ""
	if stroke != "" {
		strokeAttr = fmt.Sprintf(` stroke="%s" stroke-width="%.1f"`, stroke, strokeWidth)
	}
	s.f(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s"%s/>`,
		x, y, w, h, rx, fill, strokeAttr)
}

// Circle draws a circle from a bounding square.
func (s *SVG) Circle(x, y, size float64, fill string) {
	s.f(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`,
		x+size/2, y+size/2, size/2, fill)
}

// Line draws a line.
func (s *SVG) Line(x1, y1, x2, y2 float64, stroke string, width float64) {
	s.f(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
		x1, y1, x2, y2, stroke, width)
}

// DashedRect draws a dashed outline used for bounds/selection.
func (s *SVG) DashedRect(x, y, w, h float64, color string, width float64) {
	s.f(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="%s" stroke-width="%.1f" stroke-dasharray="7 5"/>`,
		x, y, w, h, color, width)
}

// Chip draws a small rounded label with a color swatch.
func (s *SVG) Chip(x, y, w float64, label, swatch string) {
	s.Rect(x, y, w, 26, Panel, Line, 1, 13)
	s.Rect(x+10, y+8, 10, 10, swatch, "", 0, 2)
	s.Text(x+28, y+18, label, 11, SubInk, 600)
}

// Badge draws a PASS/FAIL pill.
func (s *SVG) Badge(x, y float64, pass bool, label string) {
	fill := "#DCFCE7"
	ink := GoodGreen
	if !pass {
		fill = ErrorBg
		ink = ErrorPink
	}
	w := 12 + float64(len(label))*7.5
	s.Rect(x, y, w, 26, fill, "", 0, 13)
	s.Text(x+w/2, y+17, label, 12, ink, 800)
}

func (s *SVG) f(format string, args ...any) {
	s.body.WriteString(fmt.Sprintf(format, args...))
}

func (s *SVG) wrapText(x, y float64, text string, size float64, fill string, maxWidth float64) {
	words := strings.Split(text, "")
	line := ""
	lineY := y
	for _, ch := range words {
		trial := line + ch
		if float64(len([]rune(trial)))*size > maxWidth && line != "" {
			s.Text(x, lineY, line, size, fill, 500)
			line = ch
			lineY += size + 5
			continue
		}
		line = trial
	}
	if line != "" {
		s.Text(x, lineY, line, size, fill, 500)
	}
}

func xmlEscape(in string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(in)
}
