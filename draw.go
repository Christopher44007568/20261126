package hdrlab

import "math"

// Blend alpha-composites a color over the existing pixel.
func (im *Img) Blend(x, y int, r, g, b, a float64) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H || a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := (y*im.W + x) * 3
	im.Pix[i] += (r - im.Pix[i]) * a
	im.Pix[i+1] += (g - im.Pix[i+1]) * a
	im.Pix[i+2] += (b - im.Pix[i+2]) * a
}

func (im *Img) FillRect(x0, y0, x1, y1 int, r, g, b float64) {
	im.BlendRect(x0, y0, x1, y1, r, g, b, 1)
}

// BlendRect fills [x0,x1) x [y0,y1) with an alpha-blended color.
func (im *Img) BlendRect(x0, y0, x1, y1 int, r, g, b, a float64) {
	x0 = clampInt(x0, 0, im.W)
	x1 = clampInt(x1, 0, im.W)
	y0 = clampInt(y0, 0, im.H)
	y1 = clampInt(y1, 0, im.H)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			im.Blend(x, y, r, g, b, a)
		}
	}
}

func (im *Img) Rect(x0, y0, x1, y1 int, r, g, b, a float64) {
	for x := x0; x < x1; x++ {
		im.Blend(x, y0, r, g, b, a)
		im.Blend(x, y1-1, r, g, b, a)
	}
	for y := y0; y < y1; y++ {
		im.Blend(x0, y, r, g, b, a)
		im.Blend(x1-1, y, r, g, b, a)
	}
}

// Line draws a 1px Bresenham line.
func (im *Img) Line(x0, y0, x1, y1 int, r, g, b, a float64) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		im.Blend(x0, y0, r, g, b, a)
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

// Arrow draws a line with a small arrowhead at (x1,y1).
func (im *Img) Arrow(x0, y0, x1, y1 int, r, g, b, a float64) {
	im.Line(x0, y0, x1, y1, r, g, b, a)
	ang := math.Atan2(float64(y1-y0), float64(x1-x0))
	for _, da := range []float64{2.6, -2.6} {
		ax := x1 + int(math.Cos(ang+da)*6+0.5)
		ay := y1 + int(math.Sin(ang+da)*6+0.5)
		im.Line(x1, y1, ax, ay, r, g, b, a)
	}
}

// Ellipse draws an ellipse outline centered at (cx,cy).
func (im *Img) Ellipse(cx, cy, rx, ry int, r, g, b, a float64) {
	if rx < 1 {
		rx = 1
	}
	if ry < 1 {
		ry = 1
	}
	steps := 4 * (rx + ry)
	for i := 0; i < steps; i++ {
		t := 2 * math.Pi * float64(i) / float64(steps)
		x := cx + int(math.Cos(t)*float64(rx)+0.5)
		y := cy + int(math.Sin(t)*float64(ry)+0.5)
		im.Blend(x, y, r, g, b, a)
	}
}

func (im *Img) FillEllipse(cx, cy, rx, ry int, r, g, b float64) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 {
				if x >= 0 && y >= 0 && x < im.W && y < im.H {
					im.Set(x, y, r, g, b)
				}
			}
		}
	}
}

// Hatch draws diagonal 45-degree lines inside [x0,x1) x [y0,y1).
func (im *Img) Hatch(x0, y0, x1, y1, spacing int, r, g, b, a float64) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if (x+y)%spacing == 0 {
				im.Blend(x, y, r, g, b, a)
			}
		}
	}
}
