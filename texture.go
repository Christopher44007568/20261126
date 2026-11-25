package main

import "math"

type Texture struct {
	W, H int
	Pix  []Vec3
}

func TextureFromCanvas(c *Canvas) *Texture {
	t := &Texture{W: c.W, H: c.H, Pix: make([]Vec3, c.W*c.H)}
	for i := 0; i < c.W*c.H; i++ {
		r := float64(c.Pix[i*3+0]) / 255
		g := float64(c.Pix[i*3+1]) / 255
		b := float64(c.Pix[i*3+2]) / 255
		t.Pix[i] = V(srgbToLinear(r), srgbToLinear(g), srgbToLinear(b))
	}
	return t
}

func srgbToLinear(x float64) float64 {
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

// Sample returns bilinearly filtered texel at UV in [0,1].
func (t *Texture) Sample(u, v float64) Vec3 {
	if t == nil || t.W == 0 || t.H == 0 {
		return V(1, 0, 1)
	}
	x := clamp(u, 0, 1) * float64(t.W-1)
	imageV := 1 - clamp(v, 0, 1)
	y := imageV * float64(t.H-1)
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	x1, y1 := x0+1, y0+1
	tx, ty := x-float64(x0), y-float64(y0)
	c00 := t.texel(x0, y0)
	c10 := t.texel(x1, y0)
	c01 := t.texel(x0, y1)
	c11 := t.texel(x1, y1)
	a := c00.Lerp(c10, tx)
	b := c01.Lerp(c11, tx)
	return a.Lerp(b, ty)
}

func (t *Texture) texel(x, y int) Vec3 {
	if x < 0 {
		x = 0
	}
	if x >= t.W {
		x = t.W - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= t.H {
		y = t.H - 1
	}
	return t.Pix[y*t.W+x]
}
