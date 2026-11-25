package internal

import "math"

// Camera is a pinhole camera. A pinhole shares one origin for every pixel, so
// dO/dx = dO/dy = 0; only the direction carries a differential.
type Camera struct {
	Origin     Vec3
	Forward    Vec3
	Right      Vec3
	Up         Vec3
	Width      int
	Height     int
	TanHalfFov float64
}

func NewCamera(origin, target Vec3, fovDeg float64, width, height int) Camera {
	fwd := target.Sub(origin).Normalize()
	right := fwd.Cross(V(0, 1, 0)).Normalize()
	up := right.Cross(fwd).Normalize()
	return Camera{
		Origin:     origin,
		Forward:    fwd,
		Right:      right,
		Up:         up,
		Width:      width,
		Height:     height,
		TanHalfFov: math.Tan(fovDeg * math.Pi / 360),
	}
}

// GenerateRay builds the primary ray for pixel (px, py) together with its
// per-pixel direction differentials.
//
// Screen coordinates s live in [-1, 1]. For a pixel step,
//
//	dsx/dx = 2/width,  dsy/dy = -2/height
//	D  = fwd + tan(fov/2)*(sx*right + sy*up)
//	dD/dx = tan(fov/2)*(2/width)*right
//	dD/dy = tan(fov/2)*(-2/height)*up
func (c Camera) GenerateRay(px, py int) Ray {
	sx := (2*float64(px)+1)/float64(c.Width) - 1
	sy := 1 - (2*float64(py)+1)/float64(c.Height)

	dir := c.Forward.
		Add(c.Right.Scale(sx * c.TanHalfFov)).
		Add(c.Up.Scale(sy * c.TanHalfFov)).
		Normalize()

	ddx := c.Right.Scale(c.TanHalfFov * 2 / float64(c.Width))
	ddy := c.Up.Scale(-c.TanHalfFov * 2 / float64(c.Height))

	return Ray{
		Origin:  c.Origin,
		Dir:     dir,
		DDirX:   ddx,
		DDirY:   ddy,
		HasDiff: true,
	}
}
