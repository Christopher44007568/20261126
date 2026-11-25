package main

import "math"

// Vec3 is a three-dimensional vector used for both points and directions.
type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (v Vec3) Add(o Vec3) Vec3        { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3        { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Mul(s float64) Vec3     { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) MulVec(o Vec3) Vec3     { return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z} }
func (v Vec3) Dot(o Vec3) float64     { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }
func (v Vec3) LengthSquared() float64 { return v.Dot(v) }
func (v Vec3) Length() float64        { return math.Sqrt(v.LengthSquared()) }

func (v Vec3) Normalize() Vec3 {
	n := v.Length()
	if n < 1e-12 {
		return Vec3{}
	}
	return v.Mul(1 / n)
}

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) Clamp(min, max float64) Vec3 {
	return Vec3{
		math.Max(min, math.Min(max, v.X)),
		math.Max(min, math.Min(max, v.Y)),
		math.Max(min, math.Min(max, v.Z)),
	}
}

func reflect(d, n Vec3) Vec3 {
	return d.Sub(n.Mul(2 * d.Dot(n)))
}

func mix(a, b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

func schlick(cosine, ior float64) float64 {
	r0 := (1 - ior) / (1 + ior)
	r0 *= r0
	return r0 + (1-r0)*math.Pow(1-cosine, 5)
}
