package render

import (
	"image/color"

	"deshake/internal/stab"
)

// DrawRGB blits a linear RGB float image onto a new canvas.
func DrawRGB(pix []float64, w, h int) *Canvas {
	c := New(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 3
			c.Img.Set(x, y, color.RGBA{
				R: to8(pix[i]),
				G: to8(pix[i+1]),
				B: to8(pix[i+2]),
				A: 255,
			})
		}
	}
	return c
}

// DrawResult renders a stabilized frame, painting uncovered pixels as hatched
// magenta holes and overlaying the maximal valid crop rectangle.
func DrawResult(fr *stab.FrameResult, showBox bool) *Canvas {
	c := New(fr.W, fr.H)
	base := color.RGBA{R: 60, G: 8, B: 36, A: 255}
	c.Hatch(func(x, y int) bool { return fr.Hole[y*fr.W+x] }, base)
	for y := 0; y < fr.H; y++ {
		for x := 0; x < fr.W; x++ {
			i := y*fr.W + x
			if fr.Hole[i] {
				continue
			}
			j := i * 3
			c.Img.Set(x, y, color.RGBA{
				R: to8(fr.Pix[j]),
				G: to8(fr.Pix[j+1]),
				B: to8(fr.Pix[j+2]),
				A: 255,
			})
		}
	}
	if showBox {
		green := color.RGBA{R: 40, G: 230, B: 90, A: 255}
		b := fr.Box
		c.RectStroke(b.X0, b.Y0, b.X1-1, b.Y1-1, green)
	}
	return c
}

func to8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v * 255)
}
