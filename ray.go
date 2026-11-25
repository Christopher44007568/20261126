package rayao

import "math"

// Ray is a parametric line segment Origin + Direction*t, MinT <= t <= MaxT.
//
// Direction is normally normalized so that t is world-space distance, but the
// intersection code only assumes that it is finite and non-zero. MinT may be
// exactly zero: callers that spawn secondary rays at a hit point should either
// set a small positive bias or exclude the primitive that generated the point.
type Ray struct {
	Origin    Vec3
	Direction Vec3
	MinT      float64
	MaxT      float64
}

// PointAt returns Origin + Direction*t.
func (r Ray) PointAt(t float64) Vec3 {
	return r.Origin.Add(r.Direction.Mul(t))
}

// Finite reports whether all ray parameters can be used by traversal.
func (r Ray) Finite() bool {
	finite := func(v float64) bool {
		return !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	if !finite(r.Origin.X) || !finite(r.Origin.Y) || !finite(r.Origin.Z) {
		return false
	}
	if !finite(r.Direction.X) || !finite(r.Direction.Y) || !finite(r.Direction.Z) {
		return false
	}
	if r.Direction.LenSqr() == 0 || !finite(r.MinT) {
		return false
	}
	if math.IsNaN(r.MaxT) || r.MaxT < r.MinT || math.IsInf(r.MaxT, -1) {
		return false
	}
	return true
}
