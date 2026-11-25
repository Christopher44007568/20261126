package mipmap

import (
	"image"
	"image/color"
	"image/draw"
)

type SeamColumn struct {
	Title    string
	Levels   []*RGBImage
	AsNormal bool
}

func BuildSeamSheet(title string, columns []SeamColumn) *image.RGBA {
	const panel = 168
	const rowHeight = 190
	const labelWidth = 58
	const columnGap = 22
	const titleHeight = 40
	const headerHeight = 30

	levelCount := 0
	if len(columns) > 0 {
		levelCount = len(columns[0].Levels)
	}
	width := sheetPadding + labelWidth + len(columns)*panel + (len(columns)-1)*columnGap + sheetPadding
	height := sheetPadding + titleHeight + headerHeight + levelCount*rowHeight + sheetPadding
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fillBackground(dst, canvasColor)
	DrawText(dst, title, sheetPadding, sheetPadding-2, 3, textColor)

	gridTop := sheetPadding + titleHeight + headerHeight
	for columnIndex, column := range columns {
		columnX := sheetPadding + labelWidth + columnIndex*(panel+columnGap)
		DrawText(dst, column.Title, columnX+4, sheetPadding+titleHeight, 2, textColor)

		for levelIndex, level := range column.Levels {
			panelRect := image.Rect(
				columnX,
				gridTop+levelIndex*rowHeight,
				columnX+panel,
				gridTop+levelIndex*rowHeight+panel,
			)
			fillRect(dst, panelRect, cellColor)
			drawRectangleBorder(dst, panelRect, borderColor)
			if columnIndex == 0 {
				labelY := panelRect.Min.Y + panel/2 - 5
				DrawText(dst, levelShortLabel(levelIndex), sheetPadding, labelY, 2, mutedColor)
			}
			drawSeamCropPanel(dst, level, column.AsNormal, panelRect)
		}
	}
	return dst
}

func drawSeamCropPanel(dst *image.RGBA, level *RGBImage, asNormal bool, panel image.Rectangle) {
	radius := levelSeamRadius(level)
	crop := PeriodicCornerCrop(level, radius)
	var src image.Image
	if asNormal {
		src = crop.ToNormalRGBA()
	} else {
		src = crop.ToOutputRGBA()
	}

	bounds := src.Bounds()
	available := panel.Dx() - 12
	factor := available / maxInt(bounds.Dx(), bounds.Dy())
	if factor < 1 {
		factor = 1
	}
	scaled := ScaleNearest(src, factor)
	x0 := panel.Min.X + (panel.Dx()-scaled.Bounds().Dx())/2
	y0 := panel.Min.Y + (panel.Dy()-scaled.Bounds().Dy())/2
	draw.Draw(dst, scaled.Bounds().Add(image.Pt(x0, y0)), scaled, image.Point{}, draw.Over)

	guide := color.RGBA{R: 255, G: 54, B: 170, A: 255}
	seamX := x0 + radius*2*factor/2
	seamY := y0 + radius*2*factor/2
	scaledSize := radius * 2 * factor
	fillRect(dst, image.Rect(seamX, y0, seamX+1, y0+scaledSize), guide)
	fillRect(dst, image.Rect(x0, seamY, x0+scaledSize, seamY+1), guide)
}
