package main

import "math"

// Vec3 is a minimal 3D vector used for positions, directions and colors.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64         { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

// Comp multiplies components pairwise (used for spectral attenuation).
func (a Vec3) Comp(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}

func Lerp(a, b Vec3, t float64) Vec3 {
	return Vec3{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
		a.Z + (b.Z-a.Z)*t,
	}
}

func Clamp01(v Vec3) Vec3 {
	return Vec3{clamp(v.X), clamp(v.Y), clamp(v.Z)}
}

func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Reflect returns v reflected around unit normal n.
func Reflect(v, n Vec3) Vec3 {
	return v.Sub(n.Scale(2 * v.Dot(n)))
}

// Refract returns the transmitted direction. ok is false on total internal
// reflection. eta is n1/n2, the ratio of the incident to transmitted IOR.
func Refract(v, n Vec3, eta float64) (dir Vec3, ok bool) {
	cosi := -v.Dot(n)
	k := 1 - eta*eta*(1-cosi*cosi)
	if k < 0 {
		return Vec3{}, false
	}
	dir = v.Scale(eta).Add(n.Scale(eta*cosi - math.Sqrt(k)))
	return dir, true
}

// Schlick approximates the Fresnel reflectance at a dielectric surface.
func Schlick(cosine, ior float64) float64 {
	r0 := (1 - ior) / (1 + ior)
	r0 *= r0
	return r0 + (1-r0)*math.Pow(1-math.Max(cosine, 0), 5)
}

// BeerLambert returns spectral transmittance after travelling d units through
// a medium with the given Napierian absorption coefficients.
func BeerLambert(absorption Vec3, d float64) Vec3 {
	return Vec3{
		math.Exp(-absorption.X * d),
		math.Exp(-absorption.Y * d),
		math.Exp(-absorption.Z * d),
	}
}

func Luminance(c Vec3) float64 {
	return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
}
