package render

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"animationtiming/raster"
)

// WriteCombinedPNG stacks sheet canvases into one contact-sheet overview.
func WriteCombinedPNG(path string, sheets []*raster.Canvas) error {
	const margin = 16
	const gap = 18
	width := 0
	height := margin * 2
	for i, sheet := range sheets {
		if sheet.W > width {
			width = sheet.W
		}
		height += sheet.H
		if i > 0 {
			height += gap
		}
	}
	width += margin * 2

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{R: paper.R, G: paper.G, B: paper.B, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	y := margin
	for _, sheet := range sheets {
		x := (width - sheet.W) / 2
		draw.Draw(img, image.Rect(x, y, x+sheet.W, y+sheet.H), sheet.ToRGBA(), image.Point{}, draw.Src)
		y += sheet.H + gap
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
