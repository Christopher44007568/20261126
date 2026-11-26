package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

const (
	puzzleW = 2048
	puzzleH = 1280
	sheetW  = 720
	sheetH  = 1140
)

func renderPuzzle() image.Image {
	c := NewCanvas(puzzleW, puzzleH, bgDeep)
	drawBackdrop(c)
	drawPuzzleHeader(c)

	const (
		margin = 28
		railW  = 132
		colGap = 16
		rowGap = 14
		rowH   = 330
		rowTop = 172
		colTop = 118
		colH   = 42
		xStart = margin + railW
		colW   = (puzzleW - xStart - margin - 2*colGap) / 3
	)
	for i, p := range Pipelines {
		x := xStart + i*(colW+colGap)
		drawColumnHeader(c, x, colTop, colW, colH, p)
	}
	for row, sc := range Scenes {
		y := rowTop + row*(rowH+rowGap)
		drawSceneRail(c, margin, y, railW, rowH, row+1, sc)
		for i, p := range Pipelines {
			x := xStart + i*(colW+colGap)
			renderSceneCard(c, sc, p, x, y, colW, rowH)
		}
	}
	drawPuzzleFooter(c)
	return c.RGBA
}

func renderSheet(active Pipeline) image.Image {
	c := NewCanvas(sheetW, sheetH, bgDeep)
	drawBackdrop(c)

	drawTextShadow(c, 22, 20, 3, active.Name, cyan)
	subtitle := "SAME FLOAT SOURCE / SAME HUE PATH / 8-BIT DISPLAY"
	if active.Dither {
		subtitle = "SAME FLOAT SOURCE / SAME HUE PATH / 4X4 BAYER TO 8-BIT"
	}
	drawText(c, 22, 54, 1, subtitle, textMuted)
	drawBadge(c, sheetW-120, 24, "HUE LOCKED", lime)
	c.HLine(22, sheetW-22, 78, lineDim)

	const (
		x = 20
		w = sheetW - 40
		h = 320
	)
	y := 96
	for _, sc := range Scenes {
		renderSceneCard(c, sc, active, x, y, w, h)
		y += h + 16
	}
	drawSheetFooter(c)
	return c.RGBA
}

func drawBackdrop(c *Canvas) {
	top := color.RGBA{8, 21, 38, 255}
	for y := 0; y < c.H; y++ {
		t := float64(y) / float64(max(1, c.H-1))
		col := color.RGBA{
			R: uint8(Lerp(float64(top.R), float64(bgDeep.R), t)),
			G: uint8(Lerp(float64(top.G), float64(bgDeep.G), t)),
			B: uint8(Lerp(float64(top.B), float64(bgDeep.B), t)),
			A: 255,
		}
		c.HLine(0, c.W-1, y, col)
	}
	for x := 0; x < c.W; x += 48 {
		for y := 0; y < c.H; y += 48 {
			c.Blend(x, y, lineDim, 55)
		}
	}
}

func drawPuzzleHeader(c *Canvas) {
	c.Rect(0, 0, puzzleW, 4, cyan)
	drawTextShadow(c, 28, 24, 4, "BIT-DEPTH GRADIENT LAB", textMain)
	drawText(c, 30, 68, 1, "ONE SOURCE / 10-BIT AND 12-BIT MASTERS / 8-BIT DISPLAY", cyan)
	drawText(c, 30, 82, 1, "SHADOW STEPS - COLOR SLOPE - HIGHLIGHT CLIP", textMuted)

	drawBadge(c, puzzleW-206, 26, "HUE LOCKED", lime)
	drawBadge(c, puzzleW-112, 26, "4X4 BAYER", cyan)
	drawTextRight(c, puzzleW-28, 54, 1, "DITHERED DEPTH / HARD 8-BIT RAIL", textMuted)
	drawTextRight(c, puzzleW-28, 68, 1, "ERROR AND CLIP OVERLAYS ARE PIXEL ALIGNED", textFaint)
	c.HLine(28, puzzleW-28, 102, lineDim)
}

func drawColumnHeader(c *Canvas, x, y, w, h int, p Pipeline) {
	c.Rect(x, y, w, h, bgPanelAlt)
	c.Stroke(x, y, w, h, 1, lineDim)
	drawTextCentered(c, x+w/2, y+7, 2, p.Name, textMain)
	mode := "HARD 8-BIT / NO DITHER"
	if p.Dither {
		mode = fmt.Sprintf("MASTER %d-BIT / 4X4 BAYER", p.Bits)
	}
	drawTextCentered(c, x+w/2, y+27, 1, mode, textMuted)
	c.Rect(x, y, 5, h, cyan)
}

func drawSceneRail(c *Canvas, x, y, w, h, index int, sc Scene) {
	c.Rect(x, y, w, h, color.RGBA{7, 18, 31, 255})
	c.Stroke(x, y, w, h, 1, lineDim)
	c.Rect(x, y, 6, h, cyan)
	drawText(c, x+18, y+18, 3, fmt.Sprintf("%02d", index), cyan)
	for i, word := range strings.Fields(sc.Title) {
		drawText(c, x+18, y+54+i*22, 2, word, textMain)
	}
	drawText(c, x+18, y+108, 1, sc.Subtitle, textFaint)
	drawText(c, x+18, y+126, 1, "SOURCE LOCK", textMuted)
	drawBadge(c, x+18, y+h-30, "DEPTH", lime)
}

func drawPuzzleFooter(c *Canvas) {
	y := 1200
	c.HLine(28, puzzleW-28, y, lineDim)
	drawText(c, 30, y+15, 1,
		"READ / SAME SOURCE IS SAMPLED ONCE. COLUMNS CHANGE ONLY THE MASTER BIT DEPTH AND 8-BIT CONVERSION, SO HUE DOES NOT RESET.",
		textMain)
	drawText(c, 30, y+29, 1,
		"ERROR / HEAT STRIP = MEAN |FINAL 8-BIT - IDEAL 8-BIT| ACROSS RGB. DASHED LINES = R/G/B CLIPS. MAGENTA HATCH = SOURCE CLIPPED.",
		textMuted)
	drawText(c, 30, y+43, 1,
		"VISIBLE DEPTH / 10/12-BIT KEEP A FRACTIONAL 8-BIT COORDINATE AND BAYER-DITHER IT; THE EXTRA LEVELS BECOME DENSITY, NOT EXTRA DISPLAY CODES.",
		textMuted)
	drawText(c, 30, y+57, 1,
		"12-BIT RESOLVES 16 SUBCODE PHASES BETWEEN ADJACENT 8-BIT CODES; 10-BIT RESOLVES ABOUT 4. SEE THE SUB-CODE DITHER LADDER.",
		textFaint)
}

func drawSheetFooter(c *Canvas) {
	y := sheetH - 38
	c.HLine(22, sheetW-22, y, lineDim)
	drawText(c, 22, y+10, 1, "ALL THREE PANELS USE THE SAME SOURCE FUNCTION AND PANEL GEOMETRY.", textMuted)
	drawText(c, 22, y+22, 1, "TOGGLE SHEETS TO CHECK HUE AND CLIP ALIGNMENT.", textFaint)
}
