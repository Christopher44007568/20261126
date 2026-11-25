package render

import (
	"image/color"

	"deshake/internal/stab"
)

// Series is one labeled polyline on a chart.
type Series struct {
	Name string
	Y    []float64
	Col  color.Color
	Dash bool
}

// MarginChart draws per-frame edge margins (left) and hole rate (right).
// 2D series are dashed, 3D solid; each panel has its own normalized y-scale.
func MarginChart(n int, margin2d, margin3d, hole2d, hole3d []Series) *Canvas {
	w, h := 1000, 470
	c := New(w, h)
	c.Fill(color.RGBA{R: 245, G: 246, B: 248, A: 255})
	plotW, plotH := 420, 280
	margins := append(append([]Series{}, margin2d...), margin3d...)
	holes := append(append([]Series{}, hole2d...), hole3d...)
	c.plot(50, 80, plotW, plotH, 0, float64(n-1), 0, maxY(nil, margins),
		margins, n)
	c.DrawText(50, 26, 2, "EDGE MARGIN PER FRAME (PX)",
		color.RGBA{R: 30, G: 34, B: 42, A: 255})
	c.DrawText(50, 54, 1, "DASHED=2D TRANSLATE   SOLID=3D DE-ROLL",
		color.RGBA{R: 90, G: 96, B: 108, A: 255})
	c.legend(50, 382, margins)

	c.plot(540, 80, plotW, plotH, 0, float64(n-1), 0, maxY(nil, holes),
		holes, n)
	c.DrawText(540, 26, 2, "HOLE RATE PER FRAME (PERCENT)",
		color.RGBA{R: 30, G: 34, B: 42, A: 255})
	c.DrawText(540, 54, 1, "UNCOVERED AREA MARKED AS HOLE, NEVER STRETCHED",
		color.RGBA{R: 90, G: 96, B: 108, A: 255})
	c.legend(540, 382, holes)
	return c
}

// BoundaryChart renders a fast-turn valid mask outline with its straight crop.
func BoundaryChart(fr *stab.FrameResult) *Canvas {
	w, h := 560, 470
	c := New(w, h)
	c.Fill(color.RGBA{R: 245, G: 246, B: 248, A: 255})
	c.DrawText(24, 26, 2, "FAST-TURN VALID BOUNDARY (3D DE-ROLL)",
		color.RGBA{R: 30, G: 34, B: 42, A: 255})
	c.DrawText(24, 54, 1, "RED=WAVY RAW EDGE  GREEN=STRAIGHT SAFE CROP",
		color.RGBA{R: 90, G: 96, B: 108, A: 255})
	c.drawBoundary(40, 80, 480, 360, fr)
	return c
}

func maxY(_ []Series, b []Series) float64 {
	m := 1.0
	for _, s := range b {
		for _, v := range s.Y {
			if v > m {
				m = v
			}
		}
	}
	return m * 1.1
}

func (c *Canvas) plot(x, y, w, h int, xmin, xmax, ymin, ymax float64,
	series []Series, n int) {
	axis := color.RGBA{R: 60, G: 64, B: 72, A: 255}
	grid := color.RGBA{R: 220, G: 222, B: 228, A: 255}
	for g := 0; g <= 4; g++ {
		gy := y + h*g/4
		c.Line(x, gy, x+w, gy, grid)
		val := ymax - (ymax-ymin)*float64(g)/4
		c.DrawText(x-32, gy-3, 1, ftoa(val, 1), axis)
	}
	c.Line(x, y, x, y+h, axis)
	c.Line(x, y+h, x+w, y+h, axis)

	toX := func(i int) int {
		return x + int(float64(i)/(float64(n-1))*float64(w))
	}
	toY := func(v float64) int {
		return y + h - int((v-ymin)/(ymax-ymin)*float64(h))
	}
	for _, s := range series {
		for i := 1; i < len(s.Y); i++ {
			x0, y0 := toX(i-1), toY(s.Y[i-1])
			x1, y1 := toX(i), toY(s.Y[i])
			if s.Dash {
				c.dashedLine(x0, y0, x1, y1, s.Col)
			} else {
				c.Line(x0, y0, x1, y1, s.Col)
			}
		}
	}
}

func (c *Canvas) dashedLine(x0, y0, x1, y1 int, col color.Color) {
	steps := absInt(x1-x0) + absInt(y1-y0)
	if steps == 0 {
		return
	}
	for i := 0; i < steps; i++ {
		if i%7 < 4 {
			t0 := float64(i) / float64(steps)
			t1 := float64(i+1) / float64(steps)
			c.Line(
				x0+int(t0*float64(x1-x0)),
				y0+int(t0*float64(y1-y0)),
				x0+int(t1*float64(x1-x0)),
				y0+int(t1*float64(y1-y0)),
				col,
			)
		}
	}
}

func (c *Canvas) legend(x, y int, series []Series) {
	// Fixed two-column layout so eight margin series fit inside a subplot.
	maxPerRow := 2
	for i, s := range series {
		col := i % maxPerRow
		row := i / maxPerRow
		lx := x + col*215
		ly := y + row*16
		if s.Dash {
			c.dashedLine(lx, ly, lx+22, ly, s.Col)
		} else {
			c.Line(lx, ly, lx+22, ly, s.Col)
		}
		c.DrawText(lx+27, ly-5, 1, s.Name, color.RGBA{R: 40, G: 44, B: 52, A: 255})
	}
}

func (c *Canvas) drawBoundary(x, y, w, h int, fr *stab.FrameResult) {
	if fr == nil {
		return
	}
	left, right, top, bottom := stab.EdgeProfile(fr)
	axis := color.RGBA{R: 60, G: 64, B: 72, A: 255}
	c.Line(x, y, x, y+h, axis)
	c.Line(x, y+h, x+w, y+h, axis)

	sx := float64(w) / float64(fr.W)
	sy := float64(h) / float64(fr.H)
	wavy := color.RGBA{R: 200, G: 60, B: 60, A: 255}
	for row := 1; row < fr.H; row++ {
		if left[row] >= 0 && left[row-1] >= 0 {
			c.Line(x+int(float64(left[row-1])*sx), y+int(float64(row-1)*sy),
				x+int(float64(left[row])*sx), y+int(float64(row)*sy), wavy)
		}
		if right[row] >= 0 && right[row-1] >= 0 {
			c.Line(x+int(float64(right[row-1])*sx), y+int(float64(row-1)*sy),
				x+int(float64(right[row])*sx), y+int(float64(row)*sy), wavy)
		}
	}
	for col := 1; col < fr.W; col++ {
		if top[col] >= 0 && top[col-1] >= 0 {
			c.Line(x+int(float64(col-1)*sx), y+int(float64(top[col-1])*sy),
				x+int(float64(col)*sx), y+int(float64(top[col])*sy), wavy)
		}
		if bottom[col] >= 0 && bottom[col-1] >= 0 {
			c.Line(x+int(float64(col-1)*sx), y+int(float64(bottom[col-1])*sy),
				x+int(float64(col)*sx), y+int(float64(bottom[col])*sy), wavy)
		}
	}
	green := color.RGBA{R: 40, G: 180, B: 90, A: 255}
	b := fr.Box
	c.RectStroke(x+int(float64(b.X0)*sx), y+int(float64(b.Y0)*sy),
		x+int(float64(b.X1)*sx), y+int(float64(b.Y1)*sy), green)
}

func ftoa(v float64, prec int) string {
	// Small dependency-free formatter for non-negative chart labels.
	scaled := v
	mul := 1
	for i := 0; i < prec; i++ {
		scaled *= 10
		mul *= 10
	}
	whole := int(scaled + 0.5)
	if prec == 0 {
		return itoa(whole)
	}
	return itoa(whole/mul) + "." + pad(whole%mul, prec)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

func pad(v, width int) string {
	s := itoa(v)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
