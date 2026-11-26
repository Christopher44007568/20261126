package main

import "math"

type Camera struct {
	Position Vec3
	Target   Vec3
	Up       Vec3
	FOV      float64
	Width    int
	Height   int

	forward Vec3
	right   Vec3
	camUp   Vec3
	scale   float64
}

func NewCamera(position, target Vec3, fov float64, width, height int) Camera {
	c := Camera{
		Position: position,
		Target:   target,
		Up:       V(0, 1, 0),
		FOV:      fov,
		Width:    width,
		Height:   height,
	}
	c.forward = target.Sub(position).Normalize()
	c.right = c.forward.Cross(c.Up).Normalize()
	if c.right.Len2() == 0 {
		c.right = V(1, 0, 0)
	}
	c.camUp = c.right.Cross(c.forward).Normalize()
	c.scale = math.Tan(fov * math.Pi / 360)
	return c
}

func (c Camera) Aspect() float64 {
	if c.Height == 0 {
		return 1
	}
	return float64(c.Width) / float64(c.Height)
}

func (c Camera) RayAt(normalizedX, normalizedY float64) Ray {
	dir := c.forward.
		Add(c.right.Scale(normalizedX * c.scale * c.Aspect())).
		Add(c.camUp.Scale(normalizedY * c.scale)).
		Normalize()
	return MakeRay(c.Position, dir)
}

func (c Camera) PixelRay(x, y int) Ray {
	nx := (float64(x)+0.5)/float64(c.Width)*2 - 1
	ny := 1 - (float64(y)+0.5)/float64(c.Height)*2
	return c.RayAt(nx, ny)
}
