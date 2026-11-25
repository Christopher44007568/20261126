package scene

import "math"

type Painter struct {
	H *HDR
}

func (p *Painter) Rect(x0, y0, x1, y1 int, c RGB) {
	for y := maxInt(y0, 0); y < minInt(y1, p.H.Height); y++ {
		for x := maxInt(x0, 0); x < minInt(x1, p.H.Width); x++ {
			p.H.Set(x, y, c)
		}
	}
}

func (p *Painter) GradientV(x0, y0, x1, y1 int, top, bottom RGB) {
	for y := maxInt(y0, 0); y < minInt(y1, p.H.Height); y++ {
		t := 0.0
		if y1 > y0 {
			t = float64(y-y0) / float64(y1-y0)
		}
		c := LerpRGB(top, bottom, t)
		for x := maxInt(x0, 0); x < minInt(x1, p.H.Width); x++ {
			p.H.Set(x, y, c)
		}
	}
}

func (p *Painter) GradientH(x0, y0, x1, y1 int, left, right RGB) {
	for x := maxInt(x0, 0); x < minInt(x1, p.H.Width); x++ {
		t := 0.0
		if x1 > x0 {
			t = float64(x-x0) / float64(x1-x0)
		}
		c := LerpRGB(left, right, t)
		for y := maxInt(y0, 0); y < minInt(y1, p.H.Height); y++ {
			p.H.Set(x, y, c)
		}
	}
}

func (p *Painter) AlphaRect(x0, y0, x1, y1 int, c RGB, alpha float32) {
	for y := maxInt(y0, 0); y < minInt(y1, p.H.Height); y++ {
		for x := maxInt(x0, 0); x < minInt(x1, p.H.Width); x++ {
			old := p.H.At(x, y)
			p.H.Set(x, y, RGB{
				R: old.R*(1-alpha) + c.R*alpha,
				G: old.G*(1-alpha) + c.G*alpha,
				B: old.B*(1-alpha) + c.B*alpha,
			})
		}
	}
}

func (p *Painter) Circle(cx, cy, radius int, c RGB) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				p.H.Set(x, y, c)
			}
		}
	}
}

func (p *Painter) RadialGlow(cx, cy, radius int, c RGB, strength float32) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			d := math.Hypot(float64(dx), float64(dy)) / float64(radius)
			if d <= 1 {
				a := strength * float32((1-d)*(1-d))
				p.H.Add(x, y, RGB{c.R * a, c.G * a, c.B * a})
			}
		}
	}
}

func (p *Painter) Line(x0, y0, x1, y1, thickness int, c RGB) {
	steps := int(math.Max(math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0)))) + 1
	r := thickness / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(lerp(float64(x0), float64(x1), t)))
		y := int(math.Round(lerp(float64(y0), float64(y1), t)))
		p.Rect(x-r, y-r, x+r+1, y+r+1, c)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
