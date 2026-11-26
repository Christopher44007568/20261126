package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

type rgb struct{ r, g, b byte }

var (
	bgColor    = rgb{232, 232, 236}
	inkColor   = rgb{32, 32, 36}
	frameColor = rgb{96, 96, 104}
	sphereBG   = rgb{64, 64, 72}
	blackColor = rgb{0, 0, 0}
	whiteBar   = rgb{236, 236, 240}
)

func newCanvas(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{
		R: bgColor.r, G: bgColor.g, B: bgColor.b, A: 255,
	}}, image.Point{}, draw.Src)
	return img
}

func fillRect(img *image.NRGBA, x, y, w, h int, c rgb) {
	rect := image.Rect(x, y, x+w, y+h)
	draw.Draw(img, rect, &image.Uniform{C: color.NRGBA{
		R: c.r, G: c.g, B: c.b, A: 255,
	}}, image.Point{}, draw.Src)
}

func frameRect(img *image.NRGBA, x, y, w, h int, c rgb) {
	fillRect(img, x, y, w, 1, c)
	fillRect(img, x, y+h-1, w, 1, c)
	fillRect(img, x, y, 1, h, c)
	fillRect(img, x+w-1, y, 1, h, c)
}

func textWidth(s string) int { return len(s)*6 - 1 }

func drawText(img *image.NRGBA, x, y int, s string, c rgb) {
	for _, r := range s {
		glyph, ok := font5x7[r]
		if !ok {
			glyph = font5x7[' ']
		}
		for row := 0; row < 7; row++ {
			bits := glyph[row]
			for col := 0; col < 5; col++ {
				if bits&(1<<(4-col)) != 0 {
					img.SetNRGBA(x+col, y+row, color.NRGBA{R: c.r, G: c.g, B: c.b, A: 255})
				}
			}
		}
		x += 6
	}
}

func drawCenteredText(img *image.NRGBA, y int, s string, c rgb) {
	drawText(img, (img.Bounds().Dx()-textWidth(s))/2, y, s, c)
}

// drawGrayTile nearest-samples a scalar channel into a tile, preserving
// quantization banding instead of hiding it through interpolation.
func drawGrayTile(img *image.NRGBA, ox, oy, tw, th, w, h int, pix []byte, offset, stride int) {
	for ty := 0; ty < th; ty++ {
		sy := ty * (h - 1) / (th - 1)
		for tx := 0; tx < tw; tx++ {
			sx := tx * (w - 1) / (tw - 1)
			v := pix[(sy*w+sx)*stride+offset]
			img.SetNRGBA(ox+tx, oy+ty, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
}

func drawColorTile(img *image.NRGBA, ox, oy, tw, th, w, h int, pix []byte) {
	for ty := 0; ty < th; ty++ {
		sy := ty * (h - 1) / (th - 1)
		for tx := 0; tx < tw; tx++ {
			sx := tx * (w - 1) / (tw - 1)
			i := (sy*w + sx) * 3
			img.SetNRGBA(ox+tx, oy+ty, color.NRGBA{
				R: pix[i], G: pix[i+1], B: pix[i+2], A: 255,
			})
		}
	}
}

// heat maps an absolute byte difference to a black -> red -> yellow -> white
// ramp. A gain makes single-level differences visible in printed maps.
func heat(diff, gain int) rgb {
	t := diff * gain
	if t > 255 {
		t = 255
	}
	switch {
	case t == 0:
		return blackColor
	case t < 85:
		return rgb{byte(t * 3), 0, 0}
	case t < 170:
		k := t - 85
		return rgb{255, byte(k * 3), 0}
	default:
		k := t - 170
		return rgb{255, 255, byte(k * 3)}
	}
}

func drawDiffTile(img *image.NRGBA, ox, oy, tw, th, w, h int, ref, got []byte, offset, stride, gain int) {
	for ty := 0; ty < th; ty++ {
		sy := ty * (h - 1) / (th - 1)
		for tx := 0; tx < tw; tx++ {
			sx := tx * (w - 1) / (tw - 1)
			i := (sy*w+sx)*stride + offset
			d := int(ref[i]) - int(got[i])
			if d < 0 {
				d = -d
			}
			c := heat(d, gain)
			img.SetNRGBA(ox+tx, oy+ty, color.NRGBA{R: c.r, G: c.g, B: c.b, A: 255})
		}
	}
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
