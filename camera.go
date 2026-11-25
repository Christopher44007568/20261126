package rayao

import "math"

// Camera is a simple perspective pinhole camera.
type Camera struct {
	Position Vec3
	Forward  Vec3
	Right    Vec3
	Up       Vec3
	TanFovY  float64
}

// NewCamera builds an orthonormal view basis. fovDegrees is vertical FOV.
func NewCamera(position, target, worldUp Vec3, fovDegrees float64) Camera {
	forward := target.Sub(position).Normalize()
	right := forward.Cross(worldUp).Normalize()
	up := right.Cross(forward).Normalize()
	return Camera{
		Position: position,
		Forward:  forward,
		Right:    right,
		Up:       up,
		TanFovY:  math.Tan(fovDegrees * math.Pi / 360.0),
	}
}

// PrimaryRay returns the normalized ray through the center of a pixel.
func (c Camera) PrimaryRay(px, py, width, height int) Ray {
	u := (float64(px)+0.5)/float64(width)*2.0 - 1.0
	v := 1.0 - (float64(py)+0.5)/float64(height)*2.0
	aspect := float64(width) / float64(height)
	direction := c.Forward.
		Add(c.Right.Mul(u * c.TanFovY * aspect)).
		Add(c.Up.Mul(v * c.TanFovY)).
		Normalize()
	return Ray{
		Origin:    c.Position,
		Direction: direction,
		MinT:      0,
		MaxT:      math.Inf(1),
	}
}
