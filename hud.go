package main

import "fmt"

func (c *Canvas) Rect(x, y, w, h int, r, g, b uint8, alpha float64) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.blend(xx, yy, r, g, b, alpha)
		}
	}
}

func DrawHUD(c *Canvas, title, subtitle string) {
	scale := 3
	titleW := TextWidth(title, scale)
	c.Rect(18, 16, titleW+34, 34+18, 4, 8, 18, 0.62)
	DrawTextShadow(c, title, 35, 27, scale, 235, 242, 255)

	subScale := 2
	subW := TextWidth(subtitle, subScale)
	c.Rect(18, c.H-48, subW+30, 30, 4, 8, 18, 0.58)
	DrawText(c, subtitle, 33, c.H-40, subScale, 190, 214, 255)
}

func ApertureLabel(a Aperture, focus float64) string {
	return fmt.Sprintf("HEX %d BLADES  F/%.1F  FOCUS %.2FM  COC THIN LENS", a.Blades, a.FNumber, focus)
}
