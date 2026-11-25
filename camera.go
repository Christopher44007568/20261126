package main

import "math"

type Camera struct {
	Width   int
	Height  int
	Eye     Vec3
	Target  Vec3
	Forward Vec3
	Right   Vec3
	Up      Vec3
	Focal   float64
}

func NewCamera(width, height int) Camera {
	eye := V(0.0, 0.18, 7.4)
	target := V(0.0, -0.02, 0.02)
	forward := target.Sub(eye).Normalized()
	right := forward.Cross(V(0, 1, 0)).Normalized()
	up := right.Cross(forward).Normalized()
	fov := 39.0 * math.Pi / 180.0
	focal := float64(height) / (2 * math.Tan(fov/2))
	return Camera{
		Width:   width,
		Height:  height,
		Eye:     eye,
		Target:  target,
		Forward: forward,
		Right:   right,
		Up:      up,
		Focal:   focal,
	}
}

func (c Camera) Ray(x, y int) Ray {
	sx := float64(x) + 0.5 - float64(c.Width)/2
	sy := float64(y) + 0.5 - float64(c.Height)/2
	dir := c.Forward.
		Add(c.Right.Mul(sx / c.Focal)).
		Add(c.Up.Mul(-sy / c.Focal)).
		Normalized()
	return Ray{Origin: c.Eye, Dir: dir, TMin: 1e-6, TMax: math.Inf(1)}
}

func (c Camera) Project(p Vec3) (float64, float64, bool) {
	rel := p.Sub(c.Eye)
	depth := rel.Dot(c.Forward)
	if depth <= 0.05 {
		return 0, 0, false
	}
	x := float64(c.Width)/2 + rel.Dot(c.Right)*c.Focal/depth
	y := float64(c.Height)/2 - rel.Dot(c.Up)*c.Focal/depth
	return x, y, true
}
