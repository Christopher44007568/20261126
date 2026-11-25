package main

import "math"

// Deterministic hash noise in [0,1].
func hash2(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0xffffff)
}

func valueNoise(x, y float64) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	xf, yf := x-float64(xi), y-float64(yi)
	u := xf * xf * (3 - 2*xf)
	v := yf * yf * (3 - 2*yf)
	a := hash2(xi, yi)
	b := hash2(xi+1, yi)
	c := hash2(xi, yi+1)
	d := hash2(xi+1, yi+1)
	return a + (b-a)*u + (c-a)*v + (a-b-c+d)*u*v
}

func fbm(x, y float64, oct int) float64 {
	sum, amp, freq := 0.0, 0.5, 1.0
	for i := 0; i < oct; i++ {
		sum += amp * valueNoise(x*freq, y*freq)
		amp *= 0.5
		freq *= 2
	}
	return sum
}

func background(im *Image) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			t := float64(y) / float64(im.H)
			im.Set(x, y, 0.10+0.10*t, 0.11+0.10*t, 0.14+0.12*t)
		}
	}
}

// RenderSphere: glossy plastic sphere, Blinn-Phong highlight, subtle mottling.
func RenderSphere(size int) *Image {
	im := NewImage(size, size)
	background(im)
	cx, cy, r := float64(size)/2, float64(size)/2, float64(size)*0.40
	lx, ly, lz := -0.45, -0.55, 0.70
	ll := math.Sqrt(lx*lx + ly*ly + lz*lz)
	lx, ly, lz = lx/ll, ly/ll, lz/ll
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			nx := (float64(x) - cx) / r
			ny := (float64(y) - cy) / r
			d2 := nx*nx + ny*ny
			if d2 > 1 {
				continue
			}
			nz := math.Sqrt(1 - d2)
			mot := 0.85 + 0.3*fbm(float64(x)*0.03, float64(y)*0.03, 3)
			diff := math.Max(0, nx*lx+ny*ly+nz*lz)
			hx, hy, hz := lx, ly, lz+1
			hl := math.Sqrt(hx*hx + hy*hy + hz*hz)
			ndh := math.Max(0, nx*hx/hl+ny*hy/hl+nz*hz/hl)
			spec := math.Pow(ndh, 220) * 1.4
			ar, ag, ab := 0.72*mot, 0.30*mot, 0.22*mot
			amb := 0.12
			im.Set(x, y,
				ar*(amb+diff)+spec,
				ag*(amb+diff)+spec,
				ab*(amb+diff)+spec)
		}
	}
	return im
}

// RenderCloth: woven fabric with soft folds and a gentle sheen.
func RenderCloth(size int) *Image {
	im := NewImage(size, size)
	background(im)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			u := float64(x) / float64(size)
			v := float64(y) / float64(size)
			fold := 0.65 + 0.35*math.Sin(u*math.Pi*2.2+0.8*math.Sin(v*3.0))
			wx := 0.5 + 0.5*math.Sin(float64(x)*0.9)
			wy := 0.5 + 0.5*math.Sin(float64(y)*0.9)
			weave := 0.55 + 0.45*(wx*wy)
			n := 0.9 + 0.2*fbm(u*24, v*24, 3)
			sheen := math.Pow(math.Max(0, math.Sin(u*math.Pi*2.2+0.8*math.Sin(v*3.0)+0.6)), 24) * 0.5
			lum := fold * weave * n
			im.Set(x, y,
				0.62*lum+sheen,
				0.55*lum+sheen*0.95,
				0.66*lum+sheen*0.85)
		}
	}
	return im
}

// RenderMetal: brushed metal plate with anisotropic streak highlight.
func RenderMetal(size int) *Image {
	im := NewImage(size, size)
	background(im)
	x0, y0 := size/8, size/8
	x1, y1 := size-size/8, size-size/8
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x < x0 || x >= x1 || y < y0 || y >= y1 {
				continue
			}
			streak := fbm(float64(x)*0.35, float64(y)*0.015, 3)
			base := 0.38 + 0.35*streak
			u := float64(x-x0) / float64(x1-x0)
			v := float64(y-y0) / float64(y1-y0)
			d := (v - 0.32) * 6.0
			band := math.Exp(-d*d) * (0.75 + 0.25*math.Sin(u*math.Pi))
			val := base*(0.55+0.45*math.Sin(u*math.Pi)) + band*0.85
			im.Set(x, y, val*0.96, val*0.98, val*1.04)
		}
	}
	return im
}
