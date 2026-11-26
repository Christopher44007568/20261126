package shadow

import "math"

// Vec3 is a minimal 3D vector used for positions, directions and normals.
type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Dot(o Vec3) float64   { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) LengthSquared() float64 { return v.Dot(v) }

func (v Vec3) Length() float64 { return math.Sqrt(v.LengthSquared()) }

func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length == 0 {
		return v
	}
	return v.Scale(1 / length)
}

// Ray is a parametric ray Origin + t*Direction, t >= 0.
type Ray struct {
	Origin    Vec3
	Direction Vec3
}

func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Direction.Scale(t)) }
