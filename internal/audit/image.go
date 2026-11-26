package audit

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Image stores display-encoded sRGB channels in the range [0, 1].
type Image struct {
	Width  int
	Height int
	Pixels []Vec3
}

func NewImage(width, height int) *Image {
	return &Image{
		Width:  width,
		Height: height,
		Pixels: make([]Vec3, width*height),
	}
}

func (picture *Image) At(column, row int) Vec3 {
	return picture.Pixels[row*picture.Width+column]
}

func (picture *Image) Set(column, row int, value Vec3) {
	picture.Pixels[row*picture.Width+column] = value
}

func (picture *Image) WritePNG(writer io.Writer) error {
	if picture.Width <= 0 || picture.Height <= 0 {
		return fmt.Errorf("cannot encode empty image")
	}
	output := image.NewNRGBA64(image.Rect(0, 0, picture.Width, picture.Height))
	for row := 0; row < picture.Height; row++ {
		for column := 0; column < picture.Width; column++ {
			value := picture.At(column, row).Clamp01()
			output.SetNRGBA64(column, row, color.NRGBA64{
				R: uint16(math.Round(value.X * 65535)),
				G: uint16(math.Round(value.Y * 65535)),
				B: uint16(math.Round(value.Z * 65535)),
				A: 65535,
			})
		}
	}
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(writer, output); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}
	return nil
}
