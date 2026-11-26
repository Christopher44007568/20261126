package main

import (
	"math"
	"sort"
)

type Ray struct {
	Origin Vec3
	Dir    Vec3
	MinT   float64
	MaxT   float64
}

func MakeRay(origin, direction Vec3) Ray {
	return Ray{Origin: origin, Dir: direction.Normalize(), MinT: 0, MaxT: math.Inf(1)}
}

type AABB struct {
	Min Vec3
	Max Vec3
}

func EmptyAABB() AABB {
	inf := math.Inf(1)
	return AABB{Min: V(inf, inf, inf), Max: V(-inf, -inf, -inf)}
}

func (b AABB) Valid() bool {
	return b.Min.X <= b.Max.X && b.Min.Y <= b.Max.Y && b.Min.Z <= b.Max.Z
}

func (b AABB) IncludePoint(p Vec3) AABB {
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Min.Z = math.Min(b.Min.Z, p.Z)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
	b.Max.Z = math.Max(b.Max.Z, p.Z)
	return b
}

func (b AABB) IncludeBox(c AABB) AABB {
	if !c.Valid() {
		return b
	}
	b = b.IncludePoint(c.Min)
	b = b.IncludePoint(c.Max)
	return b
}

func coordinateScale(points ...Vec3) float64 {
	scale := 1.0
	for _, p := range points {
		scale = math.Max(scale, math.Abs(p.X))
		scale = math.Max(scale, math.Abs(p.Y))
		scale = math.Max(scale, math.Abs(p.Z))
	}
	return scale
}

func conservativeEpsilon(points ...Vec3) float64 {
	return 1e-9 * coordinateScale(points...)
}

func (b AABB) Expanded(points ...Vec3) AABB {
	eps := conservativeEpsilon(points...)
	return AABB{
		Min: V(b.Min.X-eps, b.Min.Y-eps, b.Min.Z-eps),
		Max: V(b.Max.X+eps, b.Max.Y+eps, b.Max.Z+eps),
	}
}

func (b AABB) Centroid() Vec3 {
	return V(
		(b.Min.X+b.Max.X)*0.5,
		(b.Min.Y+b.Max.Y)*0.5,
		(b.Min.Z+b.Max.Z)*0.5,
	)
}

func (b AABB) LargestAxis() int {
	sx := b.Max.X - b.Min.X
	sy := b.Max.Y - b.Min.Y
	sz := b.Max.Z - b.Min.Z
	if sx >= sy && sx >= sz {
		return 0
	}
	if sy >= sz {
		return 1
	}
	return 2
}

func (b AABB) IntersectsRay(r Ray) bool {
	tMin := r.MinT
	tMax := r.MaxT
	origin := [3]float64{r.Origin.X, r.Origin.Y, r.Origin.Z}
	dir := [3]float64{r.Dir.X, r.Dir.Y, r.Dir.Z}
	boxMin := [3]float64{b.Min.X, b.Min.Y, b.Min.Z}
	boxMax := [3]float64{b.Max.X, b.Max.Y, b.Max.Z}

	for axis := 0; axis < 3; axis++ {
		o := origin[axis]
		d := dir[axis]
		lo := boxMin[axis]
		hi := boxMax[axis]
		if math.Abs(d) < 1e-15 {
			if o < lo || o > hi {
				return false
			}
			continue
		}

		t1 := (lo - o) / d
		t2 := (hi - o) / d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tMin = math.Max(tMin, t1)
		tMax = math.Min(tMax, t2)
		intervalEps := 1e-11 * math.Max(1, math.Max(math.Abs(tMin), math.Abs(tMax)))
		if tMin > tMax+intervalEps {
			return false
		}
	}
	return true
}

type Triangle struct {
	ID     int
	Name   string
	A      Vec3
	B      Vec3
	C      Vec3
	Albedo Vec3
}

func (t Triangle) Vertices() [3]Vec3 { return [3]Vec3{t.A, t.B, t.C} }

func (t Triangle) Centroid() Vec3 {
	return t.A.Add(t.B).Add(t.C).Scale(1.0 / 3.0)
}

func (t Triangle) Bounds() AABB {
	bounds := EmptyAABB().IncludePoint(t.A).IncludePoint(t.B).IncludePoint(t.C)
	return bounds.Expanded(t.A, t.B, t.C)
}

type Hit struct {
	ID          int
	T           float64
	U           float64
	V           float64
	Point       Vec3
	Normal      Vec3
	FrontFacing bool
	BaryEpsilon float64
	DetEpsilon  float64
	Determinant float64
}

func (t Triangle) Intersect(r Ray) (Hit, bool) {
	e1 := t.B.Sub(t.A)
	e2 := t.C.Sub(t.A)
	pVec := r.Dir.Cross(e2)
	det := e1.Dot(pVec)
	detScale := math.Max(1, e1.Len()*e2.Len())
	detEps := 1e-12 * detScale
	if math.Abs(det) <= detEps {
		return Hit{ID: t.ID, Determinant: det, DetEpsilon: detEps}, false
	}

	invDet := 1 / det
	sVec := r.Origin.Sub(t.A)
	u := invDet * sVec.Dot(pVec)
	baryEps := 1e-9
	if u < -baryEps || u > 1+baryEps {
		return Hit{ID: t.ID, Determinant: det, DetEpsilon: detEps}, false
	}

	qVec := sVec.Cross(e1)
	v := invDet * r.Dir.Dot(qVec)
	if v < -baryEps || u+v > 1+baryEps {
		return Hit{ID: t.ID, Determinant: det, DetEpsilon: detEps}, false
	}

	hitT := invDet * e2.Dot(qVec)
	if math.IsNaN(hitT) || hitT < r.MinT || hitT > r.MaxT {
		return Hit{ID: t.ID, T: hitT, Determinant: det, DetEpsilon: detEps}, false
	}

	normal := e1.Cross(e2).Normalize()
	frontFacing := r.Dir.Dot(normal) < 0
	if !frontFacing {
		normal = normal.Scale(-1)
	}
	point := r.Origin.Add(r.Dir.Scale(hitT))
	return Hit{
		ID:          t.ID,
		T:           hitT,
		U:           Clamp(u, 0, 1),
		V:           Clamp(v, 0, 1),
		Point:       point,
		Normal:      normal,
		FrontFacing: frontFacing,
		BaryEpsilon: baryEps,
		DetEpsilon:  detEps,
		Determinant: det,
	}, true
}

func sortedHits(hits []Hit) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].ID != hits[j].ID {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].T < hits[j].T
	})
}

func nearestHit(hits []Hit) (Hit, bool) {
	if len(hits) == 0 {
		return Hit{}, false
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if h.T < best.T || (h.T == best.T && h.ID < best.ID) {
			best = h
		}
	}
	return best, true
}
