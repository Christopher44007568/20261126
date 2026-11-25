package main

import "math"

type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{X: v.X + o.X, Y: v.Y + o.Y}
}

func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{X: v.X - o.X, Y: v.Y - o.Y}
}

func (v Vec2) Scale(s float64) Vec2 {
	return Vec2{X: v.X * s, Y: v.Y * s}
}

func (v Vec2) Dot(o Vec2) float64 {
	return v.X*o.X + v.Y*o.Y
}

func (v Vec2) Cross(o Vec2) float64 {
	return v.X*o.Y - v.Y*o.X
}

func (v Vec2) Length() float64 {
	return math.Hypot(v.X, v.Y)
}

type AABB struct {
	Min Vec2
	Max Vec2
}

func EmptyAABB() AABB {
	inf := math.Inf(1)
	return AABB{
		Min: Vec2{X: inf, Y: inf},
		Max: Vec2{X: -inf, Y: -inf},
	}
}

func (b AABB) IsEmpty() bool {
	return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y
}

func (b AABB) AddPoint(p Vec2) AABB {
	if p.X < b.Min.X {
		b.Min.X = p.X
	}
	if p.Y < b.Min.Y {
		b.Min.Y = p.Y
	}
	if p.X > b.Max.X {
		b.Max.X = p.X
	}
	if p.Y > b.Max.Y {
		b.Max.Y = p.Y
	}
	return b
}

func (b AABB) Union(o AABB) AABB {
	if b.IsEmpty() {
		return o
	}
	if o.IsEmpty() {
		return b
	}
	return AABB{
		Min: Vec2{X: math.Min(b.Min.X, o.Min.X), Y: math.Min(b.Min.Y, o.Min.Y)},
		Max: Vec2{X: math.Max(b.Max.X, o.Max.X), Y: math.Max(b.Max.Y, o.Max.Y)},
	}
}

func (b AABB) Center() Vec2 {
	return Vec2{X: (b.Min.X + b.Max.X) * 0.5, Y: (b.Min.Y + b.Max.Y) * 0.5}
}

func (b AABB) Size() Vec2 {
	return Vec2{X: b.Max.X - b.Min.X, Y: b.Max.Y - b.Min.Y}
}

func (b AABB) Area() float64 {
	size := b.Size()
	return math.Max(0, size.X) * math.Max(0, size.Y)
}

func (b AABB) Translated(delta Vec2) AABB {
	return AABB{Min: b.Min.Add(delta), Max: b.Max.Add(delta)}
}

func (b AABB) Intersection(o AABB) AABB {
	return AABB{
		Min: Vec2{X: math.Max(b.Min.X, o.Min.X), Y: math.Max(b.Min.Y, o.Min.Y)},
		Max: Vec2{X: math.Min(b.Max.X, o.Max.X), Y: math.Min(b.Max.Y, o.Max.Y)},
	}
}

func (b AABB) ContainsPoint(p Vec2) bool {
	return !b.IsEmpty() &&
		p.X >= b.Min.X && p.X <= b.Max.X &&
		p.Y >= b.Min.Y && p.Y <= b.Max.Y
}

func (b AABB) Overlaps(o AABB) bool {
	return !b.IsEmpty() && !o.IsEmpty() &&
		b.Min.X <= o.Max.X && b.Max.X >= o.Min.X &&
		b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y
}

type Triangle struct {
	A    Vec2
	B    Vec2
	C    Vec2
	Part int
	Tone float64
}

func (t Triangle) Bounds() AABB {
	return EmptyAABB().AddPoint(t.A).AddPoint(t.B).AddPoint(t.C)
}

func (t Triangle) Centroid() Vec2 {
	return Vec2{
		X: (t.A.X + t.B.X + t.C.X) / 3,
		Y: (t.A.Y + t.B.Y + t.C.Y) / 3,
	}
}

func signedArea(a, b, c Vec2) float64 {
	return (b.Sub(a)).Cross(c.Sub(a)) * 0.5
}

func (t Triangle) Contains(p Vec2) bool {
	area := signedArea(t.A, t.B, t.C)
	if math.Abs(area) < 1e-12 {
		return false
	}
	a := signedArea(t.A, t.B, p)
	b := signedArea(t.B, t.C, p)
	c := signedArea(t.C, t.A, p)
	if area < 0 {
		a, b, c = -a, -b, -c
	}
	const eps = -1e-10
	return a >= eps && b >= eps && c >= eps
}

type Mesh struct {
	Triangles []Triangle
}

func (m Mesh) Bounds() AABB {
	bounds := EmptyAABB()
	for i := range m.Triangles {
		bounds = bounds.Union(m.Triangles[i].Bounds())
	}
	return bounds
}

func (m Mesh) TriangleAABB(i int) AABB {
	return m.Triangles[i].Bounds()
}
