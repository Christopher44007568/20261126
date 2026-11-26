package main

import "math"

// hash01 is a deterministic integer hash returning [0,1). No external
// randomness is used, so every generated map is reproducible.
func hash01(ix, iy int) float64 {
	n := uint32(ix)*374761393 + uint32(iy)*668265263
	n = (n ^ (n >> 13)) * 1274126177
	n ^= n >> 16
	return float64(n) / 4294967296.0
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// valueNoise returns smooth value noise in [0,1] at continuous coordinates.
func valueNoise(x, y float64) float64 {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := smooth(x - float64(x0))
	fy := smooth(y - float64(y0))
	v00 := hash01(x0, y0)
	v10 := hash01(x0+1, y0)
	v01 := hash01(x0, y0+1)
	v11 := hash01(x0+1, y0+1)
	return lerp(lerp(v00, v10, fx), lerp(v01, v11, fx), fy)
}

func fbm(x, y float64) float64 {
	sum, amp, norm, freq := 0.0, 1.0, 0.0, 1.0
	for oct := 0; oct < 4; oct++ {
		sum += amp * valueNoise(x*freq, y*freq)
		norm += amp
		amp *= 0.5
		freq *= 2.03
	}
	return sum / norm
}

// buildSource produces albedo RGB, a tangent-space normal map derived from
// an analytic height field, plus scalar roughness and occlusion.
func buildSource(w, h int) *Maps {
	m := &Maps{
		W:         w,
		H:         h,
		Albedo:    make([]byte, w*h*3),
		Normal:    make([]byte, w*h*3),
		Roughness: make([]byte, w*h),
		Occlusion: make([]byte, w*h),
	}
	height := make([]float64, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u := float64(x) / float64(w-1)
			v := float64(y) / float64(h-1)
			i := y*w + x

			// Large smooth bands plus low-frequency noise: gradients make any
			// channel swizzle immediately visible on the lit sphere.
			n1 := fbm(u*3.1+4.0, v*3.1+9.0)
			n2 := fbm(u*5.2+1.7, v*5.2+2.3)
			bandR := 0.5 + 0.5*math.Sin(6.0*u+3.5*v+2.2*n1)
			bandG := 0.5 + 0.5*math.Sin(4.5*v-2.8*u+1.7*n2)
			bandB := 0.5 + 0.5*math.Sin(5.0*(u+v)+3.0*n1)
			r := 95 + 120*bandR + 28*n1
			g := 90 + 110*bandG + 30*n2
			b := 95 + 105*bandB + 24*fbm(u*2.2, v*2.2)
			c := m.idx(x, y)
			m.Albedo[c] = clampByte(r)
			m.Albedo[c+1] = clampByte(g)
			m.Albedo[c+2] = clampByte(b)

			height[i] = 0.55*n1 + 0.30*n2 +
				0.15*math.Sin(8.0*u+4.0*v)

			rough := 0.42 + 0.34*fbm(u*4.0+11.0, v*4.0+5.0) +
				0.12*math.Sin(7.0*v+u)
			m.Roughness[i] = clampByte(255 * clamp01(rough))

			ao := 0.55 + 0.35*fbm(u*3.6+2.0, v*3.6+8.0) +
				0.10*math.Sin(9.0*u-2.0*v)
			m.Occlusion[i] = clampByte(255 * clamp01(ao))
		}
	}

	deriveNormals(m, height, 2.2)
	return m
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// deriveNormals builds an OpenGL-style tangent-space normal map (X right,
// Y up, Z toward viewer) using central differences on the height field.
func deriveNormals(m *Maps, height []float64, strength float64) {
	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if x >= m.W {
			x = m.W - 1
		}
		if y < 0 {
			y = 0
		}
		if y >= m.H {
			y = m.H - 1
		}
		return height[y*m.W+x]
	}
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			dx := (at(x+1, y) - at(x-1, y)) * strength
			// Flip dy because image rows grow downward but normal Y points up.
			dy := -(at(x, y+1) - at(x, y-1)) * strength
			nx, ny, nz := normalize3(-dx, -dy, 1.0)
			c := m.idx(x, y)
			m.Normal[c] = clampByte((nx*0.5 + 0.5) * 255)
			m.Normal[c+1] = clampByte((ny*0.5 + 0.5) * 255)
			m.Normal[c+2] = clampByte((nz*0.5 + 0.5) * 255)
		}
	}
}

func normalize3(x, y, z float64) (float64, float64, float64) {
	l := math.Sqrt(x*x + y*y + z*z)
	return x / l, y / l, z / l
}
