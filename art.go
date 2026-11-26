package main

import "image"

type rect struct{ x, y, w, h int }

func (r rect) imgRect() image.Rectangle {
	return image.Rect(r.x, r.y, r.x+r.w, r.y+r.h)
}

// drawShowcase renders the fixed schematic "orbit display" geometry. The
// material/base colour is intentionally not shaded onto it: the viewport is a
// deterministic fallback surface, while provenance lives in the text panel.
func drawShowcase(c *Canvas, v rect) {
	prevClip := c.Clip
	c.Clip = v.imgRect()
	defer func() { c.Clip = prevClip }()

	cx := v.x + v.w/2
	cy := v.y + v.h/2 + 8

	c.StrokeEllipse(cx, cy+30, 208, 88, 15, Hex("#222A37"))
	c.StrokeEllipse(cx, cy+30, 180, 74, 12, Hex("#2C3543"))
	c.StrokeEllipse(cx, cy+30, 152, 62, 9, Hex("#384252"))

	c.FillEllipse(cx, cy+92, 196, 46, Hex("#05080D"))
	c.FillCircle(cx, cy-4, 98, Hex("#05080D"))

	bx, by, bw, bh := cx-150, cy-86, 300, 172
	c.FillRoundRect(bx, by, bw, bh, 30, colSlate)
	c.StrokeRoundRect(bx, by, bw, bh, 30, 7, colNavyRing)

	prev := c.Clip
	c.Clip = image.Rect(bx, by, bx+bw, by+bh)
	port := 47
	c.FillCircle(bx+2, cy+2, port, Hex("#05080D"))
	c.FillCircle(bx+bw-2, cy+2, port, Hex("#05080D"))
	c.Clip = prev

	for _, gx := range []int{cx - 74, cx, cx + 74} {
		c.Line(gx, cy-58, gx, cy+60, 3, HexA("#D7E2EF", 150))
		c.FillCircle(gx, cy-58, 6, HexA("#D7E2EF", 220))
		c.FillCircle(gx, cy+60, 6, HexA("#D7E2EF", 220))
	}

	c.Ring(cx, cy-4, 46, 33, colNavyRing)
	c.FillCircle(cx, cy-4, 33, colLens)
	c.FillCircle(cx-10, cy-14, 7, HexA("#F4F8FF", 150))

	drawBrackets(c, v)
}

func drawBrackets(c *Canvas, v rect) {
	const inset = 26
	const arm = 26
	const th = 4
	col := Hex("#46536A")
	x0, y0 := v.x+inset, v.y+inset
	x1, y1 := v.x+v.w-inset, v.y+v.h-inset

	c.FillRect(x0, y0, arm, th, col)
	c.FillRect(x0, y0, th, arm, col)
	c.FillRect(x1-arm, y0, arm, th, col)
	c.FillRect(x1-th, y0, th, arm, col)
	c.FillRect(x0, y1-th, arm, th, col)
	c.FillRect(x0, y1-arm, th, arm, col)
	c.FillRect(x1-arm, y1-th, arm, th, col)
	c.FillRect(x1-th, y1-arm, th, arm, col)
}

// drawHatch paints diagonal warning stripes clipped to r.
func drawHatch(c *Canvas, r rect, col colorRGBA, spacing int) {
	prev := c.Clip
	c.Clip = r.imgRect()
	defer func() { c.Clip = prev }()
	for k := -r.h; k < r.w; k += spacing {
		x := r.x + k
		c.Line(x, r.y+r.h, x+r.h, r.y, 2, col)
	}
}

// drawFileIcon renders the readable "missing asset" document glyph.
func drawFileIcon(c *Canvas, x, y, size int, col colorRGBA) {
	w := size * 7 / 8
	c.StrokeRoundRect(x, y, w, size, 6, 4, col)
	barX := x + w/2 - 26
	for i, by := range []int{y + size/3, y + size/2, y + 2*size/3} {
		width := 52
		if i == 2 {
			width = 40
		}
		c.FillRect(barX, by-4, width, 8, col)
	}
}
