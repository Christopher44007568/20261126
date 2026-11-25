package main

import "math"

const rayEpsilon = 1e-5

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Dir.Mul(t))
}

// Shape is geometry only. Material choice lives in the hit-group mapping.
type Shape interface {
	intersect(ray Ray, tMin, tMax float64) (t float64, normal Vec3, ok bool)
	samplePoint() Vec3
}

type Quad struct {
	Corner Vec3
	EdgeU  Vec3
	EdgeV  Vec3
	Normal Vec3
}

func (q Quad) intersect(ray Ray, tMin, tMax float64) (float64, Vec3, bool) {
	denom := q.Normal.Dot(ray.Dir)
	if math.Abs(denom) < 1e-9 {
		return 0, Vec3{}, false
	}
	t := q.Normal.Dot(q.Corner.Sub(ray.Origin)) / denom
	if t < tMin || t > tMax {
		return 0, Vec3{}, false
	}
	rel := ray.At(t).Sub(q.Corner)
	uu := q.EdgeU.Dot(q.EdgeU)
	vv := q.EdgeV.Dot(q.EdgeV)
	if uu < 1e-12 || vv < 1e-12 {
		return 0, Vec3{}, false
	}
	u := rel.Dot(q.EdgeU) / uu
	v := rel.Dot(q.EdgeV) / vv
	if u < 0 || u > 1 || v < 0 || v > 1 {
		return 0, Vec3{}, false
	}
	return t, q.Normal.Normalize(), true
}

func (q Quad) samplePoint() Vec3 {
	return q.Corner.Add(q.EdgeU.Mul(0.5)).Add(q.EdgeV.Mul(0.5))
}

type Sphere struct {
	Center Vec3
	Radius float64
}

func (s Sphere) intersect(ray Ray, tMin, tMax float64) (float64, Vec3, bool) {
	oc := ray.Origin.Sub(s.Center)
	a := ray.Dir.Dot(ray.Dir)
	halfB := oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := halfB*halfB - a*c
	if disc < 0 {
		return 0, Vec3{}, false
	}
	root := math.Sqrt(disc)
	t := (-halfB - root) / a
	if t < tMin || t > tMax {
		t = (-halfB + root) / a
		if t < tMin || t > tMax {
			return 0, Vec3{}, false
		}
	}
	n := ray.At(t).Sub(s.Center).Mul(1 / s.Radius).Normalize()
	return t, n, true
}

func (s Sphere) samplePoint() Vec3 { return s.Center }

type Box struct {
	Min Vec3
	Max Vec3
}

func (b Box) intersect(ray Ray, tMin, tMax float64) (float64, Vec3, bool) {
	tNear, tFar := tMin, tMax
	nearAxis, nearSign := -1, 1.0
	origin := [3]float64{ray.Origin.X, ray.Origin.Y, ray.Origin.Z}
	direction := [3]float64{ray.Dir.X, ray.Dir.Y, ray.Dir.Z}
	lower := [3]float64{b.Min.X, b.Min.Y, b.Min.Z}
	upper := [3]float64{b.Max.X, b.Max.Y, b.Max.Z}

	for axis := 0; axis < 3; axis++ {
		if math.Abs(direction[axis]) < 1e-12 {
			if origin[axis] < lower[axis] || origin[axis] > upper[axis] {
				return 0, Vec3{}, false
			}
			continue
		}
		inv := 1 / direction[axis]
		t0 := (lower[axis] - origin[axis]) * inv
		t1 := (upper[axis] - origin[axis]) * inv
		sign := -1.0
		if t0 > t1 {
			t0, t1 = t1, t0
			sign = 1
		}
		if t0 > tNear {
			tNear = t0
			nearAxis = axis
			nearSign = sign
		}
		if t1 < tFar {
			tFar = t1
		}
		if tNear > tFar {
			return 0, Vec3{}, false
		}
	}
	if nearAxis < 0 || tNear < tMin || tNear > tMax {
		return 0, Vec3{}, false
	}
	normal := Vec3{}
	switch nearAxis {
	case 0:
		normal.X = nearSign
	case 1:
		normal.Y = nearSign
	case 2:
		normal.Z = nearSign
	}
	return tNear, normal, true
}

func (b Box) samplePoint() Vec3 {
	return b.Min.Add(b.Max).Mul(0.5)
}
