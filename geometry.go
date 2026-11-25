package main

import "math"

type Ray struct {
	Orig Vec3
	Dir  Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.Orig.Add(r.Dir.Mul(t))
}

type Material struct {
	Albedo      Vec3
	Emission    Vec3
	Metallic    float64
	Shininess   float64
	Unlit       bool
	Texture     *Texture
	TexStrength float64
}

type Hit struct {
	T      float64
	P      Vec3
	Normal Vec3
	U, V   float64
	Mat    *Material
}

type Object interface {
	Hit(r Ray, tMin, tMax float64) (Hit, bool)
}

type Sphere struct {
	Center Vec3
	Radius float64
	Mat    *Material
}

func (s *Sphere) Hit(r Ray, tMin, tMax float64) (Hit, bool) {
	oc := r.Orig.Sub(s.Center)
	b := oc.Dot(r.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return Hit{}, false
	}
	root := math.Sqrt(disc)
	t := -b - root
	if t < tMin || t > tMax {
		t = -b + root
		if t < tMin || t > tMax {
			return Hit{}, false
		}
	}
	p := r.At(t)
	n := p.Sub(s.Center).Div(s.Radius)
	return Hit{T: t, P: p, Normal: n, Mat: s.Mat}, true
}

// Rod is a vertical cylinder with closed caps.
type Rod struct {
	X, Z       float64
	Radius     float64
	YMin, YMax float64
	Mat        *Material
}

func (rod *Rod) Hit(r Ray, tMin, tMax float64) (Hit, bool) {
	nearest := math.Inf(1)
	var result Hit
	found := false

	ox, oz := r.Orig.X-rod.X, r.Orig.Z-rod.Z
	dx, dz := r.Dir.X, r.Dir.Z
	a := dx*dx + dz*dz
	if a > 1e-12 {
		b := 2 * (ox*dx + oz*dz)
		c := ox*ox + oz*oz - rod.Radius*rod.Radius
		disc := b*b - 4*a*c
		if disc >= 0 {
			sq := math.Sqrt(disc)
			for _, t := range []float64{(-b - sq) / (2 * a), (-b + sq) / (2 * a)} {
				if t < tMin || t > tMax || t >= nearest {
					continue
				}
				y := r.Orig.Y + t*r.Dir.Y
				if y < rod.YMin || y > rod.YMax {
					continue
				}
				p := r.At(t)
				n := V(p.X-rod.X, 0, p.Z-rod.Z).Norm()
				nearest, found = t, true
				result = Hit{T: t, P: p, Normal: n, Mat: rod.Mat}
			}
		}
	}

	if math.Abs(r.Dir.Y) > 1e-12 {
		for _, yCap := range []float64{rod.YMin, rod.YMax} {
			t := (yCap - r.Orig.Y) / r.Dir.Y
			if t < tMin || t > tMax || t >= nearest {
				continue
			}
			p := r.At(t)
			dxCap, dzCap := p.X-rod.X, p.Z-rod.Z
			if dxCap*dxCap+dzCap*dzCap > rod.Radius*rod.Radius {
				continue
			}
			n := V(0, 1, 0)
			if yCap == rod.YMin {
				n = V(0, -1, 0)
			}
			nearest, found = t, true
			result = Hit{T: t, P: p, Normal: n, Mat: rod.Mat, U: dxCap/(2*rod.Radius) + 0.5, V: dzCap/(2*rod.Radius) + 0.5}
		}
	}
	return result, found
}

// Quad is an oriented parallelogram. UAxis and VAxis are full edge vectors.
type Quad struct {
	Origin Vec3
	UAxis  Vec3
	VAxis  Vec3
	Mat    *Material
}

func (q *Quad) Hit(r Ray, tMin, tMax float64) (Hit, bool) {
	n := q.UAxis.Cross(q.VAxis).Norm()
	denom := n.Dot(r.Dir)
	if math.Abs(denom) < 1e-12 {
		return Hit{}, false
	}
	t := q.Origin.Sub(r.Orig).Dot(n) / denom
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	p := r.At(t)
	rel := p.Sub(q.Origin)
	uu := q.UAxis.Dot(q.UAxis)
	vv := q.VAxis.Dot(q.VAxis)
	u := rel.Dot(q.UAxis) / uu
	v := rel.Dot(q.VAxis) / vv
	if u < 0 || u > 1 || v < 0 || v > 1 {
		return Hit{}, false
	}
	if denom > 0 {
		n = n.Mul(-1)
	}
	return Hit{T: t, P: p, Normal: n, U: u, V: v, Mat: q.Mat}, true
}

type Scene struct {
	Objects         []Object
	BackgroundColor func(Ray) Vec3
}

func (s *Scene) Add(o Object) *Scene {
	s.Objects = append(s.Objects, o)
	return s
}

func (s *Scene) Nearest(r Ray) (Hit, bool) {
	tMax := math.Inf(1)
	var best Hit
	found := false
	for _, o := range s.Objects {
		if h, ok := o.Hit(r, 0.0001, tMax); ok {
			best = h
			tMax = h.T
			found = true
		}
	}
	return best, found
}
