package audit

import "math"

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (vector Vec3) Add(other Vec3) Vec3 {
	return Vec3{vector.X + other.X, vector.Y + other.Y, vector.Z + other.Z}
}

func (vector Vec3) Sub(other Vec3) Vec3 {
	return Vec3{vector.X - other.X, vector.Y - other.Y, vector.Z - other.Z}
}

func (vector Vec3) Scale(amount float64) Vec3 {
	return Vec3{vector.X * amount, vector.Y * amount, vector.Z * amount}
}

func (vector Vec3) Mul(other Vec3) Vec3 {
	return Vec3{vector.X * other.X, vector.Y * other.Y, vector.Z * other.Z}
}

func (vector Vec3) Dot(other Vec3) float64 {
	return vector.X*other.X + vector.Y*other.Y + vector.Z*other.Z
}

func (vector Vec3) Cross(other Vec3) Vec3 {
	return Vec3{
		vector.Y*other.Z - vector.Z*other.Y,
		vector.Z*other.X - vector.X*other.Z,
		vector.X*other.Y - vector.Y*other.X,
	}
}

func (vector Vec3) Length() float64 {
	return math.Sqrt(vector.Dot(vector))
}

func (vector Vec3) Normalize() Vec3 {
	length := vector.Length()
	if length <= 1e-12 {
		return Vec3{}
	}
	return vector.Scale(1 / length)
}

func (vector Vec3) Clamp01() Vec3 {
	return Vec3{clamp01(vector.X), clamp01(vector.Y), clamp01(vector.Z)}
}
