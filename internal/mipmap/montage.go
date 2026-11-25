package mipmap

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

var (
	canvasColor = color.RGBA{R: 18, G: 21, B: 27, A: 255}
	cellColor   = color.RGBA{R: 31, G: 36, B: 46, A: 255}
	borderColor = color.RGBA{R: 74, G: 84, B: 104, A: 255}
	textColor   = color.RGBA{R: 235, G: 241, B: 250, A: 255}
	mutedColor  = color.RGBA{R: 166, G: 178, B: 196, A: 255}
)

type MontageItem struct {
	Image image.Image
	Label string
}

func BuildMipMontage(title string, items []MontageItem) *image.RGBA {
	const columns = 3
	const cellWidth = 248
	const cellHeight = 214
	const gap = 14
	const titleHeight = 44

	rows := (len(items) + columns - 1) / columns
	width := sheetPadding*2 + columns*cellWidth + (columns-1)*gap
	height := sheetPadding + titleHeight + rows*cellHeight + (rows-1)*gap + sheetPadding
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fillBackground(dst, canvasColor)
	DrawText(dst, title, sheetPadding, sheetPadding-2, 3, textColor)

	for index, item := range items {
		column := index % columns
		row := index / columns
		x := sheetPadding + column*(cellWidth+gap)
		y := sheetPadding + titleHeight + row*(cellHeight+gap)
		cell := image.Rect(x, y, x+cellWidth, y+cellHeight)
		fillRect(dst, cell, cellColor)
		drawRectangleBorder(dst, cell, borderColor)

		imageArea := image.Rect(x+10, y+10, x+cellWidth-10, y+cellHeight-labelHeight)
		drawImageAtIntegerFit(dst, item.Image, imageArea)
		DrawText(dst, item.Label, x+12, y+cellHeight-19, 2, mutedColor)
	}
	return dst
}

func drawRectangleBorder(dst *image.RGBA, rect image.Rectangle, c color.Color) {
	fillRect(dst, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+1), c)
	fillRect(dst, image.Rect(rect.Min.X, rect.Max.Y-1, rect.Max.X, rect.Max.Y), c)
	fillRect(dst, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+1, rect.Max.Y), c)
	fillRect(dst, image.Rect(rect.Max.X-1, rect.Min.Y, rect.Max.X, rect.Max.Y), c)
}

func LevelLabel(level int, img *RGBImage) string {
	return fmt.Sprintf("L%02d %dX%d", level, img.Width, img.Height)
}

func EncodedColorItems(levels []*RGBImage) []MontageItem {
	items := make([]MontageItem, 0, len(levels))
	for index, level := range levels {
		items = append(items, MontageItem{Image: level.ToOutputRGBA(), Label: LevelLabel(index, level)})
	}
	return items
}

func EncodedNormalItems(normals []*RGBImage) []MontageItem {
	items := make([]MontageItem, 0, len(normals))
	for index, normal := range normals {
		items = append(items, MontageItem{Image: normal.ToNormalRGBA(), Label: LevelLabel(index, normal)})
	}
	return items
}

func drawImageAtIntegerFit(dst *image.RGBA, src image.Image, area image.Rectangle) {
	bounds := src.Bounds()
	scale := float64(area.Dx()-4) / float64(maxInt(1, bounds.Dx()))
	if heightScale := float64(area.Dy()-4) / float64(maxInt(1, bounds.Dy())); heightScale < scale {
		scale = heightScale
	}

	var scaled *image.RGBA
	if scale >= 1 {
		factor := int(scale)
		if factor < 1 {
			factor = 1
		}
		scaled = ScaleNearest(src, factor)
	} else {
		targetWidth := maxInt(1, int(float64(bounds.Dx())*scale))
		targetHeight := maxInt(1, int(float64(bounds.Dy())*scale))
		scaled = ResizeNearest(src, targetWidth, targetHeight)
	}
	x := area.Min.X + (area.Dx()-scaled.Bounds().Dx())/2
	y := area.Min.Y + (area.Dy()-scaled.Bounds().Dy())/2
	draw.Draw(dst, scaled.Bounds().Add(image.Pt(x, y)), scaled, image.Point{}, draw.Over)
}

func levelShortLabel(level int) string {
	return fmt.Sprintf("L%02d", level)
}
