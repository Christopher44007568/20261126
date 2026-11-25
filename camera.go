package main

import "math"

type Camera struct {
	Eye           Vec3
	LookAt        Vec3
	Up            Vec3
	FOVDegrees    float64
	Aspect        float64
	Aperture      Aperture
	FocusDistance float64

	forward Vec3
	right   Vec3
	up      Vec3
}

func NewCamera(eye, lookAt, up Vec3, fovDegrees, aspect, focusDistance float64, aperture Aperture) *Camera {
	c := &Camera{
		Eye:           eye,
		LookAt:        lookAt,
		Up:            up,
		FOVDegrees:    fovDegrees,
		Aspect:        aspect,
		Aperture:      aperture,
		FocusDistance: focusDistance,
	}
	w := eye.Sub(lookAt).Norm()
	c.right = w.Cross(up).Norm()
	c.up = c.right.Cross(w).Norm()
	c.forward = w.Mul(-1)
	return c
}

func (c *Camera) ray(s, t, lensU, lensV float64) Ray {
	halfH := c.FocusDistance * math.Tan(c.FOVDegrees*math.Pi/360)
	halfW := c.Aspect * halfH
	target := c.Eye.
		Add(c.forward.Mul(c.FocusDistance)).
		Add(c.right.Mul((s - 0.5) * 2 * halfW)).
		Add(c.up.Mul((t - 0.5) * 2 * halfH))

	lx, ly := c.Aperture.Sample(lensU, lensV)
	origin := c.Eye.Add(c.right.Mul(lx)).Add(c.up.Mul(ly))
	dir := target.Sub(origin).Norm()
	return Ray{Orig: origin, Dir: dir}
}

// Project maps a world point to top-left pixel coordinates.
func (c *Camera) Project(p Vec3, width, height int) (float64, float64, float64, bool) {
	rel := p.Sub(c.Eye)
	depth := rel.Dot(c.forward)
	if depth <= 0.01 {
		return 0, 0, depth, false
	}
	x := rel.Dot(c.right)
	y := rel.Dot(c.up)
	halfH := depth * math.Tan(c.FOVDegrees*math.Pi/360)
	halfW := c.Aspect * halfH
	s := 0.5 + x/(2*halfW)
	t := 0.5 + y/(2*halfH)
	return s * float64(width), (1 - t) * float64(height), depth, s >= -0.1 && s <= 1.1 && t >= -0.1 && t <= 1.1
}

// SensorHeight derives the sensor size that makes FocalLength and FOVDegrees
// describe the same pinhole field of view. Units match Aperture.FocalLength.
func (c *Camera) SensorHeight() float64 {
	return 2 * c.Aperture.FocalLength * math.Tan(c.FOVDegrees*math.Pi/360)
}

// CircleOfConfusionPixels returns the CoC radius at a forward depth in pixels.
func (c *Camera) CircleOfConfusionPixels(depth float64, imageHeight int) float64 {
	return CoCRadiusPixels(
		depth,
		c.FocusDistance,
		c.Aperture.FocalLength,
		c.Aperture.FNumber,
		c.SensorHeight(),
		imageHeight,
	)
}
