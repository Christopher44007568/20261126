package render

import (
	"fmt"
	"image"
	"image/color"
)

const (
	ladderW = 1280
	ladderH = 470
)

func renderLadder(report MetricsReport) image.Image {
	c := NewCanvas(ladderW, ladderH, bgDeep)
	drawBackdrop(c)
	c.Rect(0, 0, ladderW, 4, cyan)
	drawTextShadow(c, 28, 20, 3, "SUB-CODE DITHER LADDER", textMain)
	drawText(c, 29, 52, 1,
		"16 FRACTIONS BETWEEN TWO ADJACENT 8-BIT CODES / 4X4 BAYER / FALSE-COLOR DENSITY MAP",
		textMuted)
	drawBadge(c, ladderW-270, 25, "CYAN = UPPER CODE", cyan)
	drawBadge(c, ladderW-120, 25, "100 -> 101", whiteWarm)
	c.HLine(28, ladderW-28, 78, lineDim)

	const (
		x0     = 150
		right  = ladderW - 30
		blockH = 54
		colGap = 5
		rowTop = 112
		rowGap = 15
		rowH   = 92
	)
	colW := (right - x0 - 15*colGap) / 16
	for i := 0; i < 16; i++ {
		x := x0 + i*(colW+colGap)
		drawTextCentered(c, x+colW/2, 88, 1, fmt.Sprintf("%02d", i), textFaint)
	}

	for row, p := range Pipelines {
		y := rowTop + row*(rowH+rowGap)
		c.Rect(24, y, ladderW-48, rowH, color.RGBA{8, 21, 38, 255})
		c.Stroke(24, y, ladderW-48, rowH, 1, lineDim)
		c.Rect(24, y, 5, rowH, cyan)
		drawText(c, 42, y+12, 2, p.Short, textMain)
		drawText(c, 42, y+38, 1,
			fmt.Sprintf("DENS %d/16", report.LadderDensities[p.Short]),
			lime)
		if p.Dither {
			drawText(c, 42, y+54, 1, "FRACTIONAL", textMuted)
		} else {
			drawText(c, 42, y+54, 1, "ROUNDED", textMuted)
		}
		for i := 0; i < 16; i++ {
			x := x0 + i*(colW+colGap)
			target := 100 + float64(i)/16
			drawLadderBlock(c, p, x, y+8, colW, blockH, target)
		}
	}

	y := ladderH - 46
	c.HLine(28, ladderW-28, y, lineDim)
	drawText(c, 30, y+12, 1,
		"TOP = WHICH PIXELS RECEIVE THE UPPER CODE; BOTTOM 8 PX = ACTUAL 8-BIT COLOR. THE ACTUAL STRIPS LOOK NEARLY SOLID.",
		textMuted)
	drawText(c, 30, y+26, 1,
		"THE VISIBLE WORK HAPPENS IN DENSITY: 12-BIT HAS 16 DISTINCT BAYER PHASES, 10-BIT ABOUT 4, DIRECT 8-BIT ONLY TWO.",
		textFaint)
	return c.RGBA
}

func drawLadderBlock(c *Canvas, p Pipeline, x, y, w, h int, target float64) {
	c.Rect(x, y, w, h, bgInset)
	c.Stroke(x, y, w, h, 1, lineDim)
	for yy := 0; yy < h-8; yy++ {
		for xx := 0; xx < w; xx++ {
			code := DisplayCode(target/255, p, xx, yy)
			if code > 100 {
				c.Set(x+xx, y+yy, color.RGBA{29, 98, 116, 255})
			} else {
				c.Set(x+xx, y+yy, color.RGBA{7, 20, 34, 255})
			}
		}
	}
	for yy := h - 8; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			code := uint8(DisplayCode(target/255, p, xx, yy))
			c.Set(x+xx, y+yy, color.RGBA{code, code, code, 255})
		}
	}
	c.HLine(x, x+w-1, y+h-9, lineMid)
}
