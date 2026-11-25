package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func pngDecode(file *os.File) (image.Image, error) {
	return png.Decode(file)
}

func sRGB(linear float64) float64 {
	linear = clampFloat(linear, 0, 1)
	if linear <= 0.0031308 {
		return 12.92 * linear
	}
	return 1.055*math.Pow(linear, 1/2.4) - 0.055
}

func encodeNormalized(scene *floatImage, blackLevel, whiteLevel float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, scene.width, scene.height))
	for y := 0; y < scene.height; y++ {
		for x := 0; x < scene.width; x++ {
			base := img.PixOffset(x, y)
			img.Pix[base+3] = 255
			for c := 0; c < 3; c++ {
				linear := (scene.at(x, y, c) - blackLevel) / (whiteLevel - blackLevel)
				img.Pix[base+c] = uint8(clampFloat(math.Round(sRGB(linear)*255), 0, 255))
			}
		}
	}
	return img
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func copyScaled(dst *image.NRGBA, src image.Image, x0, y0, scale int) {
	bounds := src.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					dst.SetNRGBA(x0+(x-bounds.Min.X)*scale+dx, y0+(y-bounds.Min.Y)*scale+dy, c)
				}
			}
		}
	}
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	width := rect.Dx()
	height := rect.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dst.SetNRGBA(x-rect.Min.X, y-rect.Min.Y, color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA))
		}
	}
	return dst
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
