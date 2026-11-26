package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

type colorRGBA = color.RGBA

// Canvas is an RGBA pixel buffer with alpha-aware drawing primitives.
type Canvas struct {
	Img  *image.RGBA
	Clip image.Rectangle
}

func NewCanvas(w, h int, bg color.RGBA) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	c := &Canvas{Img: img, Clip: image.Rect(0, 0, w, h)}
	c.FillRect(0, 0, w, h, bg)
	return c
}

func (c *Canvas) W() int { return c.Img.Bounds().Dx() }
func (c *Canvas) H() int { return c.Img.Bounds().Dy() }

// set blends one pixel using source-over compositing.
func (c *Canvas) set(x, y int, src color.RGBA) {
	if x < c.Clip.Min.X || y < c.Clip.Min.Y || x >= c.Clip.Max.X || y >= c.Clip.Max.Y {
		return
	}
	if x < 0 || y < 0 || x >= c.W() || y >= c.H() {
		return
	}
	i := c.Img.PixOffset(x, y)
	a := uint32(src.A)
	if a == 255 {
		c.Img.Pix[i+0] = src.R
		c.Img.Pix[i+1] = src.G
		c.Img.Pix[i+2] = src.B
		c.Img.Pix[i+3] = 255
		return
	}
	if a == 0 {
		return
	}
	dr := uint32(c.Img.Pix[i+0])
	dg := uint32(c.Img.Pix[i+1])
	db := uint32(c.Img.Pix[i+2])
	da := uint32(c.Img.Pix[i+3])
	sa := a
	outA := sa + da*(255-sa)/255
	if outA == 0 {
		return
	}
	br := (uint32(src.R)*sa + dr*da*(255-sa)/255) / outA
	bg := (uint32(src.G)*sa + dg*da*(255-sa)/255) / outA
	bb := (uint32(src.B)*sa + db*da*(255-sa)/255) / outA
	c.Img.Pix[i+0] = uint8(br)
	c.Img.Pix[i+1] = uint8(bg)
	c.Img.Pix[i+2] = uint8(bb)
	c.Img.Pix[i+3] = uint8(outA)
}

func withAlpha(col color.RGBA, a uint8) color.RGBA {
	col.A = a
	return col
}

// FillRect fills an axis aligned rectangle.
func (c *Canvas) FillRect(x, y, w, h int, col color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.set(xx, yy, col)
		}
	}
}

// VGradient paints a vertical linear gradient between two opaque colours.
func (c *Canvas) VGradient(x, y, w, h int, top, bottom color.RGBA) {
	for yy := 0; yy < h; yy++ {
		t := 0
		if h > 1 {
			t = yy * 255 / (h - 1)
		}
		col := lerpColor(top, bottom, t)
		c.FillRect(x, y+yy, w, 1, col)
	}
}

func lerpColor(a, b color.RGBA, t int) color.RGBA {
	return color.RGBA{
		R: uint8((int(a.R)*(255-t) + int(b.R)*t) / 255),
		G: uint8((int(a.G)*(255-t) + int(b.G)*t) / 255),
		B: uint8((int(a.B)*(255-t) + int(b.B)*t) / 255),
		A: 255,
	}
}

func insideRR(px, py float64, x, y, w, h int, r float64) bool {
	fx0, fy0 := float64(x), float64(y)
	fw, fh := float64(w), float64(h)
	if r <= 0 {
		return px >= fx0 && px < fx0+fw && py >= fy0 && py < fy0+fh
	}
	cx := px
	cy := py
	if px < fx0+r {
		cx = fx0 + r
	} else if px > fx0+fw-1-r {
		cx = fx0 + fw - 1 - r
	}
	if py < fy0+r {
		cy = fy0 + r
	} else if py > fy0+fh-1-r {
		cy = fy0 + fh - 1 - r
	}
	dx := px - cx
	dy := py - cy
	return dx*dx+dy*dy <= r*r
}

// FillRoundRect fills a rounded rectangle with 3x3 edge supersampling.
func (c *Canvas) FillRoundRect(x, y, w, h, r int, col color.RGBA) {
	const ss = 3
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					if insideRR(fx, fy, x, y, w, h, float64(r)) {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			alpha := uint32(col.A) * uint32(hit) / (ss * ss)
			cc := col
			cc.A = uint8(alpha)
			c.set(px, py, cc)
		}
	}
}

// StrokeRoundRect draws a rounded-rectangle outline of the given thickness.
func (c *Canvas) StrokeRoundRect(x, y, w, h, r, thick int, col color.RGBA) {
	const ss = 3
	for py := y - thick; py < y+h+thick; py++ {
		for px := x - thick; px < x+w+thick; px++ {
			hit := 0
			total := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					total++
					outer := insideRR(fx, fy, x, y, w, h, float64(r))
					inner := insideRR(fx, fy, x+thick, y+thick, w-2*thick, h-2*thick, float64(maxInt(r-thick, 0)))
					if outer && !inner {
						hit++
					}
				}
			}
			alpha := uint32(col.A) * uint32(hit) / uint32(total)
			if alpha == 0 {
				continue
			}
			cc := col
			cc.A = uint8(alpha)
			c.set(px, py, cc)
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SavePNG writes the canvas as a PNG.
func (c *Canvas) SavePNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	return enc.Encode(f, c.Img)
}
