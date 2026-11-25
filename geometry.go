package main

import (
	"math"
)

const (
	SurfaceNone       uint64 = 0
	SurfaceGold       uint64 = 1001
	BoundsSlack              = 1e-7
	IntersectionSlack        = 1e-9
)

type RGB = Vec3

type Ray struct {
	Origin Vec3
	Dir    Vec3
	TMin   float64
	TMax   float64
}

func NewRay(origin, dir Vec3) Ray {
	return Ray{Origin: origin, Dir: dir.Normalized(), TMin: 1e-7, TMax: math.Inf(1)}
}

func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Dir.Mul(t)) }

type AABB struct {
	Min Vec3 `json:"min"`
	Max Vec3 `json:"max"`
}

func EmptyAABB() AABB {
	inf := math.Inf(1)
	ninf := math.Inf(-1)
	return AABB{Min: V(inf, inf, inf), Max: V(ninf, ninf, ninf)}
}

func (b AABB) Valid() bool {
	return b.Min.X <= b.Max.X && b.Min.Y <= b.Max.Y && b.Min.Z <= b.Max.Z
}

func (b AABB) AddPoint(p Vec3) AABB {
	return AABB{Min: b.Min.Min(p), Max: b.Max.Max(p)}
}

func (b AABB) AddBox(other AABB) AABB {
	if !other.Valid() {
		return b
	}
	if !b.Valid() {
		return other
	}
	return AABB{Min: b.Min.Min(other.Min), Max: b.Max.Max(other.Max)}
}

func (b AABB) Expanded(d float64) AABB {
	return AABB{
		Min: b.Min.Sub(V(d, d, d)),
		Max: b.Max.Add(V(d, d, d)),
	}
}

func (b AABB) Center() Vec3 { return b.Min.Add(b.Max).Mul(0.5) }

func (b AABB) Intersect(r Ray) (tNear, tFar float64, ok bool) {
	b = b.Expanded(BoundsSlack)
	tNear = r.TMin
	tFar = r.TMax

	for axis := 0; axis < 3; axis++ {
		o := rayAxis(r.Origin, axis)
		d := rayAxis(r.Dir, axis)
		lo := boxAxis(b.Min, axis)
		hi := boxAxis(b.Max, axis)

		if math.Abs(d) < 1e-15 {
			if o < lo || o > hi {
				return 0, 0, false
			}
			continue
		}

		inv := 1 / d
		t1 := (lo - o) * inv
		t2 := (hi - o) * inv
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tNear {
			tNear = t1
		}
		if t2 < tFar {
			tFar = t2
		}
		if tNear > tFar+1e-10 {
			return 0, 0, false
		}
	}
	return tNear, tFar, true
}

func rayAxis(v Vec3, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func boxAxis(v Vec3, axis int) float64 { return rayAxis(v, axis) }

type Triangle struct {
	V             [3]Vec3 `json:"vertices"`
	Normal        Vec3    `json:"normal"`
	Color         RGB     `json:"color"`
	SurfaceID     uint64  `json:"surface_id"`
	OriginalIndex int     `json:"original_index"`
	Degenerate    bool    `json:"degenerate"`
}

func NewTriangle(a, b, c, color RGB, surfaceID uint64, originalIndex int) Triangle {
	e1 := b.Sub(a)
	e2 := c.Sub(a)
	n := e1.Cross(e2)
	degenerate := n.LenSqr() <= 1e-22
	if !degenerate {
		n = n.Normalized()
	}
	return Triangle{
		V:             [3]Vec3{a, b, c},
		Normal:        n,
		Color:         color,
		SurfaceID:     surfaceID,
		OriginalIndex: originalIndex,
		Degenerate:    degenerate,
	}
}

func (t Triangle) Bounds() AABB {
	b := EmptyAABB().AddPoint(t.V[0]).AddPoint(t.V[1]).AddPoint(t.V[2])
	return b.Expanded(BoundsSlack)
}

type TriHit struct {
	T        float64
	Point    Vec3
	Normal   Vec3
	BaryU    float64
	BaryV    float64
	Triangle int
}

// IntersectTriangle uses Möller–Trumbore with a small barycentric tolerance.
// Exact edges and vertices are accepted; zero-area triangles are rejected
// before division can amplify floating-point noise.
func IntersectTriangle(r Ray, tri Triangle, index int) (TriHit, bool) {
	e1 := tri.V[1].Sub(tri.V[0])
	e2 := tri.V[2].Sub(tri.V[0])
	geometricNormal := e1.Cross(e2)
	if geometricNormal.LenSqr() <= 1e-22 {
		return TriHit{}, false
	}

	p := r.Dir.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-14 {
		return TriHit{}, false
	}
	invDet := 1 / det

	tv := r.Origin.Sub(tri.V[0])
	u := tv.Dot(p) * invDet
	baryEps := 2e-9
	if u < -baryEps || u > 1+baryEps {
		return TriHit{}, false
	}

	q := tv.Cross(e1)
	v := r.Dir.Dot(q) * invDet
	if v < -baryEps || u+v > 1+baryEps {
		return TriHit{}, false
	}

	t := e2.Dot(q) * invDet
	if t < r.TMin || t > r.TMax {
		return TriHit{}, false
	}

	u = clampFloat(u, 0, 1)
	v = clampFloat(v, 0, 1-u)
	point := r.At(t)
	n := tri.Normal
	if n.Dot(r.Dir) > 0 {
		n = n.Mul(-1)
	}
	return TriHit{
		T:        t,
		Point:    point,
		Normal:   n,
		BaryU:    u,
		BaryV:    v,
		Triangle: index,
	}, true
}

type Hit struct {
	OK               bool
	T                float64
	Point            Vec3
	Normal           Vec3
	BaryU            float64
	BaryV            float64
	SurfaceID        uint64
	OriginalTriangle int
	StorageTriangle  int
	Color            RGB
	LocalPoint       Vec3
	InstancePath     []uint32
	Depth            int
}

func floatClose(a, b, eps float64) bool {
	d := math.Abs(a - b)
	scale := math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
	return d <= eps*scale
}

func vecClose(a, b Vec3, eps float64) bool {
	return floatClose(a.X, b.X, eps) && floatClose(a.Y, b.Y, eps) && floatClose(a.Z, b.Z, eps)
}
