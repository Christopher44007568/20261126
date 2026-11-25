// Package atlas stitches mip images into deterministic contact sheets and
// writes standard PNG files using only image/png.
package atlas

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

const (
	Padding = 28
	Gap     = 24
	Border  = 2
)

// Cell records where one mip level was placed in the contact sheet.
type Cell struct {
	Level  int    `json:"level"`
	Label  string `json:"label"`
	X      int    `json:"atlas_x"`
	Y      int    `json:"atlas_y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Sheet is a composed RGBA atlas and its level rectangles.
type Sheet struct {
	Image  *image.RGBA `json:"-"`
	Cells  []Cell      `json:"cells"`
	Width  int         `json:"width"`
	Height int         `json:"height"`
}

// HorizontalSheet lays levels out from largest to smallest on one row.
func HorizontalSheet(levels []*image.RGBA, labels []string) (*Sheet, error) {
	if len(levels) == 0 {
		return nil, fmt.Errorf("atlas: no images")
	}
	if labels != nil && len(labels) != len(levels) {
		return nil, fmt.Errorf("atlas: got %d labels for %d images", len(labels), len(levels))
	}

	maxHeight := 0
	totalWidth := Padding*2 + Gap*(len(levels)-1)
	for _, img := range levels {
		if img == nil {
			return nil, fmt.Errorf("atlas: nil level image")
		}
		bounds := img.Bounds()
		totalWidth += bounds.Dx()
		if bounds.Dy() > maxHeight {
			maxHeight = bounds.Dy()
		}
	}
	sheetHeight := Padding*2 + maxHeight
	canvas := image.NewRGBA(image.Rect(0, 0, totalWidth, sheetHeight))
	fill(canvas, color.RGBA{R: 28, G: 32, B: 40, A: 255})

	sheet := &Sheet{Image: canvas, Width: totalWidth, Height: sheetHeight}
	x := Padding
	y := Padding
	for i, img := range levels {
		width := img.Bounds().Dx()
		height := img.Bounds().Dy()
		drawRGBA(canvas, img, x, y)
		drawBorder(canvas, x-Border, y-Border, width+Border*2, height+Border*2,
			color.RGBA{R: 91, G: 192, B: 235, A: 255})

		label := fmt.Sprintf("level-%d", i)
		if labels != nil {
			label = labels[i]
		}
		sheet.Cells = append(sheet.Cells, Cell{
			Level: i, Label: label, X: x, Y: y, Width: width, Height: height,
		})
		x += width + Gap
	}
	return sheet, nil
}

// WritePNG encodes the sheet as a standard PNG without external codecs.
func WritePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, img); err != nil {
		return err
	}
	return file.Close()
}

func fill(img *image.RGBA, fillColor color.Color) {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.Set(x, y, fillColor)
		}
	}
}

func drawRGBA(dst *image.RGBA, src *image.RGBA, x, y int) {
	for sy := 0; sy < src.Rect.Dy(); sy++ {
		for sx := 0; sx < src.Rect.Dx(); sx++ {
			sourceOffset := src.PixOffset(src.Rect.Min.X+sx, src.Rect.Min.Y+sy)
			destinationOffset := dst.PixOffset(x+sx, y+sy)
			copy(dst.Pix[destinationOffset:destinationOffset+4],
				src.Pix[sourceOffset:sourceOffset+4])
		}
	}
}

func drawBorder(img *image.RGBA, x, y, width, height int, borderColor color.Color) {
	for bx := x; bx < x+width; bx++ {
		for b := 0; b < Border; b++ {
			img.Set(bx, y+b, borderColor)
			img.Set(bx, y+height-1-b, borderColor)
		}
	}
	for by := y; by < y+height; by++ {
		for b := 0; b < Border; b++ {
			img.Set(x+b, by, borderColor)
			img.Set(x+width-1-b, by, borderColor)
		}
	}
}
