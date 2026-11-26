package render

import (
	"image"
	"image/color"
	"math"
)

type Canvas struct {
	*image.RGBA
	W int
	H int
}

func NewCanvas(w, h int, bg color.RGBA) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := &Canvas{RGBA: img, W: w, H: h}
	c.Fill(bg)
	return c
}

func (c *Canvas) Fill(col color.RGBA) {
	for y := 0; y < c.H; y++ {
		off := y * c.Stride
		for x := 0; x < c.W; x++ {
			i := off + x*4
			c.Pix[i+0] = col.R
			c.Pix[i+1] = col.G
			c.Pix[i+2] = col.B
			c.Pix[i+3] = 255
		}
	}
}

func (c *Canvas) Set(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := y*c.Stride + x*4
	c.Pix[i+0] = col.R
	c.Pix[i+1] = col.G
	c.Pix[i+2] = col.B
	c.Pix[i+3] = 255
}

func (c *Canvas) Blend(x, y int, col color.RGBA, alpha uint8) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || alpha == 0 {
		return
	}
	a := float64(alpha) / 255
	i := y*c.Stride + x*4
	c.Pix[i+0] = uint8(float64(c.Pix[i+0])*(1-a) + float64(col.R)*a)
	c.Pix[i+1] = uint8(float64(c.Pix[i+1])*(1-a) + float64(col.G)*a)
	c.Pix[i+2] = uint8(float64(c.Pix[i+2])*(1-a) + float64(col.B)*a)
}

func (c *Canvas) Rect(x, y, w, h int, col color.RGBA) {
	x0, y0 := max(0, x), max(0, y)
	x1, y1 := min(c.W, x+w), min(c.H, y+h)
	for yy := y0; yy < y1; yy++ {
		off := yy * c.Stride
		for xx := x0; xx < x1; xx++ {
			i := off + xx*4
			c.Pix[i+0] = col.R
			c.Pix[i+1] = col.G
			c.Pix[i+2] = col.B
			c.Pix[i+3] = 255
		}
	}
}

func (c *Canvas) Stroke(x, y, w, h, thick int, col color.RGBA) {
	c.Rect(x, y, w, thick, col)
	c.Rect(x, y+h-thick, w, thick, col)
	c.Rect(x, y, thick, h, col)
	c.Rect(x+w-thick, y, thick, h, col)
}

func (c *Canvas) HLine(x0, x1, y int, col color.RGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	c.Rect(x0, y, x1-x0+1, 1, col)
}

func (c *Canvas) VLine(x, y0, y1 int, col color.RGBA) {
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	c.Rect(x, y0, 1, y1-y0+1, col)
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col color.RGBA) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		c.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
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

func (c *Canvas) DashedV(x, y0, y1, dash, gap int, col color.RGBA) {
	for y := y0; y <= y1; y += dash + gap {
		c.VLine(x, y, min(y+dash-1, y1), col)
	}
}

func (c *Canvas) DashedH(x0, x1, y, dash, gap int, col color.RGBA) {
	for x := x0; x <= x1; x += dash + gap {
		c.HLine(x, min(x+dash-1, x1), y, col)
	}
}

func (c *Canvas) Triangle(cx, y, half, height int, col color.RGBA) {
	for row := 0; row < height; row++ {
		halfRow := half * (height - row) / height
		c.HLine(cx-halfRow, cx+halfRow, y+row, col)
	}
}

func (c *Canvas) Pill(x, y, w, h int, fill, border color.RGBA) {
	c.Rect(x+1, y, w-2, h, border)
	c.Rect(x, y+1, w, h-2, border)
	c.Rect(x+1, y+1, w-2, h-2, fill)
}

func (c *Canvas) Hatch(x, y, w, h, spacing int, col color.RGBA, alpha uint8) {
	x0, y0 := max(0, x), max(0, y)
	x1, y1 := min(c.W, x+w), min(c.H, y+h)
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			if (xx+yy)%spacing == 0 {
				c.Blend(xx, yy, col, alpha)
			}
		}
	}
}

func (c *Canvas) Checker(x, y, w, h, cell int, a, b color.RGBA) {
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			col := a
			if ((xx/cell)+(yy/cell))&1 == 1 {
				col = b
			}
			c.Set(x+xx, y+yy, col)
		}
	}
}

func heatColor(t float64) color.RGBA {
	t = Clamp01(t)
	type stop struct {
		t float64
		c color.RGBA
	}
	stops := []stop{
		{0.00, color.RGBA{8, 15, 31, 255}},
		{0.18, color.RGBA{28, 48, 96, 255}},
		{0.42, color.RGBA{127, 34, 160, 255}},
		{0.66, color.RGBA{241, 77, 75, 255}},
		{0.84, color.RGBA{252, 176, 68, 255}},
		{1.00, color.RGBA{255, 248, 220, 255}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			u := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
			return color.RGBA{
				R: uint8(Lerp(float64(stops[i-1].c.R), float64(stops[i].c.R), u)),
				G: uint8(Lerp(float64(stops[i-1].c.G), float64(stops[i].c.G), u)),
				B: uint8(Lerp(float64(stops[i-1].c.B), float64(stops[i].c.B), u)),
				A: 255,
			}
		}
	}
	return stops[len(stops)-1].c
}

func rgb8(c RGB) color.RGBA {
	return color.RGBA{
		R: uint8(ClampCode(int(math.Floor(c.R + 0.5)))),
		G: uint8(ClampCode(int(math.Floor(c.G + 0.5)))),
		B: uint8(ClampCode(int(math.Floor(c.B + 0.5)))),
		A: 255,
	}
}

func alpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
