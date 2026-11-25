package atlas

import (
	"fmt"
	"image"
	"image/color"
)

var (
	colorUsed    = color.RGBA{R: 70, G: 220, B: 150, A: 255}
	colorPadding = color.RGBA{R: 255, G: 176, B: 64, A: 255}
	colorFree    = color.RGBA{R: 70, G: 80, B: 105, A: 255}
)

// RenderUtilization draws a stacked atlas-area bar: source texels, mip-safe
// padding cells, and unused free space.
func RenderUtilization(u Utilization) *image.RGBA {
	const w, h = 760, 240
	out := NewRGBA(w, h, colorPanel)
	DrawText(out, 32, 24, "ATLAS UTILIZATION", 3, color.RGBA{R: 235, G: 240, B: 255, A: 255})
	DrawText(out, 32, 58, fmt.Sprintf("%dx%d  MIP CHAIN %.2fx", u.AtlasWidth, u.AtlasHeight, u.MipChainAreaRatio), 2, color.RGBA{R: 170, G: 182, B: 210, A: 255})

	bar := R(32, 92, w-64, 54)
	usedW := int(float64(bar.W) * u.TextureUtilization)
	cellW := int(float64(bar.W) * u.CellUtilization)
	paddingW := maxInt(0, cellW-usedW)
	freeW := bar.W - usedW - paddingW
	if freeW < 0 {
		freeW = 0
	}

	fillRGBA(out, R(bar.X, bar.Y, usedW, bar.H), colorUsed)
	fillRGBA(out, R(bar.X+usedW, bar.Y, paddingW, bar.H), colorPadding)
	fillRGBA(out, R(bar.X+usedW+paddingW, bar.Y, freeW, bar.H), colorFree)
	drawRectOutline(out, bar, colorPanelLn, 2)

	pct := func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
	legendY := 170
	legend := []struct {
		c     color.RGBA
		label string
		x     int
	}{
		{colorUsed, "SPRITE " + pct(u.TextureUtilization), 32},
		{colorPadding, "PADDING " + pct(float64(u.PaddingArea)/float64(u.AtlasArea)), 280},
		{colorFree, "FREE " + pct(float64(u.FreeArea)/float64(u.AtlasArea)), 540},
	}
	for _, item := range legend {
		fillRGBA(out, R(item.x, legendY, 18, 18), item.c)
		DrawText(out, item.x+28, legendY+2, item.label, 2, color.RGBA{R: 230, G: 236, B: 248, A: 255})
	}
	DrawText(out, 32, 210, fmt.Sprintf("PACK EFFICIENCY %s  CELL %s", pct(u.PackingEfficiency), pct(u.CellUtilization)), 2, color.RGBA{R: 150, G: 220, B: 255, A: 255})

	border := image.Rect(0, 0, w, h)
	for x := border.Min.X; x < border.Max.X; x++ {
		out.SetRGBA(x, 0, colorPanelLn)
		out.SetRGBA(x, h-1, colorPanelLn)
	}
	for y := 0; y < h; y++ {
		out.SetRGBA(0, y, colorPanelLn)
		out.SetRGBA(w-1, y, colorPanelLn)
	}
	return out
}
