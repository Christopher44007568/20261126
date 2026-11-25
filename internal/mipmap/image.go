package mipmap

import (
	"image"
	"image/color"
	"math"
)

// RGBImage stores non-alpha-premultiplied color in linear-light space.
// Samples are interleaved as R, G, B.
type RGBImage struct {
	Width  int
	Height int
	Pix    []float64
	// Linear reports whether Pix stores linear-light values. When false, Pix
	// contains gamma-encoded/output RGB values in [0,1].
	Linear bool
}

func NewRGBImage(width, height int) *RGBImage {
	return &RGBImage{
		Width:  width,
		Height: height,
		Pix:    make([]float64, width*height*3),
	}
}

func NewLinearRGBImage(width, height int) *RGBImage {
	img := NewRGBImage(width, height)
	img.Linear = true
	return img
}

func (img *RGBImage) index(x, y int) int {
	return (y*img.Width + x) * 3
}

func (img *RGBImage) Set(x, y int, r, g, b float64) {
	i := img.index(x, y)
	img.Pix[i+0] = r
	img.Pix[i+1] = g
	img.Pix[i+2] = b
}

func (img *RGBImage) RGB(x, y int) (r, g, b float64) {
	i := img.index(x, y)
	return img.Pix[i+0], img.Pix[i+1], img.Pix[i+2]
}

// WrapRGB returns a sample using periodic/tileable coordinates.
func (img *RGBImage) WrapRGB(x, y int) (r, g, b float64) {
	x = Mod(x, img.Width)
	y = Mod(y, img.Height)
	return img.RGB(x, y)
}

func (img *RGBImage) Clone() *RGBImage {
	clone := NewRGBImage(img.Width, img.Height)
	copy(clone.Pix, img.Pix)
	clone.Linear = img.Linear
	return clone
}

func (img *RGBImage) FillGray(value float64) {
	for i := 0; i < len(img.Pix); i += 3 {
		img.Pix[i+0] = value
		img.Pix[i+1] = value
		img.Pix[i+2] = value
	}
}

// Mean returns the average linear-light RGB color.
func (img *RGBImage) Mean() [3]float64 {
	var sum [3]float64
	pixels := float64(img.Width * img.Height)
	for i := 0; i < len(img.Pix); i += 3 {
		sum[0] += img.Pix[i+0]
		sum[1] += img.Pix[i+1]
		sum[2] += img.Pix[i+2]
	}
	return [3]float64{sum[0] / pixels, sum[1] / pixels, sum[2] / pixels}
}

func (img *RGBImage) MeanLuminance() float64 {
	mean := img.Mean()
	return Luminance(mean)
}

func Luminance(rgb [3]float64) float64 {
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}

func Mod(value, period int) int {
	if period <= 0 {
		return 0
	}
	result := value % period
	if result < 0 {
		result += period
	}
	return result
}

func sRGBToLinear(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func linearToSRGB(value float64) float64 {
	if value <= 0.0031308 {
		return value * 12.92
	}
	return 1.055*math.Pow(value, 1/2.4) - 0.055
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// FromImageLinear decodes and linearizes an sRGB image.
func FromImageLinear(src image.Image) *RGBImage {
	bounds := src.Bounds()
	img := NewLinearRGBImage(bounds.Dx(), bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r32, g32, b32, _ := src.At(x, y).RGBA()
			r := sRGBToLinear(float64(r32) / 65535.0)
			g := sRGBToLinear(float64(g32) / 65535.0)
			b := sRGBToLinear(float64(b32) / 65535.0)
			img.Set(x-bounds.Min.X, y-bounds.Min.Y, r, g, b)
		}
	}
	return img
}

// FromImageEncoded keeps decoded values in their gamma-encoded/output space.
func FromImageEncoded(src image.Image) *RGBImage {
	bounds := src.Bounds()
	img := NewRGBImage(bounds.Dx(), bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r32, g32, b32, _ := src.At(x, y).RGBA()
			img.Set(
				x-bounds.Min.X,
				y-bounds.Min.Y,
				float64(r32)/65535.0,
				float64(g32)/65535.0,
				float64(b32)/65535.0,
			)
		}
	}
	return img
}

// FromImage defaults to encoded RGB so metrics match written PNG averages.
func FromImage(src image.Image) *RGBImage {
	return FromImageEncoded(src)
}

// ToSRGBImage encodes the float image as an 8-bit sRGB image.
func (img *RGBImage) ToSRGBImage() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b := img.RGB(x, y)
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(clamp01(linearToSRGB(clamp01(r))) * 255)),
				G: uint8(math.Round(clamp01(linearToSRGB(clamp01(g))) * 255)),
				B: uint8(math.Round(clamp01(linearToSRGB(clamp01(b))) * 255)),
				A: 255,
			})
		}
	}
	return dst
}

// ToEncodedRGBA writes working values directly. Use for encoded-space color
// and data/normal images that must not be gamma converted.
func (img *RGBImage) ToEncodedRGBA() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b := img.RGB(x, y)
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(clamp01(r) * 255)),
				G: uint8(math.Round(clamp01(g) * 255)),
				B: uint8(math.Round(clamp01(b) * 255)),
				A: 255,
			})
		}
	}
	return dst
}

func (img *RGBImage) ToOutputRGBA() *image.RGBA {
	if img.Linear {
		return img.ToSRGBImage()
	}
	return img.ToEncodedRGBA()
}
