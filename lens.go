package main

import "math"

// Aperture is a polygonal thin-lens aperture. Radius is the circumradius in
// camera/world units; FNumber and FocalLength are retained for CoC reporting.
type Aperture struct {
	Blades      int
	Radius      float64
	FNumber     float64
	FocalLength float64
	Rotation    float64
}

func NewAperture(blades int, focalLength, fNumber float64) Aperture {
	if blades < 3 {
		blades = 6
	}
	if fNumber <= 0 {
		fNumber = 1.4
	}
	diameter := focalLength / fNumber
	return Aperture{
		Blades:      blades,
		Radius:      diameter * 0.5,
		FNumber:     fNumber,
		FocalLength: focalLength,
		Rotation:    math.Pi / float64(blades),
	}
}

func RegularAperture(blades int, radius float64) Aperture {
	if blades < 3 {
		blades = 6
	}
	return Aperture{Blades: blades, Radius: radius, Rotation: math.Pi / float64(blades)}
}

func (a Aperture) vertex(i int) (float64, float64) {
	angle := a.Rotation + 2*math.Pi*float64(i)/float64(a.Blades)
	return math.Cos(angle) * a.Radius, math.Sin(angle) * a.Radius
}

// Contains reports whether a lens-plane point is inside the convex aperture.
func (a Aperture) Contains(x, y float64) bool {
	if a.Radius <= 0 {
		return x == 0 && y == 0
	}
	n := a.Blades
	if n < 3 {
		n = 6
	}
	for i := 0; i < n; i++ {
		x1, y1 := a.vertex(i)
		x2, y2 := a.vertex((i + 1) % n)
		ex, ey := x2-x1, y2-y1
		cross := ex*(y-y1) - ey*(x-x1)
		if cross < -1e-9 {
			return false
		}
	}
	return true
}

// Sample maps u,v in [0,1) to a uniformly distributed point in the aperture.
func (a Aperture) Sample(u, v float64) (float64, float64) {
	if a.Radius <= 0 {
		return 0, 0
	}
	n := a.Blades
	if n < 3 {
		n = 6
	}
	scaled := clamp(u, 0, 0.999999999) * float64(n)
	tri := int(scaled)
	localU := scaled - float64(tri)
	ax, ay := a.vertex(tri % n)
	bx, by := a.vertex((tri + 1) % n)
	r := math.Sqrt(clamp(v, 0, 1))
	w0 := 1 - r
	w1 := r * (1 - localU)
	w2 := r * localU
	return w0*0 + w1*ax + w2*bx, w0*0 + w1*ay + w2*by
}

// SignedCircleOfConfusion returns the signed sensor-plane CoC diameter.
// Distances and focalLength must use the same unit. The absolute value is the
// ordinary CoC; its sign separates near-focus and far-focus directions.
func SignedCircleOfConfusion(subjectDistance, focusDistance, focalLength, apertureDiameter float64) float64 {
	if subjectDistance <= focalLength || focusDistance <= focalLength {
		return math.Inf(1)
	}
	imageFocus := focalLength * focusDistance / (focusDistance - focalLength)
	imageSubject := focalLength * subjectDistance / (subjectDistance - focalLength)
	return apertureDiameter * (imageFocus - imageSubject) / imageSubject
}

func CircleOfConfusion(subjectDistance, focusDistance, focalLength, apertureDiameter float64) float64 {
	return math.Abs(SignedCircleOfConfusion(subjectDistance, focusDistance, focalLength, apertureDiameter))
}

// CoCRadiusPixels converts a sensor-plane CoC diameter to pixel radius.
func CoCRadiusPixels(subjectDistance, focusDistance, focalLength, fNumber, sensorHeight float64, imageHeight int) float64 {
	aperture := focalLength / fNumber
	coc := CircleOfConfusion(subjectDistance, focusDistance, focalLength, aperture)
	pixelPitch := sensorHeight / float64(imageHeight)
	return coc / pixelPitch * 0.5
}
