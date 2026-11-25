package internal

import "math"

// Vec3 is a 3D vector in double precision. Ray differentials accumulate over
// several bounces, so float64 is used to keep the footprint numerically stable.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Length2() float64 { return a.Dot(a) }
func (a Vec3) Length() float64  { return math.Sqrt(a.Length2()) }

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n == 0 {
		return a
	}
	return a.Scale(1 / n)
}

// Neg flips the vector direction.
func (a Vec3) Neg() Vec3 { return Vec3{-a.X, -a.Y, -a.Z} }

// Reflect mirrors direction d around unit normal n.
func Reflect(d, n Vec3) Vec3 {
	return d.Sub(n.Scale(2 * d.Dot(n)))
}

// Clamp compresses v into [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Lerp linearly blends a and b.
func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

// TangentSpace returns two unit tangents perpendicular to unit normal n.
// Deterministic branching keeps the basis stable frame to frame, which matters
// for continuity of the roughness footprints.
func TangentSpace(n Vec3) (t, b Vec3) {
	axis := V(0, 1, 0)
	if math.Abs(n.Y) > 0.95 {
		axis = V(1, 0, 0)
	}
	t = n.Cross(axis).Normalize()
	b = n.Cross(t).Normalize()
	return t, b
}
