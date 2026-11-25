package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func renderImage(scene Scene, winners []PixelWinner) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, scene.Width, scene.Height))
	for _, winner := range winners {
		rgb := winner.Opinion.Material.Color
		img.SetRGBA(winner.X, winner.Y, color.RGBA{R: rgb.R, G: rgb.G, B: rgb.B, A: 0xFF})
	}
	return img
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

// contactSheet lays the three snapshots out left-to-right with neutral gutters.
// The order matches RenderModes and README.md.
func contactSheet(images []*image.RGBA) *image.RGBA {
	const gutter = 12
	if len(images) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	width := gutter*(len(images)+1) + images[0].Bounds().Dx()*len(images)
	height := gutter*2 + images[0].Bounds().Dy()
	sheet := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 238, G: 241, B: 245, A: 0xFF}}, image.Point{}, draw.Src)
	x := gutter
	for _, img := range images {
		target := image.Rect(x, gutter, x+img.Bounds().Dx(), gutter+img.Bounds().Dy())
		draw.Draw(sheet, target, img, image.Point{}, draw.Src)
		x += img.Bounds().Dx() + gutter
	}
	return sheet
}
