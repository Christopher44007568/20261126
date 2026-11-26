package render

import "image/color"

var (
	bgDeep     = color.RGBA{4, 10, 20, 255}
	bgPanel    = color.RGBA{10, 24, 42, 255}
	bgPanelAlt = color.RGBA{13, 31, 52, 255}
	bgInset    = color.RGBA{5, 14, 27, 255}
	lineDim    = color.RGBA{34, 55, 80, 255}
	lineMid    = color.RGBA{55, 84, 115, 255}
	textMain   = color.RGBA{220, 234, 245, 255}
	textMuted  = color.RGBA{122, 151, 177, 255}
	textFaint  = color.RGBA{77, 104, 130, 255}
	cyan       = color.RGBA{67, 224, 238, 255}
	lime       = color.RGBA{174, 239, 91, 255}
	amber      = color.RGBA{255, 184, 77, 255}
	magenta    = color.RGBA{255, 75, 181, 255}
	red        = color.RGBA{255, 89, 102, 255}
	green      = color.RGBA{94, 236, 151, 255}
	blue       = color.RGBA{99, 164, 255, 255}
	whiteWarm  = color.RGBA{255, 249, 229, 255}
)

func drawPanel(c *Canvas, x, y, w, h int) {
	c.Rect(x, y, w, h, bgPanel)
	c.Stroke(x, y, w, h, 1, lineDim)
	c.HLine(x+1, x+w-2, y+1, color.RGBA{20, 43, 67, 255})
}

func drawInset(c *Canvas, x, y, w, h int) {
	c.Rect(x, y, w, h, bgInset)
	c.Stroke(x, y, w, h, 1, lineDim)
}

func drawBadge(c *Canvas, x, y int, label string, col color.RGBA) int {
	w := textWidth(label, 1) + 12
	c.Pill(x, y, w, 13, color.RGBA{7, 18, 31, 235}, col)
	drawText(c, x+6, y+3, 1, label, col)
	return w
}
