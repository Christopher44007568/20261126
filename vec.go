package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64       { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Norm() Vec3         { return a.Scale(1 / a.Len()) }

// Reflect mirrors v about unit normal n.
func Reflect(v, n Vec3) Vec3 { return v.Sub(n.Scale(2 * v.Dot(n))) }

type Ray struct {
	Orig, Dir Vec3
}

func (r Ray) At(t float64) Vec3 { return r.Orig.Add(r.Dir.Scale(t)) }
