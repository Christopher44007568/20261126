package render

import (
	"fmt"
	"image/color"
	"math"
)

const sceneMainHeight = 128

func renderSceneCard(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	drawPanel(c, x, y, w, h)
	drawText(c, x+14, y+10, 2, sc.Title, textMain)
	drawText(c, x+14, y+30, 1, sc.Subtitle, textFaint)
	tag := "HARD / NO DITHER"
	tagCol := amber
	if p.Dither {
		tag = "4X4 BAYER"
		tagCol = cyan
	}
	tw := textWidth(tag, 1) + 12
	drawBadge(c, x+w-tw-12, y+12, tag, tagCol)

	mainX, mainY := x+12, y+44
	mainW := w - 24
	drawScenePixels(c, sc, p, mainX, mainY, mainW, sceneMainHeight)
	c.Stroke(mainX, mainY, mainW, sceneMainHeight, 1, lineMid)
	drawSceneOverlay(c, sc, p, mainX, mainY, mainW, sceneMainHeight)

	traceY := mainY + sceneMainHeight + 4
	drawSceneTrace(c, sc, p, x+12, traceY, mainW, 30)

	heatY := traceY + 34
	drawErrorHeat(c, sc, p, x+12, heatY, mainW, 28)

	infoY := heatY + 34
	m := measure(sc, p)
	hueText := "HUE PATH LOCKED"
	if m.HueApplicable {
		hueText = fmt.Sprintf("HUE %.2f/%.2f DEG", m.HueMAE, m.HueMax)
	}
	dens := ladderDensities()[p.Short]
	drawText(c, x+14, infoY, 1,
		fmt.Sprintf("DENS %d/16  QMAE %.2f  BLK %.2f  %s",
			dens, m.HardQuantMAE, m.DitherBlockRMSE, hueText),
		textMuted)
	drawText(c, x+14, infoY+14, 1,
		fmt.Sprintf("RUNS %d  SUBSTEP %.2f CODE  CLIP %.0f%%  LOSS %.1f",
			m.HardBandRuns, m.SubcodeStep8Bit, m.ClippedPixelFraction*100, m.HardClipLoss),
		textFaint)
	mode := "HARD 8-BIT: THE EXTRA MASTER LEVELS WERE ROUNDED AWAY"
	if p.Dither {
		mode = "FRACTIONAL 8-BIT CODE RETAINED, THEN SPATIALLY AVERAGED"
	}
	drawText(c, x+14, infoY+28, 1, mode, textFaint)
}

func drawSceneTrace(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	drawInset(c, x, y, w, h)
	const labelW = 66
	plotX := x + labelW
	plotY := y + 3
	plotW := w - labelW - 5
	plotH := h - 6

	switch sc.ID {
	case "shadow-steps":
		drawText(c, x+5, y+3, 1, "CODE 1-56", textFaint)
		prevIdeal, prevHard := -1, -1
		for px := 0; px < plotW; px++ {
			sx := px * w / max(1, plotW)
			src := sourceAt(sc, sx, sceneMainHeight/2, w, sceneMainHeight)
			ideal := int(math.Floor(Clamp01(src.R)*255 + 0.5))
			hard := HardCode(src.R, p.Bits)
			yi := mapCode(ideal, 0, 56, plotY, plotH)
			yh := mapCode(hard, 0, 56, plotY, plotH)
			if prevIdeal >= 0 {
				c.Line(plotX+px-1, prevIdeal, plotX+px, yi, cyan)
				c.Line(plotX+px-1, prevHard, plotX+px, yh, amber)
			}
			prevIdeal, prevHard = yi, yh
		}
	case "color-slope":
		drawText(c, x+5, y+3, 1, "HUE PATH", textFaint)
		prev := -1
		for px := 0; px < plotW; px++ {
			u := float64(px) / float64(max(1, plotW-1))
			hue := 300 + 360*u
			yy := plotY + plotH - 1 - int(math.Round((hue-300)/360*float64(plotH-1)))
			if prev >= 0 {
				c.Line(plotX+px-1, prev, plotX+px, yy, cyan)
			}
			prev = yy
		}
		wrapX := plotX + int(math.Round(sc.HueWrapU*float64(plotW-1)))
		for yy := plotY; yy < plotY+plotH; yy += 5 {
			c.VLine(wrapX, yy, min(yy+2, plotY+plotH-1), whiteWarm)
		}
		drawText(c, plotX+3, plotY+plotH-9, 1, "300", textFaint)
		drawTextRight(c, plotX+plotW-2, plotY+1, 1, "660", textFaint)
	case "highlight-clip":
		drawText(c, x+5, y+3, 1, "RGB 0-255", textFaint)
		c.DashedH(plotX, plotX+plotW-1, plotY, 3, 4, lineMid)
		curves := [3]color.RGBA{red, green, blue}
		for ch := 0; ch < 3; ch++ {
			prev := -1
			for px := 0; px < plotW; px++ {
				sx := px * w / max(1, plotW)
				src := sourceAt(sc, sx, sceneMainHeight/2, w, sceneMainHeight)
				v := []float64{src.R, src.G, src.B}[ch]
				code := HardCode(v, p.Bits)
				yy := mapCode(code, 0, 255, plotY, plotH)
				if prev >= 0 {
					c.Line(plotX+px-1, prev, plotX+px, yy, curves[ch])
				}
				prev = yy
			}
		}
		drawText(c, x+5, y+15, 1, "R G B", textFaint)
	}
}

func mapCode(code, lo, hi, y, h int) int {
	t := (float64(code) - float64(lo)) / float64(max(1, hi-lo))
	t = Clamp01(t)
	return y + h - 1 - int(math.Round(t*float64(h-1)))
}

func drawErrorHeat(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	drawInset(c, x, y, w, h)
	drawText(c, x+5, y+10, 1, "Q ERR", textFaint)
	const legendW = 64
	barX := x + 47
	barW := w - 47 - legendW - 16
	barY := y + 5
	barH := 11
	profile := sceneErrorProfile(sc, p, barW)
	for i, err := range profile {
		t := Clamp01(err / 0.75)
		c.VLine(barX+i, barY, barY+barH-1, heatColor(t))
	}
	c.Stroke(barX, barY, barW, barH, 1, lineMid)
	legendX := barX + barW + 8
	for i := 0; i < legendW; i++ {
		c.VLine(legendX+i, barY, barY+barH-1, heatColor(float64(i)/float64(legendW-1)))
	}
	c.Stroke(legendX, barY, legendW, barH, 1, lineMid)
	drawText(c, barX, y+19, 1, "MEAN ABS ERROR / CODE / 0 TO 0.75", textFaint)
	drawTextRight(c, x+w-5, y+19, 1, "0.75", textFaint)

	if sc.ID == "highlight-clip" {
		clip := clipColumn(sc, 0, w)
		if clip > 0 {
			c.HLine(barX+clip*barW/w, barX+barW-1, barY+barH+1, magenta)
		}
	}
}

func sceneErrorProfile(sc Scene, p Pipeline, samples int) []float64 {
	out := make([]float64, samples)
	for i := range out {
		sx := i * metricW / max(1, samples)
		var sum float64
		for yy := 0; yy < sceneMainHeight; yy++ {
			src := sourceAt(sc, sx, yy, metricW, sceneMainHeight)
			disp := DisplayRGB(src, p, sx, yy)
			sum += math.Abs(disp.R - Clamp01(src.R)*255)
			sum += math.Abs(disp.G - Clamp01(src.G)*255)
			sum += math.Abs(disp.B - Clamp01(src.B)*255)
		}
		out[i] = sum / float64(sceneMainHeight*3)
	}
	return out
}

func drawScenePixels(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			src := sourceAt(sc, xx, yy, w, h)
			disp := DisplayRGB(src, p, xx, yy)
			c.Set(x+xx, y+yy, color.RGBA{
				R: uint8(ClampCode(int(disp.R))),
				G: uint8(ClampCode(int(disp.G))),
				B: uint8(ClampCode(int(disp.B))),
				A: 255,
			})
		}
	}
}

func drawSceneOverlay(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	switch sc.ID {
	case "shadow-steps":
		drawShadowOverlay(c, sc, p, x, y, w, h)
	case "color-slope":
		drawColorOverlay(c, sc, x, y, w, h)
	case "highlight-clip":
		drawHighlightOverlay(c, sc, x, y, w, h)
	}
}

func drawShadowOverlay(c *Canvas, sc Scene, p Pipeline, x, y, w, h int) {
	drawBadge(c, x+6, y+6, "CODE 1-56", whiteWarm)
	prev := -1
	for xx := 0; xx < w; xx++ {
		src := sourceAt(sc, xx, 0, w, h)
		code := HardCode(src.R, p.Bits)
		if prev >= 0 && code != prev {
			c.VLine(x+xx, y+2, y+9, color.RGBA{255, 249, 229, 190})
		}
		prev = code
	}
	prevY := -1
	for xx := 1; xx < w-1; xx++ {
		src := sourceAt(sc, xx, h/2, w, h)
		code := int(math.Floor(Clamp01(src.R)*255 + 0.5))
		yy := mapCode(code, 0, 56, y+7, h-20)
		if prevY >= 0 {
			c.Line(x+xx-1, prevY, x+xx, yy, color.RGBA{52, 168, 184, 255})
		}
		prevY = yy
	}
	drawText(c, x+6, y+h-13, 1, "TOP TICKS = HARD STEP EDGES", color.RGBA{197, 222, 235, 205})
	drawTextRight(c, x+w-6, y+h-13, 1, "CODE TRACE", color.RGBA{52, 168, 184, 235})
}

func drawColorOverlay(c *Canvas, sc Scene, x, y, w, h int) {
	col := hueWrapColumn(sc, w)
	if col < 0 {
		return
	}
	for yy := y + 2; yy < y+h-2; yy += 9 {
		c.VLine(x+col, yy, min(yy+5, y+h-2), whiteWarm)
	}
	drawBadge(c, x+col+5, y+7, "HUE 360 = 0", whiteWarm)
	drawBadge(c, x+7, y+h-20, "VALUE SLOPE", color.RGBA{230, 240, 245, 255})
}

func drawHighlightOverlay(c *Canvas, sc Scene, x, y, w, h int) {
	clipR := clipColumn(sc, 0, w)
	clipG := clipColumn(sc, 1, w)
	clipB := clipColumn(sc, 2, w)
	if clipR < 0 {
		return
	}
	c.Hatch(x+clipR, y, w-clipR, h, 8, magenta, 42)
	lines := []struct {
		col   int
		label string
		colr  color.RGBA
		yy    int
	}{
		{clipR, "R 58%", red, y + 9},
		{clipG, "G 70%", green, y + 28},
		{clipB, "B 90%", blue, y + 47},
	}
	for _, ln := range lines {
		for yy := y + 2; yy < y+h-2; yy += 9 {
			c.VLine(x+ln.col, yy, min(yy+5, y+h-2), ln.colr)
		}
		c.Triangle(x+ln.col, y+h-7, 4, 6, ln.colr)
		badgeW := textWidth(ln.label, 1) + 12
		drawBadge(c, min(x+ln.col+5, x+w-badgeW-4), ln.yy, ln.label, ln.colr)
	}
	drawText(c, x+7, y+h-13, 1, "MAGENTA HATCH = ANY CHANNEL CLIPPED", magenta)
}
