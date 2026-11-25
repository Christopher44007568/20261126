package rayao

// Triangle is a single double-sided geometric primitive. Winding determines
// GeometricNormal; shading normals are flipped toward the incident ray.
type Triangle struct {
	Vertices [3]Vec3
	Normal   Vec3
	Material int
}

// NewTriangle constructs a triangle and its unit geometric normal.
func NewTriangle(a, b, c Vec3, material int) Triangle {
	return Triangle{
		Vertices: [3]Vec3{a, b, c},
		Normal:   b.Sub(a).Cross(c.Sub(a)).Normalize(),
		Material: material,
	}
}

// Hit describes one accepted ray/triangle intersection.
type Hit struct {
	T               float64
	U               float64
	V               float64
	Point           Vec3
	Normal          Vec3
	GeometricNormal Vec3
	Primitive       int
	Material        int
	FrontFacing     bool
}

func (t Triangle) hitForRay(r Ray, id int, distance, u, v float64) Hit {
	front := r.Direction.Dot(t.Normal) <= 0
	normal := t.Normal
	if !front {
		normal = t.Normal.Mul(-1)
	}
	return Hit{
		T:               distance,
		U:               u,
		V:               v,
		Point:           r.PointAt(distance),
		Normal:          normal,
		GeometricNormal: t.Normal,
		Primitive:       id,
		Material:        t.Material,
		FrontFacing:     front,
	}
}
