package main

import "math"

type BodyPart struct {
	Name  string
	Color RGB
}

var bodyParts = []BodyPart{
	{Name: "rear leg", Color: RGB{R: 48, G: 66, B: 106}},
	{Name: "rear arm", Color: RGB{R: 51, G: 91, B: 137}},
	{Name: "legs", Color: RGB{R: 58, G: 79, B: 126}},
	{Name: "torso", Color: RGB{R: 42, G: 119, B: 125}},
	{Name: "head", Color: RGB{R: 233, G: 174, B: 124}},
	{Name: "front arm", Color: RGB{R: 78, G: 145, B: 187}},
	{Name: "front leg", Color: RGB{R: 78, G: 105, B: 157}},
}

type EllipsePart struct {
	Center Vec2
	RX     float64
	RY     float64
	Rotate float64
	Part   int
}

type SoftCharacter struct {
	Leaf          []EllipsePart
	MaxBend       float64
	PivotY        float64
	BendHeight    float64
	RestTriangles []Triangle
}

func NewSoftCharacter() *SoftCharacter {
	return &SoftCharacter{
		MaxBend:    0.86,
		PivotY:     -1.28,
		BendHeight: 2.35,
		Leaf: []EllipsePart{
			{Center: Vec2{X: -0.31, Y: -0.94}, RX: 0.18, RY: 0.49, Rotate: 0.06, Part: 0},
			{Center: Vec2{X: -0.73, Y: 0.11}, RX: 0.39, RY: 0.12, Rotate: -0.16, Part: 1},
			{Center: Vec2{X: 0.00, Y: -0.90}, RX: 0.27, RY: 0.48, Rotate: 0, Part: 2},
			{Center: Vec2{X: 0.28, Y: -0.94}, RX: 0.18, RY: 0.49, Rotate: -0.06, Part: 0},
			{Center: Vec2{X: 0.00, Y: -0.05}, RX: 0.43, RY: 0.57, Rotate: 0, Part: 3},
			{Center: Vec2{X: 0.00, Y: 0.63}, RX: 0.29, RY: 0.32, Rotate: 0, Part: 4},
			{Center: Vec2{X: 0.73, Y: 0.11}, RX: 0.39, RY: 0.12, Rotate: 0.16, Part: 5},
			{Center: Vec2{X: 0.36, Y: -0.94}, RX: 0.18, RY: 0.49, Rotate: 0.03, Part: 6},
		},
	}
}

func (c *SoftCharacter) Pose(progress float64) Mesh {
	progress = math.Max(0, math.Min(1, progress))
	// The remaining bend follows a smooth release curve. progress=0 is the
	// strongly bent bind pose; progress=1 is the stretched final pose.
	remaining := 1 - progress
	bend := c.MaxBend * remaining * remaining * (3 - 2*remaining)

	mesh := Mesh{Triangles: make([]Triangle, 0, 8*28)}
	for _, part := range c.Leaf {
		partMesh := makeEllipse(part)
		for i := range partMesh.Triangles {
			tri := partMesh.Triangles[i]
			tri.A = c.bendPoint(tri.A, bend)
			tri.B = c.bendPoint(tri.B, bend)
			tri.C = c.bendPoint(tri.C, bend)
			mesh.Triangles = append(mesh.Triangles, tri)
		}
	}
	return mesh
}

func (c *SoftCharacter) bendPoint(p Vec2, bend float64) Vec2 {
	d := p.Sub(Vec2{X: 0, Y: c.PivotY})
	t := d.Y / c.BendHeight
	t = math.Max(0, math.Min(1, t))
	angle := bend * t
	co, si := math.Cos(angle), math.Sin(angle)
	return Vec2{
		X: co*d.X - si*d.Y,
		Y: c.PivotY + si*d.X + co*d.Y,
	}
}

func makeEllipse(part EllipsePart) Mesh {
	const segments = 28
	outline := make([]Vec2, segments)
	for i := range outline {
		a := 2 * math.Pi * float64(i) / segments
		co, si := math.Cos(a), math.Sin(a)
		x := part.RX * co
		y := part.RY * si
		rc, rs := math.Cos(part.Rotate), math.Sin(part.Rotate)
		outline[i] = Vec2{
			X: part.Center.X + rc*x - rs*y,
			Y: part.Center.Y + rs*x + rc*y,
		}
	}

	mesh := Mesh{Triangles: make([]Triangle, 0, segments)}
	for i := range outline {
		j := (i + 1) % segments
		mesh.Triangles = append(mesh.Triangles, Triangle{
			A:    part.Center,
			B:    outline[i],
			C:    outline[j],
			Part: part.Part,
			Tone: math.Sin(2*math.Pi*float64(i)/segments) * 0.5,
		})
	}
	return mesh
}

func (c *SoftCharacter) Rest() Mesh {
	return c.Pose(1)
}

func colorForTriangle(mesh Mesh, triangleIndex int) RGB {
	if triangleIndex < 0 || triangleIndex >= len(mesh.Triangles) {
		return RGB{R: 20, G: 25, B: 36}
	}
	tri := mesh.Triangles[triangleIndex]
	part := 0
	if tri.Part >= 0 && tri.Part < len(bodyParts) {
		part = tri.Part
	}
	base := bodyParts[part].Color
	k := 0.82 + tri.Tone*0.18
	return RGB{
		R: clampByte(float64(base.R) * k),
		G: clampByte(float64(base.G) * k),
		B: clampByte(float64(base.B) * k),
	}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
