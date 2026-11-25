package rayao

import "math"

// Vec3 is a float64 Cartesian vector or point.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Div(s float64) Vec3 { return Vec3{a.X / s, a.Y / s, a.Z / s} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}
func (a Vec3) LenSqr() float64 { return a.Dot(a) }
func (a Vec3) Len() float64    { return math.Sqrt(a.LenSqr()) }

func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l == 0 || math.IsNaN(l) || math.IsInf(l, 0) {
		return Vec3{}
	}
	return a.Div(l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return Vec3{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
		a.Z + (b.Z-a.Z)*t,
	}
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
