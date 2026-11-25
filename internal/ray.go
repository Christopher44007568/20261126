package internal

// Ray carries both the transported ray and its screen-space differential.
//
// dOdx/dOdy are the derivatives of the ray origin with respect to moving one
// pixel on the image plane in x/y. dDdx/dDdy are the same derivatives for the
// (unnormalized) direction. Direction derivatives stay unnormalized on purpose:
// normalization would lose the angular-spread information that drives LOD.
type Ray struct {
	Origin   Vec3
	Dir      Vec3
	DOriginX Vec3
	DOriginY Vec3
	DDirX    Vec3
	DDirY    Vec3
	HasDiff  bool
}

// TransferDifferentials propagates the parent ray's differential to the hit
// point P = O + t*D:
//
//	dP/dx = dO/dx + t*dD/dx + D*dt/dx
//
// dt/dx is solved from the surface constraint (sphere/plane) and supplied by
// the intersector through dtdx/dtdy.
func TransferDifferentials(r Ray, t float64, dtdx, dtdy float64) (dpx, dpy Vec3) {
	if !r.HasDiff {
		return Vec3{}, Vec3{}
	}
	dpx = r.DOriginX.Add(r.DDirX.Scale(t)).Add(r.Dir.Scale(dtdx))
	dpy = r.DOriginY.Add(r.DDirY.Scale(t)).Add(r.Dir.Scale(dtdy))
	return dpx, dpy
}
