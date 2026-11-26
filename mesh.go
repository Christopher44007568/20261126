package main

import "math"

type Material struct {
	Name  string
	Base  RGB
	Sheen float64
}

type Triangle struct {
	A, B, C  Vec3
	ID       int
	Material int
}

type Mesh struct {
	Name      string
	Detail    int
	Tris      []Triangle
	Materials []Material
}

func DefaultMaterials() []Material {
	return []Material{
		{Name: "gold", Base: RGB{244, 190, 74}, Sheen: 0.18},
		{Name: "amber", Base: RGB{218, 116, 42}, Sheen: 0.10},
		{Name: "ruby", Base: RGB{177, 47, 70}, Sheen: 0.28},
		{Name: "violet", Base: RGB{103, 76, 158}, Sheen: 0.22},
		{Name: "teal", Base: RGB{43, 137, 143}, Sheen: 0.20},
	}
}

// NewCrest builds a radial faceted sun-crest. detail is the number of angular
// samples; alternating outer radii create spikes. A small detail value is the
// deliberate low-poly fallback.
func NewCrest(detail int, name string) *Mesh {
	if detail < 6 {
		detail = 6
	}
	mesh := &Mesh{Name: name, Detail: detail, Materials: DefaultMaterials()}
	n := detail

	center := Vec3{0, 0, 0.92}
	ring0 := make([]Vec3, n)
	ring1 := make([]Vec3, n)
	ring2 := make([]Vec3, n)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		c, s := math.Cos(a), math.Sin(a)
		ring0[i] = Vec3{0.43*c + 0.08, 0.43*s + 0.10, 0.58}
		ring1[i] = Vec3{0.84*c - 0.05, 0.84*s + 0.04, 0.28}
		outer := 1.58
		if i%2 == 1 {
			outer = 1.06
		}
		ring2[i] = Vec3{outer * c, outer*s - 0.08, -0.04}
	}

	add := func(a, b, c Vec3, mat int) {
		id := len(mesh.Tris)
		mesh.Tris = append(mesh.Tris, Triangle{A: a, B: b, C: c, ID: id, Material: mat})
	}

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		add(center, ring0[i], ring0[j], i%2)

		mat1 := 2
		if i%3 == 0 {
			mat1 = 3
		}
		add(ring0[i], ring1[i], ring1[j], mat1)
		add(ring0[i], ring1[j], ring0[j], mat1)

		mat2 := 4
		if i%2 == 0 {
			mat2 = 1
		}
		add(ring1[i], ring2[i], ring2[j], mat2)
		add(ring1[i], ring2[j], ring1[j], mat2)
	}
	return mesh
}

func (m *Mesh) Bounds() AABB {
	if len(m.Tris) == 0 {
		return AABB{}
	}
	b := triangleAABB(m.Tris[0])
	for i := range m.Tris[1:] {
		b = boundsUnion(b, triangleAABB(m.Tris[i+1]))
	}
	return b
}

func triangleAABB(t Triangle) AABB {
	return AABB{
		Min: vMin(vMin(t.A, t.B), t.C),
		Max: vMax(vMax(t.A, t.B), t.C),
	}
}

func triangleNormal(t Triangle) Vec3 {
	n := vCross(vSub(t.B, t.A), vSub(t.C, t.A))
	return vNorm(n)
}
