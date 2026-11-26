package shadow

import "math"

// Material controls how a primitive participates in shadow rays.
type Material struct {
	Name string
	// DoubleSided means both geometric faces are shading-relevant.
	DoubleSided bool
	// Transmittance is the fraction of shadow energy kept per solid crossing.
	// 0 means a fully opaque blocker; 0.6 keeps 60% (soft/translucent shadow).
	Transmittance float64
	// Closure marks the material as a closed solid whose entry/exit hits must
	// be merged into a single per-solid energy event.
	Closure bool
}

// Primitive is a single triangle with a fixed geometric (winding) normal.
type Primitive struct {
	Material *Material
	Vertices [3]Vec3
	Normal   Vec3
}

// Hit records a ray/triangle intersection without any flag filtering.
type Hit struct {
	T     float64
	Prim  *Primitive
	Front bool
}

const rayEpsilon = 1e-9

// Intersect runs unfiltered Moller-Trumbore; face orientation is reported in
// Front so the ray-flag policy can decide what counts as a hit.
func (p *Primitive) Intersect(ray Ray) (Hit, bool) {
	e1 := p.Vertices[1].Sub(p.Vertices[0])
	e2 := p.Vertices[2].Sub(p.Vertices[0])
	pvec := ray.Direction.Cross(e2)
	det := e1.Dot(pvec)
	if math.Abs(det) < rayEpsilon {
		return Hit{}, false
	}
	invDet := 1 / det
	tvec := ray.Origin.Sub(p.Vertices[0])
	u := tvec.Dot(pvec) * invDet
	if u < -rayEpsilon || u > 1+rayEpsilon {
		return Hit{}, false
	}
	qvec := tvec.Cross(e1)
	v := ray.Direction.Dot(qvec) * invDet
	if v < -rayEpsilon || u+v > 1+rayEpsilon {
		return Hit{}, false
	}
	t := e2.Dot(qvec) * invDet
	if t <= rayEpsilon {
		return Hit{}, false
	}
	return Hit{T: t, Prim: p, Front: ray.Direction.Dot(p.Normal) < 0}, true
}

// NewQuad builds a two-triangle sheet at center spanning edge vectors u, v.
// Its winding normal equals Normalize(-Cross(u, v)).
func NewQuad(mat *Material, center, u, v Vec3) []*Primitive {
	halfU := u.Scale(0.5)
	halfV := v.Scale(0.5)
	a := center.Sub(halfU).Sub(halfV)
	b := center.Add(halfU).Sub(halfV)
	c := center.Add(halfU).Add(halfV)
	d := center.Sub(halfU).Add(halfV)
	normal := u.Cross(v).Normalize()
	return []*Primitive{
		{mat, [3]Vec3{a, b, d}, normal},
		{mat, [3]Vec3{b, c, d}, normal},
	}
}

// NewCube builds an axis-aligned closed box with outward winding normals.
func NewCube(mat *Material, center Vec3, half float64) []*Primitive {
	x, y, z := half, half, half
	corners := func(dx, dy, dz float64) Vec3 {
		return center.Add(Vec3{dx, dy, dz})
	}
	type face struct {
		normal Vec3
		points [4]Vec3
	}
	faces := []face{
		{Vec3{1, 0, 0}, [4]Vec3{corners(x, -y, -z), corners(x, y, -z), corners(x, y, z), corners(x, -y, z)}},
		{Vec3{-1, 0, 0}, [4]Vec3{corners(-x, -y, z), corners(-x, y, z), corners(-x, y, -z), corners(-x, -y, -z)}},
		{Vec3{0, 1, 0}, [4]Vec3{corners(-x, y, -z), corners(-x, y, z), corners(x, y, z), corners(x, y, -z)}},
		{Vec3{0, -1, 0}, [4]Vec3{corners(-x, -y, z), corners(-x, -y, -z), corners(x, -y, -z), corners(x, -y, z)}},
		{Vec3{0, 0, 1}, [4]Vec3{corners(-x, -y, z), corners(x, -y, z), corners(x, y, z), corners(-x, y, z)}},
		{Vec3{0, 0, -1}, [4]Vec3{corners(x, -y, -z), corners(-x, -y, -z), corners(-x, y, -z), corners(x, y, -z)}},
	}
	var prims []*Primitive
	for _, f := range faces {
		prims = append(prims,
			&Primitive{mat, [3]Vec3{f.points[0], f.points[1], f.points[3]}, f.normal},
			&Primitive{mat, [3]Vec3{f.points[1], f.points[2], f.points[3]}, f.normal},
		)
	}
	return prims
}

// Scene holds every occluder; the receiver ground is not part of the list.
type Scene struct {
	Primitives []*Primitive
	Materials  []*Material
}

func (s *Scene) Add(prims ...*Primitive) {
	s.Primitives = append(s.Primitives, prims...)
}

// ClosestHit returns the nearest unfiltered intersection, if any.
func (s *Scene) ClosestHit(ray Ray) (Hit, bool) {
	best := Hit{}
	found := false
	for _, prim := range s.Primitives {
		hit, ok := prim.Intersect(ray)
		if !ok {
			continue
		}
		if !found || hit.T < best.T {
			best, found = hit, true
		}
	}
	return best, found
}

// AllHits returns every intersection sorted front-to-back along the ray.
func (s *Scene) AllHits(ray Ray) []Hit {
	var hits []Hit
	for _, prim := range s.Primitives {
		if hit, ok := prim.Intersect(ray); ok {
			hits = append(hits, hit)
		}
	}
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j-1].T > hits[j].T; j-- {
			hits[j-1], hits[j] = hits[j], hits[j-1]
		}
	}
	return hits
}

// Material looks up a material by name; used by tests and diagnostics.
func (s *Scene) Material(name string) *Material {
	for _, mat := range s.Materials {
		if mat.Name == name {
			return mat
		}
	}
	return nil
}
