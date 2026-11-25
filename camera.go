package main

import "math"

type Camera struct {
	Eye           Vec3
	Target        Vec3
	Forward       Vec3
	Right         Vec3
	Up            Vec3
	FOV           float64
	Aspect        float64
	tanHalf       float64
	Width, Height int
}

func NewCamera(eye, target, worldUp Vec3, fovDeg, aspect float64, width, height int) Camera {
	forward := target.Sub(eye).Norm()
	right := forward.Cross(worldUp).Norm()
	up := right.Cross(forward).Norm()
	return Camera{
		Eye:     eye,
		Target:  target,
		Forward: forward,
		Right:   right,
		Up:      up,
		FOV:     fovDeg,
		Aspect:  aspect,
		tanHalf: math.Tan(deg(fovDeg) * 0.5),
		Width:   width,
		Height:  height,
	}
}

func (c Camera) RayAt(x, y float64) Ray {
	ndcX := (x/float64(c.Width))*2 - 1
	ndcY := 1 - (y/float64(c.Height))*2
	dir := c.Forward.
		Add(c.Right.Mul(ndcX * c.tanHalf * c.Aspect)).
		Add(c.Up.Mul(ndcY * c.tanHalf)).
		Norm()
	return NewRay(c.Eye, dir)
}

func (c Camera) Project(p Vec3) (x, y, depth float64, visible bool) {
	rel := p.Sub(c.Eye)
	depth = rel.Dot(c.Forward)
	if depth <= 0.05 {
		return 0, 0, depth, false
	}
	viewX := rel.Dot(c.Right)
	viewY := rel.Dot(c.Up)
	ndcX := (viewX / depth) / (c.tanHalf * c.Aspect)
	ndcY := (viewY / depth) / c.tanHalf
	x = (ndcX*0.5 + 0.5) * float64(c.Width)
	y = (1 - (ndcY*0.5 + 0.5)) * float64(c.Height)
	visible = x > -100 && x < float64(c.Width)+100 && y > -100 && y < float64(c.Height)+100
	return x, y, depth, visible
}
