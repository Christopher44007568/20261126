package main

import (
	"fmt"
	"image/color"
)

const (
	panelW    = SceneW
	panelH    = SceneH + headerH + footerH
	headerH   = 42
	footerH   = 30
	panelGap  = 18
	outerPad  = 22
	maskW     = 150
	maskH     = 96
	topTitleH = 34
)

var (
	bgColor      = color.NRGBA{0x14, 0x16, 0x1a, 255}
	panelBG      = color.NRGBA{0x0c, 0x0d, 0x10, 255}
	textColor    = color.NRGBA{0xe8, 0xea, 0xee, 255}
	subTextColor = color.NRGBA{0xa8, 0xb0, 0xbc, 255}
	maskOnColor  = color.NRGBA{0xff, 0x2d, 0x95, 255} // magenta = clipped
	maskOffColor = color.NRGBA{0x1c, 0x1f, 0x26, 255}
	frameColor   = color.NRGBA{0x3a, 0x40, 0x4b, 255}
)

// PanelResult binds a display to its rendered output for the contact sheet.
type PanelResult struct {
	Display *Display
	Render  *Rendered
}

// BuildComposite lays out the three display panels left/middle/right and
// draws each panel's clipping mask as both an overlay and an inset.
func BuildComposite(results []PanelResult, ev float64) *RGBImage {
	sheetW := outerPad*2 + panelW*3 + panelGap*2
	sheetH := outerPad + topTitleH + panelH + outerPad
	sheet := NewRGBImage(sheetW, sheetH)
	sheet.fillRect(0, 0, sheetW, sheetH, bgColor)

	drawText(sheet, outerPad, outerPad, 2,
		fmt.Sprintf("SAME SCENE, SHARED EXPOSURE  EV %+0.1F  -  SOFT PROOFED TO sRGB D65", ev),
		textColor)

	for i, res := range results {
		px := outerPad + i*(panelW+panelGap)
		py := outerPad + topTitleH
		drawPanel(sheet, px, py, res, ev)
	}
	return sheet
}

func drawPanel(sheet *RGBImage, px, py int, res PanelResult, ev float64) {
	d, r := res.Display, res.Render

	sheet.fillRect(px, py, px+panelW, py+panelH, panelBG)
	sheet.border(px, py, px+panelW, py+panelH, 2, frameColor)

	// Header: display name and display-specific characteristics.
	drawText(sheet, px+10, py+7, 2, d.Name, textColor)
	drawText(sheet, px+10, py+25, 1, d.Sub, subTextColor)

	imgY := py + headerH
	for y := 0; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			idx := y*SceneW + x
			sheet.drawVec(px+x, imgY+y, r.Preview[idx])
			if r.Mask[idx] {
				// Magenta overlay marks pixels that lost information.
				sheet.fill(px+x, imgY+y, maskOnColor)
			}
		}
	}

	// Dedicated clipping-mask inset, drawn with nearest-neighbor sampling.
	maskX := px + panelW - maskW - 12
	maskY := imgY + SceneH - maskH - 12
	drawMaskInset(sheet, maskX, maskY, r.Mask)

	// Footer: shared exposure and this display's measured clip percentage.
	footY := py + headerH + SceneH + 8
	footer := fmt.Sprintf("EV %+0.1F   CLIP %.2F%%   PEAK %.0F NITS",
		ev, r.ClipPct, d.PeakNits)
	drawText(sheet, px+10, footY, 1, footer, textColor)
}

func drawMaskInset(sheet *RGBImage, x0, y0 int, mask []bool) {
	sheet.fillRect(x0-2, y0-2, x0+maskW+2, y0+maskH+2, frameColor)
	for my := 0; my < maskH; my++ {
		sy := my * SceneH / maskH
		for mx := 0; mx < maskW; mx++ {
			sx := mx * SceneW / maskW
			col := maskOffColor
			if mask[sy*SceneW+sx] {
				col = maskOnColor
			}
			sheet.fill(x0+mx, y0+my, col)
		}
	}
	drawText(sheet, x0+4, y0+4, 1, "CLIP MASK", color.NRGBA{0, 0, 0, 255})
}
