package main

import "math"

const textureSize = 64

type Scenario struct {
	ID       string
	Title    string
	Original *Bitmap
}

func GenerateScenarios() []Scenario {
	return []Scenario{
		{ID: "smooth", Title: "Smooth gradient", Original: smoothTexture()},
		{ID: "edges", Title: "Sharp edges", Original: sharpEdgesTexture()},
		{ID: "holes", Title: "Transparent holes", Original: transparentHolesTexture()},
	}
}

func smoothTexture() *Bitmap {
	im := NewBitmap(textureSize, textureSize)
	for y := 0; y < im.H; y++ {
		v := float64(y) / float64(im.H-1)
		for x := 0; x < im.W; x++ {
			u := float64(x) / float64(im.W-1)
			dx, dy := u-0.5, v-0.5
			d := math.Sqrt(dx*dx+dy*dy) / 0.7071
			im.Set(x, y, Pixel{
				R: round8(34 + 188*u + 26*v),
				G: round8(28 + 150*v + 55*(1-d)),
				B: round8(42 + 205*v + 24*math.Sin(u*math.Pi)),
				A: 255,
			})
		}
	}
	return im
}

func sharpEdgesTexture() *Bitmap {
	im := NewBitmap(textureSize, textureSize)
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			var p Pixel
			switch {
			case y < 22 && x < 25:
				p = Pixel{R: 245, G: 236, B: 210, A: 255}
			case y < 22:
				p = Pixel{R: 14, G: 172, B: 210, A: 255}
			case x < 25:
				p = Pixel{R: 214, G: 54, B: 46, A: 255}
			case y > 44:
				p = Pixel{R: 230, G: 176, B: 28, A: 255}
			default:
				p = Pixel{R: 20, G: 28, B: 58, A: 255}
			}
			if absInt(x+y-61) <= 1 {
				p = Pixel{R: 248, G: 248, B: 240, A: 255}
			}
			if x >= 40 && x <= 43 && y >= 27 && y <= 36 {
				p = Pixel{R: 16, G: 28, B: 22, A: 255}
			}
			im.Set(x, y, p)
		}
	}
	return im
}

func transparentHolesTexture() *Bitmap {
	im := NewBitmap(textureSize, textureSize)
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			checker := (x/8 + y/8) & 1
			p := Pixel{
				R: uint8(48 + x*2 + checker*38),
				G: uint8(92 + y*2 + checker*24),
				B: uint8(188 + checker*40),
				A: 255,
			}
			if (x-16)*(x-16)+(y-44)*(y-44) <= 11*11 ||
				(x-48)*(x-48)+(y-16)*(y-16) <= 9*9 ||
				(x >= 4 && x <= 13 && y >= 25 && y <= 32) ||
				(x <= 7 && y >= 54) {
				p.A = 0
			}
			im.Set(x, y, p)
		}
	}
	return im
}

func round8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
