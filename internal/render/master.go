package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

const (
	masterW = 1280
	masterH = 486
)

func renderMaster(bits int) (image.Image, error) {
	img := image.NewNRGBA64(image.Rect(0, 0, masterW, masterH))
	fill16(img, color.NRGBA64{R: 0x0500, G: 0x0d00, B: 0x1a00, A: 0xffff})
	fillRect16(img, 0, 0, masterW, 5, color.NRGBA64{R: 0x3e00, G: 0xda00, B: 0xeb00, A: 0xffff})
	drawText16(img, 28, 22, 3, fmt.Sprintf("%d-BIT MASTER", bits), color.NRGBA64{R: 0xffff, G: 0xffff, B: 0xffff, A: 0xffff})
	drawText16(img, 28, 50, 1,
		"STORED AS 16-BIT PNG / NORMALIZED INTEGER LEVELS / NO 8-BIT DISPLAY DITHER",
		color.NRGBA64{R: 0xb000, G: 0xd000, B: 0xe000, A: 0xffff})

	const (
		x      = 28
		w      = masterW - 56
		bandH  = 88
		rowGap = 28
		y0     = 92
	)
	for i, sc := range Scenes {
		y := y0 + i*(bandH+rowGap)
		label := fmt.Sprintf("%02d / %s / %s", i+1, sc.Title, sc.Subtitle)
		drawText16(img, x, y-12, 1, label, color.NRGBA64{R: 0xd800, G: 0xe800, B: 0xf400, A: 0xffff})
		for yy := 0; yy < bandH; yy++ {
			for xx := 0; xx < w; xx++ {
				src := sourceAt(sc, xx, yy, w, bandH)
				img.SetNRGBA64(x+xx, y+yy, color.NRGBA64{
					R: code16(src.R, bits),
					G: code16(src.G, bits),
					B: code16(src.B, bits),
					A: 0xffff,
				})
			}
		}
		stroke16(img, x, y, w, bandH, color.NRGBA64{R: 0x4000, G: 0x5800, B: 0x7000, A: 0xffff})
	}
	drawText16(img, 28, masterH-24, 1,
		"EACH MASTER LEVEL IS EXACT FOR THE SELECTED DEPTH. 8-BIT SHEETS ARE GENERATED FROM THESE VALUES WITH 4X4 BAYER.",
		color.NRGBA64{R: 0x8000, G: 0x9400, B: 0xaa00, A: 0xffff})
	return img, nil
}

func code16(v float64, bits int) uint16 {
	q := Quantize(v, bits)
	return uint16(math.Floor(q*65535 + 0.5))
}

func fill16(img *image.NRGBA64, col color.NRGBA64) {
	fillRect16(img, 0, 0, img.Bounds().Dx(), img.Bounds().Dy(), col)
}

func fillRect16(img *image.NRGBA64, x, y, w, h int, col color.NRGBA64) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			img.SetNRGBA64(xx, yy, col)
		}
	}
}

func stroke16(img *image.NRGBA64, x, y, w, h int, col color.NRGBA64) {
	fillRect16(img, x, y, w, 1, col)
	fillRect16(img, x, y+h-1, w, 1, col)
	fillRect16(img, x, y, 1, h, col)
	fillRect16(img, x+w-1, y, 1, h, col)
}

func drawText16(img *image.NRGBA64, x, y, scale int, text string, col color.NRGBA64) {
	cursor := x
	for _, r := range text {
		g, ok := glyphs[r]
		if !ok {
			g = glyphs[' ']
		}
		for gy, row := range g {
			for gx, ch := range row {
				if ch == '#' {
					fillRect16(img, cursor+gx*scale, y+gy*scale, scale, scale, col)
				}
			}
		}
		cursor += 6 * scale
	}
}
