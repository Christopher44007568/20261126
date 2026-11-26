package main

import (
	"image"
	"image/color"
	"math"
)

func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// MakeMaterialTexture builds a deterministic, high-contrast material swatch.
// The same pixels are used by both normal and stale-binding frames, which makes
// a wrong camera projection immediately visible.
func MakeMaterialTexture(size int) *image.RGBA {
	if size < 128 {
		size = 128
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) / float64(size-1)
			fy := float64(y) / float64(size-1)
			base := mixColor(
				color.RGBA{R: 8, G: 25, B: 43, A: 255},
				color.RGBA{R: 17, G: 53, B: 75, A: 255},
				0.42*fx+0.58*fy,
			)
			if ((x/(size/8))+(y/(size/8)))%2 == 0 {
				base = mixColor(base, color.RGBA{R: 32, G: 75, B: 100, A: 255}, 0.25)
			}

			diagA := math.Abs(float64((x+2*y)%128) - 64)
			if diagA < 15 {
				t := 1 - diagA/15
				base = mixColor(base, color.RGBA{R: 34, G: 211, B: 229, A: 255}, 0.62*t)
			}
			diagB := math.Abs(float64((2*x+y+43)%144) - 72)
			if diagB < 11 {
				t := 1 - diagB/11
				base = mixColor(base, color.RGBA{R: 245, G: 66, B: 145, A: 255}, 0.58*t)
			}

			grid := size / 8
			if x%grid == 0 || y%grid == 0 {
				base = mixColor(base, color.RGBA{R: 120, G: 158, B: 184, A: 255}, 0.34)
			}
			dx, dy := x-size/2, y-size/2
			r := math.Hypot(float64(dx), float64(dy))
			if math.Abs(r-float64(size)*0.34) < 2.5 {
				base = mixColor(base, color.RGBA{R: 244, G: 247, B: 252, A: 255}, 0.72)
			}
			img.SetRGBA(x, y, base)
		}
	}

	drawRect(img, 2, 2, size-4, size-4, 5, colCyan)
	drawRect(img, 10, 10, size-20, size-20, 1, color.RGBA{R: 190, G: 226, B: 240, A: 255})

	plateX := size / 9
	plateY := size*3/8 - 8
	plateW := size - 2*plateX
	plateH := size / 4
	fillRect(img, plateX, plateY, plateW, plateH, color.RGBA{R: 5, G: 12, B: 22, A: 214})
	drawRect(img, plateX, plateY, plateW, plateH, 2, color.RGBA{R: 219, G: 238, B: 248, A: 235})
	drawTextCentered(img, size/2, plateY+plateH/2-18, "TEX V1", 5, colWhite)

	marker := size / 20
	fillRect(img, 18, 18, marker, marker, colCyan)
	fillRect(img, size-18-marker, 18, marker, marker, colAmber)
	fillRect(img, 18, size-18-marker, marker, marker, colMagenta)
	fillRect(img, size-18-marker, size-18-marker, marker, marker, colGreen)
	return img
}

// sampleTexture uses clamped bilinear filtering. u and v are in [0,1].
func sampleTexture(img *image.RGBA, u, v float64) color.RGBA {
	if img == nil || img.Bounds().Empty() {
		return color.RGBA{}
	}
	if u < 0 {
		u = 0
	} else if u > 1 {
		u = 1
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	fx := u * float64(w-1)
	fy := v * float64(h-1)
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	x1, y1 := x0+1, y0+1
	if x1 >= w {
		x1 = w - 1
	}
	if y1 >= h {
		y1 = h - 1
	}
	tx, ty := fx-float64(x0), fy-float64(y0)
	c00 := img.RGBAAt(x0, y0)
	c10 := img.RGBAAt(x1, y0)
	c01 := img.RGBAAt(x0, y1)
	c11 := img.RGBAAt(x1, y1)
	lerp := func(a, b uint8, t float64) uint8 {
		return clampByte(float64(a)*(1-t) + float64(b)*t)
	}
	top := color.RGBA{
		R: lerp(c00.R, c10.R, tx),
		G: lerp(c00.G, c10.G, tx),
		B: lerp(c00.B, c10.B, tx),
		A: lerp(c00.A, c10.A, tx),
	}
	bottom := color.RGBA{
		R: lerp(c01.R, c11.R, tx),
		G: lerp(c01.G, c11.G, tx),
		B: lerp(c01.B, c11.B, tx),
		A: lerp(c01.A, c11.A, tx),
	}
	return color.RGBA{
		R: lerp(top.R, bottom.R, ty),
		G: lerp(top.G, bottom.G, ty),
		B: lerp(top.B, bottom.B, ty),
		A: lerp(top.A, bottom.A, ty),
	}
}
