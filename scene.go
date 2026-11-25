package main

import "math"

// Mask is a per-instance visibility bitmask. Each ray type consults
// exactly one bit, so clearing a bit never affects other channels.
type Mask uint8

const (
	MaskVisible    Mask = 1 << iota // primary camera rays
	MaskShadow                      // shadow (occlusion) rays
	MaskReflection                  // reflection rays
	MaskAll        = MaskVisible | MaskShadow | MaskReflection
)

type RayType int

const (
	RayPrimary RayType = iota
	RayShadow
	RayReflection
)

func maskFor(rt RayType) Mask {
	switch rt {
	case RayShadow:
		return MaskShadow
	case RayReflection:
		return MaskReflection
	default:
		return MaskVisible
	}
}

// Instance is a sphere with a per-instance mask.
type Instance struct {
	ID     int
	Center Vec3
	Radius float64
	Color  Vec3
	Mask   Mask
}

type Hit struct {
	Inst *Instance
	T    float64
	P    Vec3
	N    Vec3
}

// Excluded records an instance the ray geometrically crossed but that
// was filtered out by its mask for this ray type.
type Excluded struct {
	Inst *Instance
	T    float64
}

type Scene struct {
	Instances []*Instance
	Light     Vec3
	Ambient   float64
}

func (s *Scene) Add(inst *Instance) {
	inst.ID = len(s.Instances)
	s.Instances = append(s.Instances, inst)
}

func sphereT(inst *Instance, r Ray) (float64, bool) {
	oc := r.Orig.Sub(inst.Center)
	b := oc.Dot(r.Dir)
	c := oc.Dot(oc) - inst.Radius*inst.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}
	sq := math.Sqrt(disc)
	const eps = 1e-4
	if t := -b - sq; t > eps {
		return t, true
	}
	if t := -b + sq; t > eps {
		return t, true
	}
	return 0, false
}

// Intersect finds the closest instance whose mask admits rt. Instances
// crossed by the ray but filtered out are reported in excluded; they
// never block or replace an admitted hit.
func (s *Scene) Intersect(r Ray, rt RayType) (Hit, []Excluded, bool) {
	need := maskFor(rt)
	var best Hit
	var excluded []Excluded
	ok := false
	bestT := math.Inf(1)
	for _, inst := range s.Instances {
		t, hit := sphereT(inst, r)
		if !hit {
			continue
		}
		if inst.Mask&need == 0 {
			excluded = append(excluded, Excluded{Inst: inst, T: t})
			continue
		}
		if t < bestT {
			bestT = t
			p := r.At(t)
			best = Hit{Inst: inst, T: t, P: p, N: p.Sub(inst.Center).Norm()}
			ok = true
		}
	}
	return best, excluded, ok
}
