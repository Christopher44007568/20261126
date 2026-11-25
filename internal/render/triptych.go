package render

import (
	"image/color"
)

const (
	gap      = 14
	headerH  = 34
	footerH  = 30
	panelGap = 10
)

// Triptych lays out the three pipelines side by side with titles and a footer.
func Triptych(panels []*Canvas, titles, footer []string) *Canvas {
	pw, ph := panels[0].W(), panels[0].H()
	width := gap*2 + pw*3 + panelGap*2
	height := headerH + ph + footerH + gap
	c := New(width, height)
	c.Fill(color.RGBA{R: 24, G: 26, B: 32, A: 255})

	white := color.RGBA{R: 232, G: 234, B: 240, A: 255}
	grey := color.RGBA{R: 150, G: 156, B: 168, A: 255}
	for i, p := range panels {
		px := gap + i*(pw+panelGap)
		c.FillRect(px-2, headerH-2, px+pw+2, headerH+ph+2,
			color.RGBA{R: 48, G: 52, B: 62, A: 255})
		c.blit(p, px, headerH)
		c.DrawText(px, 12, 2, titles[i], white)
	}

	// Footer metrics: up to three short lines distributed across the width.
	for i, line := range footer {
		if i >= 3 {
			break
		}
		x := gap + i*(width-gap*2)/3
		c.DrawText(x, headerH+ph+9, 1, line, grey)
	}
	return c
}

func (c *Canvas) blit(src *Canvas, ox, oy int) {
	for y := 0; y < src.H(); y++ {
		for x := 0; x < src.W(); x++ {
			c.Set(ox+x, oy+y, src.Img.RGBAAt(x, y))
		}
	}
}
