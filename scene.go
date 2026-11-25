package main

import (
	"math"
	"sort"
)

const Eps = 1e-4

type Ray struct {
	Orig Vec3
	Dir  Vec3
}

func (r Ray) At(t float64) Vec3 { return r.Orig.Add(r.Dir.Scale(t)) }

// Glass describes one dielectric volume. Absorption is a Napierian
// coefficient per unit distance; Thickness changes with the animation.
type Glass struct {
	ID         int
	IOR        float64
	Absorption Vec3
}

type Hit struct {
	T        float64
	Prim     Primitive
	Glass    *Glass
	Entering bool
	Normal   Vec3 // outward surface normal, pointing against incident ray
}

// Primitive reports every positive ray/primitive intersection. Any-hit
// traversal needs the full event list, not just a nearest hit.
type Primitive interface {
	Intersect(r Ray, out *[]Hit)
}

type Sphere struct {
	Center Vec3
	Radius float64
	Albedo Vec3
	Glass  *Glass // non-nil makes the sphere a dielectric volume
}

func (s *Sphere) Intersect(r Ray, out *[]Hit) {
	oc := r.Orig.Sub(s.Center)
	b := oc.Dot(r.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return
	}
	sq := math.Sqrt(disc)
	for _, t := range []float64{-b - sq, -b + sq} {
		if t <= Eps {
			continue
		}
		p := r.At(t)
		n := p.Sub(s.Center).Scale(1 / s.Radius)
		entering := r.Dir.Dot(n) < 0
		*out = append(*out, Hit{T: t, Prim: s, Glass: s.Glass, Entering: entering, Normal: n})
	}
}

// Ground is the y=0 plane with a procedural checker albedo.
type Ground struct{}

func (Ground) Intersect(r Ray, out *[]Hit) {
	if math.Abs(r.Dir.Y) < 1e-9 {
		return
	}
	t := -r.Orig.Y / r.Dir.Y
	if t <= Eps {
		return
	}
	n := Vec3{0, 1, 0}
	if r.Dir.Y > 0 {
		n = Vec3{0, -1, 0}
	}
	*out = append(*out, Hit{T: t, Prim: Ground{}, Normal: n})
}

// Slab is an axis-aligned glass box spanning [Min, Max].
type Slab struct {
	Min, Max Vec3
	G        *Glass
}

func (b *Slab) Intersect(r Ray, out *[]Hit) {
	tmin, tmax := 0.0, math.Inf(1)
	var nmin, nmax Vec3
	for ax := 0; ax < 3; ax++ {
		o, d, lo, hi := axis(r.Orig, ax), axis(r.Dir, ax), axis(b.Min, ax), axis(b.Max, ax)
		if math.Abs(d) < 1e-12 {
			if o < lo || o > hi {
				return
			}
			continue
		}
		tNear := (lo - o) / d
		tFar := (hi - o) / d
		nNear := withAxis(ax, -1)
		nFar := withAxis(ax, 1)
		if tNear > tFar {
			tNear, tFar = tFar, tNear
			nNear, nFar = nFar, nNear
		}
		if tNear > tmin {
			tmin, nmin = tNear, nNear
		}
		if tFar < tmax {
			tmax, nmax = tFar, nFar
		}
		if tmin > tmax {
			return
		}
	}
	originInside := r.Orig.X > b.Min.X && r.Orig.X < b.Max.X &&
		r.Orig.Y > b.Min.Y && r.Orig.Y < b.Max.Y &&
		r.Orig.Z > b.Min.Z && r.Orig.Z < b.Max.Z
	if tmin > Eps && !originInside {
		*out = append(*out, Hit{T: tmin, Prim: b, Glass: b.G, Entering: true, Normal: nmin})
	}
	if tmax > Eps {
		*out = append(*out, Hit{T: tmax, Prim: b, Glass: b.G, Entering: false, Normal: nmax})
	}
}

func axis(v Vec3, i int) float64 {
	switch i {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func withAxis(i int, s float64) Vec3 {
	v := Vec3{}
	switch i {
	case 0:
		v.X = s
	case 1:
		v.Y = s
	default:
		v.Z = s
	}
	return v
}

type Scene struct {
	Primitives []Primitive
	LightDir   Vec3
}

// AllHits collects every intersection and returns events front-to-back.
func (s *Scene) AllHits(r Ray) []Hit {
	hits := make([]Hit, 0, 8)
	for _, p := range s.Primitives {
		p.Intersect(r, &hits)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].T != hits[j].T {
			return hits[i].T < hits[j].T
		}
		// Opaque surfaces win coincident-surface ties: an opaque hit blocks
		// the ray regardless of any glass boundary sharing the same point.
		return hits[i].Glass == nil
	})
	return hits
}

// NearestHit returns the first opaque/dielectric surface a camera ray sees.
func (s *Scene) NearestHit(r Ray) (Hit, bool) {
	hits := s.AllHits(r)
	if len(hits) == 0 {
		return Hit{}, false
	}
	return hits[0], true
}

// Albedo resolves the checkerboard for the ground and constant color otherwise.
func Albedo(h Hit, p Vec3) Vec3 {
	switch v := h.Prim.(type) {
	case Ground:
		c := int(math.Floor(p.X*0.5)) + int(math.Floor(p.Z*0.5))
		if c&1 == 0 {
			return Vec3{0.75, 0.75, 0.72}
		}
		return Vec3{0.28, 0.3, 0.34}
	case *Sphere:
		return v.Albedo
	default:
		return Vec3{0.8, 0.8, 0.8}
	}
}
