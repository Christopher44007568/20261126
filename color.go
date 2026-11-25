package main

import "math"

// Vec3 is a linear-light triple (RGB or CIE XYZ depending on context).
type Vec3 struct{ R, G, B float64 }

type Mat3 [3][3]float64

func (a Mat3) mul(b Mat3) Mat3 {
	var c Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}

func (m Mat3) apply(v Vec3) Vec3 {
	return Vec3{
		R: m[0][0]*v.R + m[0][1]*v.G + m[0][2]*v.B,
		G: m[1][0]*v.R + m[1][1]*v.G + m[1][2]*v.B,
		B: m[2][0]*v.R + m[2][1]*v.G + m[2][2]*v.B,
	}
}

func (m Mat3) inverse() Mat3 {
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	g, h, i := m[2][0], m[2][1], m[2][2]
	det := a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
	inv := 1 / det
	return Mat3{
		{(e*i - f*h) * inv, (c*h - b*i) * inv, (b*f - c*e) * inv},
		{(f*g - d*i) * inv, (a*i - c*g) * inv, (c*d - a*f) * inv},
		{(d*h - e*g) * inv, (b*g - a*h) * inv, (a*e - b*d) * inv},
	}
}

// Chromaticity holds CIE 1931 x,y coordinates.
type Chromaticity struct{ X, Y float64 }

func (p Chromaticity) xyz() Vec3 {
	if p.Y == 0 {
		return Vec3{0, 0, 0}
	}
	return Vec3{p.X / p.Y, 1, (1 - p.X - p.Y) / p.Y}
}

// Primaries describe a display's RGB color volume in CIE 1931 xy.
type Primaries struct{ Red, Green, Blue Chromaticity }

var (
	// CIE standard illuminant / display white points (x, y).
	WP_D65 = Chromaticity{0.3127, 0.3290}
	WP_D93 = Chromaticity{0.2848, 0.2932} // cool blue-white
	WP_5K  = Chromaticity{0.3457, 0.3585} // warm 5000 K
)

var (
	P3Primaries   = Primaries{Chromaticity{0.680, 0.320}, Chromaticity{0.265, 0.690}, Chromaticity{0.150, 0.060}}
	Rec2020Prim   = Primaries{Chromaticity{0.708, 0.292}, Chromaticity{0.170, 0.797}, Chromaticity{0.131, 0.046}}
	sRGBPrimaries = Primaries{Chromaticity{0.640, 0.330}, Chromaticity{0.300, 0.600}, Chromaticity{0.150, 0.060}}
)

// rgbToXYZ builds the linear-RGB -> XYZ matrix for given primaries/white.
func rgbToXYZ(p Primaries, white Chromaticity) Mat3 {
	pr, pg, pb := p.Red.xyz(), p.Green.xyz(), p.Blue.xyz()
	m := Mat3{
		{pr.R, pg.R, pb.R},
		{pr.G, pg.G, pb.G},
		{pr.B, pg.B, pb.B},
	}
	ws := white.xyz()
	s := m.inverse().apply(ws)
	return Mat3{
		{pr.R * s.R, pg.R * s.G, pb.R * s.B},
		{pr.G * s.R, pg.G * s.G, pb.G * s.B},
		{pr.B * s.R, pg.B * s.G, pb.B * s.B},
	}
}

// Bradford cone response matrices for chromatic adaptation (von Kries).
var bradford = Mat3{
	{0.8951, 0.2664, -0.1614},
	{-0.7502, 1.7135, 0.0367},
	{0.0389, -0.0685, 1.0296},
}

// adaptation returns an XYZ->XYZ matrix that maps src white to dst white.
func adaptation(src, dst Chromaticity) Mat3 {
	cs := bradford.apply(src.xyz())
	cd := bradford.apply(dst.xyz())
	scale := Mat3{
		{cd.R / cs.R, 0, 0},
		{0, cd.G / cs.G, 0},
		{0, 0, cd.B / cs.B},
	}
	return bradford.inverse().mul(scale).mul(bradford)
}

// sRGB OETF (linear -> gamma-encoded), standard library transfer curve.
func sRGBEncode(linear float64) float64 {
	if linear <= 0.0031308 {
		return 12.92 * linear
	}
	return 1.055*math.Pow(linear, 1/2.4) - 0.055
}

// PQ OETF (SMPTE ST 2084). Linear is absolute scene linear in [0,1]
// where 1.0 corresponds to 10000 cd/m^2. Returns perceptual signal [0,1].
func pqEncode(linear10k float64) float64 {
	const m1, m2 = 0.1593017578125, 78.84375
	const c1, c2, c3 = 0.8359375, 18.8515625, 18.6875
	lp := math.Pow(math.Max(linear10k, 0), m1)
	num := c1 + c2*lp
	den := 1 + c3*lp
	return math.Pow(num/den, m2)
}

// pqDecode is the inverse PQ OETF: perceptual signal -> linear [0,1] of
// 10000 cd/m^2. Used when soft-proofing the HDR panel.
func pqDecode(signal float64) float64 {
	const m1, m2 = 0.1593017578125, 78.84375
	const c1, c2, c3 = 0.8359375, 18.8515625, 18.6875
	ep := math.Pow(math.Max(signal, 0), 1/m2)
	num := math.Max(ep-c1, 0)
	den := c2 - c3*ep
	return math.Pow(num/den, 1/m1)
}

func sRGBDecode(encoded float64) float64 {
	if encoded <= 0.04045 {
		return encoded / 12.92
	}
	return math.Pow((encoded+0.055)/1.055, 2.4)
}

// sRGB XYZ (D65) -> linear sRGB RGB, used for the shared preview canvas.
var sRGBXYZToRGB = rgbToXYZ(sRGBPrimaries, WP_D65).inverse()

// previewNormalize maps absolute preview luminance (nits) onto a common
// 200-nit sRGB reference. Below the reference it is 1:1 (so the dim display
// stays visibly dimmer); above it a soft shoulder preserves HDR highlights
// instead of hard-clipping the comparison image.
func previewNormalize(xNits float64) float64 {
	const ref = 200.0
	v := xNits / ref
	if v <= 1 {
		return v
	}
	return 1 + (v-1)/(1+(v-1)*0.5)
}

func clamp01(v float64) float64 { return math.Min(1, math.Max(0, v)) }
