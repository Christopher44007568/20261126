package internal

import "math"

// Geometry is intersected by differential rays. Beyond the hit distance and
// shading frame, each intersector returns:
//
//	dPdx/dPdy: world-space footprint of one pixel at the hit point
//	dNdx/dNdy: derivative of the shading normal across that footprint
//
// The normal derivatives are required to correctly transfer differentials
// through a reflection event.
type Geometry interface {
	Intersect(r Ray) (t float64, p, n, dpx, dpy, dnx, dny Vec3, ok bool)
}

// Sphere with analytic differential intersection.
type Sphere struct {
	Center Vec3
	Radius float64
}

func (s Sphere) Intersect(r Ray) (float64, Vec3, Vec3, Vec3, Vec3, Vec3, Vec3, bool) {
	oc := r.Origin.Sub(s.Center)
	a := r.Dir.Dot(r.Dir)
	b := 2 * oc.Dot(r.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, false
	}
	sqrtDisc := math.Sqrt(disc)
	t := (-b - sqrtDisc) / (2 * a)
	if t <= 1e-4 {
		t = (-b + sqrtDisc) / (2 * a)
		if t <= 1e-4 {
			return 0, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, false
		}
	}

	p := r.Origin.Add(r.Dir.Scale(t))
	n := p.Sub(s.Center).Scale(1 / s.Radius)

	// Differentiate |O+tD-C|^2=R^2 for dt:
	// dt/dx = -(oc.Dot(dD) + D.Dot(dO)) / (oc+tD).Dot(D)
	denom := oc.Add(r.Dir.Scale(t)).Dot(r.Dir)
	var dtdx, dtdy float64
	if r.HasDiff && math.Abs(denom) > 1e-12 {
		dtdx = -(oc.Dot(r.DDirX) + r.Dir.Dot(r.DOriginX)) / denom
		dtdy = -(oc.Dot(r.DDirY) + r.Dir.Dot(r.DOriginY)) / denom
	}
	dpx, dpy := TransferDifferentials(r, t, dtdx, dtdy)

	// n = (P-C)/R  =>  dn = dP/R
	dnx := dpx.Scale(1 / s.Radius)
	dny := dpy.Scale(1 / s.Radius)
	return t, p, n, dpx, dpy, dnx, dny, true
}

// Plane is the y = height ground plane with unit normal +Y.
type Plane struct {
	Height float64
}

func (pl Plane) Intersect(r Ray) (float64, Vec3, Vec3, Vec3, Vec3, Vec3, Vec3, bool) {
	if math.Abs(r.Dir.Y) < 1e-12 {
		return 0, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, false
	}
	t := (pl.Height - r.Origin.Y) / r.Dir.Y
	if t <= 1e-4 {
		return 0, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, Vec3{}, false
	}

	p := r.Origin.Add(r.Dir.Scale(t))
	n := V(0, 1, 0)

	// dt/dx = -dOy/dx / Dy  (plane normal is constant, dP projected on plane)
	var dtdx, dtdy float64
	if r.HasDiff && math.Abs(r.Dir.Y) > 1e-12 {
		// Differentiate y(t)=h: dOy + t*dDy + Dy*dt = 0
		dtdx = -(r.DOriginX.Y + t*r.DDirX.Y) / r.Dir.Y
		dtdy = -(r.DOriginY.Y + t*r.DDirY.Y) / r.Dir.Y
	}
	dpx, dpy := TransferDifferentials(r, t, dtdx, dtdy)
	// Flat shading normal: no normal differential.
	return t, p, n, dpx, dpy, Vec3{}, Vec3{}, true
}
