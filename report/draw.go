package report

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

var (
	colorBackground = color.RGBA{R: 13, G: 18, B: 26, A: 255}
	colorPanel      = color.RGBA{R: 24, G: 32, B: 44, A: 255}
	colorGrid       = color.RGBA{R: 49, G: 62, B: 78, A: 255}
	colorText       = color.RGBA{R: 225, G: 232, B: 240, A: 255}
	colorMuted      = color.RGBA{R: 139, G: 154, B: 173, A: 255}
	colorGreen      = color.RGBA{R: 74, G: 222, B: 128, A: 255}
	colorCyan       = color.RGBA{R: 80, G: 195, B: 255, A: 255}
	colorOrange     = color.RGBA{R: 255, G: 177, B: 66, A: 255}
	colorRed        = color.RGBA{R: 242, G: 92, B: 92, A: 255}
)

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func blendFill(img *image.RGBA, r image.Rectangle, c color.RGBA, alpha uint8) {
	r = r.Intersect(img.Bounds())
	a := uint32(alpha)
	inv := 255 - a
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			old := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((uint32(c.R)*a + uint32(old.R)*inv) / 255),
				G: uint8((uint32(c.G)*a + uint32(old.G)*inv) / 255),
				B: uint8((uint32(c.B)*a + uint32(old.B)*inv) / 255),
				A: 255,
			})
		}
	}
}

func stroke(img *image.RGBA, r image.Rectangle, c color.RGBA, width int) {
	for i := 0; i < width; i++ {
		rr := image.Rect(r.Min.X+i, r.Min.Y+i, r.Max.X-i, r.Max.Y-i)
		if rr.Empty() {
			return
		}
		drawLine(img, rr.Min.X, rr.Min.Y, rr.Max.X-1, rr.Min.Y, c)
		drawLine(img, rr.Min.X, rr.Max.Y-1, rr.Max.X-1, rr.Max.Y-1, c)
		drawLine(img, rr.Min.X, rr.Min.Y, rr.Min.X, rr.Max.Y-1, c)
		drawLine(img, rr.Max.X-1, rr.Min.Y, rr.Max.X-1, rr.Max.Y-1, c)
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		set(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func dottedVertical(img *image.RGBA, x, y0, y1 int, c color.RGBA, on, off int) {
	for y := y0; y <= y1; y++ {
		if (y-y0)%(on+off) < on {
			set(img, x, y, c)
			set(img, x+1, y, c)
		}
	}
}

func set(img *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetRGBA(x, y, c)
	}
}

func resize(src image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return dst
	}
	for y := 0; y < height; y++ {
		fy := (float64(y)+0.5)*float64(b.Dy())/float64(height) - 0.5
		y0 := clampInt(int(math.Floor(fy)), 0, b.Dy()-1)
		y1 := clampInt(y0+1, 0, b.Dy()-1)
		wy := fy - math.Floor(fy)
		for x := 0; x < width; x++ {
			fx := (float64(x)+0.5)*float64(b.Dx())/float64(width) - 0.5
			x0 := clampInt(int(math.Floor(fx)), 0, b.Dx()-1)
			x1 := clampInt(x0+1, 0, b.Dx()-1)
			wx := fx - math.Floor(fx)
			c00 := src.At(b.Min.X+x0, b.Min.Y+y0)
			c10 := src.At(b.Min.X+x1, b.Min.Y+y0)
			c01 := src.At(b.Min.X+x0, b.Min.Y+y1)
			c11 := src.At(b.Min.X+x1, b.Min.Y+y1)
			r := bilinear(c00, c10, c01, c11, wx, wy, 0)
			g := bilinear(c00, c10, c01, c11, wx, wy, 1)
			bl := bilinear(c00, c10, c01, c11, wx, wy, 2)
			a := bilinear(c00, c10, c01, c11, wx, wy, 3)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: bl, A: a})
		}
	}
	return dst
}

func bilinear(c00, c10, c01, c11 color.Color, wx, wy float64, channel int) uint8 {
	r00, g00, b00, a00 := c00.RGBA()
	r10, g10, b10, a10 := c10.RGBA()
	r01, g01, b01, a01 := c01.RGBA()
	r11, g11, b11, a11 := c11.RGBA()
	var top, bottom, value uint32
	switch channel {
	case 0:
		top = blend16(r00, r10, wx)
		bottom = blend16(r01, r11, wx)
		value = blend16(top, bottom, wy)
	case 1:
		top = blend16(g00, g10, wx)
		bottom = blend16(g01, g11, wx)
		value = blend16(top, bottom, wy)
	case 2:
		top = blend16(b00, b10, wx)
		bottom = blend16(b01, b11, wx)
		value = blend16(top, bottom, wy)
	default:
		top = blend16(a00, a10, wx)
		bottom = blend16(a01, a11, wx)
		value = blend16(top, bottom, wy)
	}
	return uint8(value >> 8)
}

func blend16(a, b uint32, weight float64) uint32 {
	return uint32(float64(a)*(1-weight) + float64(b)*weight)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
