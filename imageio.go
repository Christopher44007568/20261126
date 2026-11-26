package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	return encoder.Encode(file, img)
}

func SaveJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func HorizontalSheet(images []image.Image, gap int, border color.RGBA) *image.RGBA {
	if len(images) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	height := images[0].Bounds().Dy()
	width := gap + len(images)*(images[0].Bounds().Dx()+gap)
	sheet := image.NewRGBA(image.Rect(0, 0, width, height+2*gap))
	fillRect(sheet, sheet.Bounds(), color.RGBA{R: 18, G: 22, B: 34, A: 255})

	x := gap
	for _, img := range images {
		drawImage(sheet, img, x, gap)
		rect := image.Rect(x-1, gap-1, x+img.Bounds().Dx()+1, gap+height+1)
		strokeRect(sheet, rect, border)
		x += img.Bounds().Dx() + gap
	}
	return sheet
}

func GridSheet(images []image.Image, cols, gap int) *image.RGBA {
	if len(images) == 0 || cols <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	rows := (len(images) + cols - 1) / cols
	cellW := images[0].Bounds().Dx()
	cellH := images[0].Bounds().Dy()
	width := gap + cols*(cellW+gap)
	height := gap + rows*(cellH+gap)
	sheet := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(sheet, sheet.Bounds(), color.RGBA{R: 18, G: 22, B: 34, A: 255})

	for i, img := range images {
		col := i % cols
		row := i / cols
		x := gap + col*(cellW+gap)
		y := gap + row*(cellH+gap)
		drawImage(sheet, img, x, y)
		strokeRect(sheet, image.Rect(x-1, y-1, x+cellW+1, y+cellH+1), color.RGBA{R: 210, G: 220, B: 255, A: 255})
	}
	return sheet
}

func drawImage(dst *image.RGBA, src image.Image, x, y int) {
	bounds := src.Bounds()
	for sy := bounds.Min.Y; sy < bounds.Max.Y; sy++ {
		for sx := bounds.Min.X; sx < bounds.Max.X; sx++ {
			dst.Set(x+sx-bounds.Min.X, y+sy-bounds.Min.Y, src.At(sx, sy))
		}
	}
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	rgba := c.(color.RGBA)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetRGBA(x, y, rgba)
		}
	}
}

func strokeRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.SetRGBA(x, rect.Min.Y, c)
		img.SetRGBA(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.SetRGBA(rect.Min.X, y, c)
		img.SetRGBA(rect.Max.X-1, y, c)
	}
}
