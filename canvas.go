package main

import (
	"image"
	"image/color"
)

// RGBImage is a thin wrapper over image.NRGBA so drawing helpers stay
// dependency free and use straight 8-bit sRGB.
type RGBImage struct {
	*image.NRGBA
}

func NewRGBImage(w, h int) *RGBImage {
	return &RGBImage{NRGBA: image.NewNRGBA(image.Rect(0, 0, w, h))}
}

func (m *RGBImage) set(x, y int, c [3]uint8) {
	if x < 0 || y < 0 || x >= m.Bounds().Dx() || y >= m.Bounds().Dy() {
		return
	}
	i := m.PixOffset(x, y)
	m.Pix[i+0] = c[0]
	m.Pix[i+1] = c[1]
	m.Pix[i+2] = c[2]
	m.Pix[i+3] = 255
}

func (m *RGBImage) fill(x, y int, col color.Color) {
	r, g, b, _ := col.RGBA()
	m.set(x, y, [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)})
}

func (m *RGBImage) fillRect(x0, y0, x1, y1 int, col color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			m.fill(x, y, col)
		}
	}
}

func (m *RGBImage) drawVec(x, y int, v Vec3) {
	m.set(x, y, [3]uint8{
		uint8(clamp01(v.R) * 255),
		uint8(clamp01(v.G) * 255),
		uint8(clamp01(v.B) * 255),
	})
}

func (m *RGBImage) border(x0, y0, x1, y1, thick int, col color.Color) {
	m.fillRect(x0, y0, x1, y0+thick, col)
	m.fillRect(x0, y1-thick, x1, y1, col)
	m.fillRect(x0, y0, x0+thick, y1, col)
	m.fillRect(x1-thick, y0, x1, y1, col)
}

// putImage copies src into m at an offset (no alpha blending needed).
func (m *RGBImage) putImage(ox, oy int, src *RGBImage) {
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			i := src.PixOffset(x, y)
			j := m.PixOffset(ox+x, oy+y)
			m.Pix[j+0] = src.Pix[i+0]
			m.Pix[j+1] = src.Pix[i+1]
			m.Pix[j+2] = src.Pix[i+2]
			m.Pix[j+3] = 255
		}
	}
}
