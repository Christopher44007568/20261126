package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec3) MulVec(b Vec3) Vec3 { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }

func (a Vec3) Div(s float64) Vec3 { return a.Mul(1 / s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) LenSqr() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.LenSqr()) }

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Div(l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
