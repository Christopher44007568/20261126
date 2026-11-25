package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// SavePNG writes an image to disk, creating parent directories.
func SavePNG(img image.Image, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// MeterOverlay renders the exposed photo tinted by the metering weight
// map: high-weight pixels glow red, so the metered region is visible.
// A crosshair marks the spot point when drawSpot is true.
func MeterOverlay(photo *image.NRGBA, weights []float64, spotX, spotY int, drawSpot bool) *image.NRGBA {
	b := photo.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(b)
	maxW := 0.0
	for _, v := range weights {
		if v > maxW {
			maxW = v
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			src := photo.NRGBAAt(x, y)
			a := 0.0
			if maxW > 0 {
				a = math.Sqrt(weights[i] / maxW) // perceptual boost
			}
			blend := 0.55 * a
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8(float64(src.R)*(1-blend) + 255*blend),
				G: uint8(float64(src.G) * (1 - blend)),
				B: uint8(float64(src.B) * (1 - blend)),
				A: 255,
			})
		}
	}
	if drawSpot {
		drawCross(out, spotX, spotY, 8, color.NRGBA{0, 255, 255, 255})
	}
	return out
}

func drawCross(img *image.NRGBA, cx, cy, r int, c color.NRGBA) {
	b := img.Bounds()
	for d := -r; d <= r; d++ {
		if p := image.Pt(cx+d, cy); p.In(b) {
			img.SetNRGBA(p.X, p.Y, c)
		}
		if p := image.Pt(cx, cy+d); p.In(b) {
			img.SetNRGBA(p.X, p.Y, c)
		}
	}
}

// Histogram renders a 256-bin luminance histogram of the photo.
func Histogram(photo *image.NRGBA, w, h int) *image.NRGBA {
	bins := make([]int, 256)
	b := photo.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := photo.NRGBAAt(x, y)
			lum := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
			bin := int(lum)
			if bin > 255 {
				bin = 255
			}
			bins[bin]++
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bg := color.NRGBA{24, 24, 28, 255}
	for i := 0; i < w*h; i++ {
		img.SetNRGBA(i%w, i/w, bg)
	}
	maxBin := 1
	for _, v := range bins {
		if v > maxBin {
			maxBin = v
		}
	}
	bar := color.NRGBA{235, 235, 240, 255}
	mid := color.NRGBA{90, 200, 120, 255}
	midX := w * 46 / 256 // ~18% gray sits near sRGB code 117/255
	for x := 0; x < w; x++ {
		bin := x * 256 / w
		barH := int(float64(bins[bin]) / float64(maxBin) * float64(h-8))
		for y := h - 1; y >= h-barH; y-- {
			img.SetNRGBA(x, y, bar)
		}
	}
	for y := 0; y < h; y++ {
		img.SetNRGBA(midX, y, mid)
	}
	return img
}
