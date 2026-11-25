package main

// Stats accumulates the any-hit query counters requested for every frame.
type Stats struct {
	ShadowRays         int // visibility queries issued
	OpaqueOccluded     int // rays terminated by an opaque surface
	GlassPenetrated    int // rays that crossed at least one glass boundary
	PenetratingRays    int // rays still traveling after glass: glass + clear
	ClearRays          int // rays reaching the light with no interaction
	GlassCrossings     int // individual glass boundary crossings
	CameraRays         int // primary rays cast
	CameraThroughGlass int // primary rays whose first hit was a glass volume
}

type Occlusion struct {
	Opaque      bool
	Transmitted Vec3 // spectral transmittance over the glass segments
	Penetrated  bool // crossed glass and was not opaque-blocked
	Crossings   int
}

// QueryOcclusion walks every hit in front-to-back order (the any-hit
// approach). Opaque geometry hard-stops the ray. Glass entry/exit events are
// matched by glass identity so the optical path length of each traversed
// segment contributes Beer-Lambert attenuation; overlapping/nested volumes
// are handled with an entry stack.
func (s *Scene) QueryOcclusion(r Ray, st *Stats) Occlusion {
	res := Occlusion{Transmitted: Vec3{1, 1, 1}}
	hits := s.AllHits(r)

	type entry struct {
		glass *Glass
		hit   Hit
	}
	var stack []entry

	st.ShadowRays++

	for _, h := range hits {
		if h.Glass == nil {
			// Hard shadow: the ray is truncated exactly like a binary query.
			st.OpaqueOccluded++
			res.Opaque = true
			return res
		}

		st.GlassCrossings++
		res.Crossings++
		if h.Entering {
			stack = append(stack, entry{h.Glass, h})
			continue
		}

		// Exiting: match the innermost entry of the same glass volume.
		idx := -1
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].glass == h.Glass {
				idx = i
				break
			}
		}
		if idx < 0 {
			// Unpaired exit: the ray started inside this volume, so the
			// segment from the origin to the exit still carries absorption.
			res.Transmitted = res.Transmitted.Comp(BeerLambert(h.Glass.Absorption, h.T))
			res.Penetrated = true
			continue
		}
		in := stack[idx]
		stack = append(stack[:idx], stack[idx+1:]...)
		path := h.T - in.hit.T
		res.Transmitted = res.Transmitted.Comp(BeerLambert(in.glass.Absorption, path))
		res.Penetrated = true
	}

	if res.Penetrated {
		st.GlassPenetrated++
		st.PenetratingRays++
	} else {
		st.ClearRays++
		// A clear ray is also a penetrating ray: it reaches the light
		// without any truncation from hard shadow.
		st.PenetratingRays++
	}
	return res
}
