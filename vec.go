package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func vAdd(a, b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func vSub(a, b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func vScale(a Vec3, s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func vDot(a, b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func vCross(a, b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func vLen(a Vec3) float64 { return math.Sqrt(vDot(a, a)) }

func vNorm(a Vec3) Vec3 {
	l := vLen(a)
	if l < 1e-12 {
		return Vec3{0, 0, 1}
	}
	return vScale(a, 1/l)
}

func vMin(a, b Vec3) Vec3 {
	return Vec3{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)}
}

func vMax(a, b Vec3) Vec3 {
	return Vec3{math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)}
}

type AABB struct {
	Min Vec3
	Max Vec3
}

func boundsUnion(a, b AABB) AABB {
	return AABB{Min: vMin(a.Min, b.Min), Max: vMax(a.Max, b.Max)}
}

func (b AABB) Hit(origin, dir Vec3) bool {
	const eps = 1e-9
	tMin := -math.MaxFloat64
	tMax := math.MaxFloat64

	axes := [3]struct{ o, d, lo, hi float64 }{
		{origin.X, dir.X, b.Min.X, b.Max.X},
		{origin.Y, dir.Y, b.Min.Y, b.Max.Y},
		{origin.Z, dir.Z, b.Min.Z, b.Max.Z},
	}
	for _, ax := range axes {
		if math.Abs(ax.d) < eps {
			if ax.o < ax.lo || ax.o > ax.hi {
				return false
			}
			continue
		}
		t1 := (ax.lo - ax.o) / ax.d
		t2 := (ax.hi - ax.o) / ax.d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tMin {
			tMin = t1
		}
		if t2 < tMax {
			tMax = t2
		}
		if tMin > tMax+eps || tMax < 0 {
			return false
		}
	}
	return true
}

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

type Hit struct {
	T        float64
	U, V     float64
	TriIndex int
}

// Intersect implements Möller–Trumbore without back-face culling.
func (tri Triangle) Intersect(ray Ray) (Hit, bool) {
	const eps = 1e-10
	e1 := vSub(tri.B, tri.A)
	e2 := vSub(tri.C, tri.A)
	p := vCross(ray.Dir, e2)
	det := vDot(e1, p)
	if math.Abs(det) < eps {
		return Hit{}, false
	}
	invDet := 1 / det
	tv := vSub(ray.Origin, tri.A)
	u := vDot(tv, p) * invDet
	if u < 0 || u > 1 {
		return Hit{}, false
	}
	q := vCross(tv, e1)
	v := vDot(ray.Dir, q) * invDet
	if v < 0 || u+v > 1 {
		return Hit{}, false
	}
	t := vDot(e2, q) * invDet
	if t < eps {
		return Hit{}, false
	}
	return Hit{T: t, U: u, V: v, TriIndex: tri.ID}, true
}
