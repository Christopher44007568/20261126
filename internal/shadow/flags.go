package shadow

// RayFlag selects the shadow-ray intersection policy.
type RayFlag int

const (
	// BackfaceCull keeps only front-face hits, as a conventional shadow ray
	// with back-face culling enabled would.
	BackfaceCull RayFlag = iota
	// DoubleSidedHit accepts back faces on materials marked DoubleSided;
	// single-sided materials still cull their back faces.
	DoubleSidedHit
	// ReflectionClosure mirrors the double-sided result but keeps material
	// closure state, so every closed solid contributes energy exactly once.
	ReflectionClosure
)

// ColorTag marks why a hit is or is not counted, for the overlay images.
type ColorTag int

const (
	TagNone ColorTag = iota
	// TagExtraOccluder: an accepted hit that BackfaceCull would have rejected.
	TagExtraOccluder
	// TagIgnoredBackface: a geometric hit on the back of a single-sided sheet.
	TagIgnoredBackface
	// TagClosureExit: the exit hit of a closed solid, geometrically real but
	// merged into the entry event so energy is not counted a second time.
	TagClosureExit
)

// Eval is the per-pixel outcome of a shadow ray under one RayFlag.
type Eval struct {
	Energy        float64
	AcceptedHits  int
	GeometricHits int
	Tag           ColorTag
	LastMaterial  string
}

// acceptHit applies the ray-flag policy to a single raw intersection.
func acceptHit(flag RayFlag, hit Hit) bool {
	if hit.Front {
		return true
	}
	switch flag {
	case BackfaceCull:
		return false
	case DoubleSidedHit, ReflectionClosure:
		return hit.Prim.Material.DoubleSided
	default:
		return false
	}
}

// Evaluate traces one shadow ray through the scene under the given flag.
func (s *Scene) Evaluate(ray Ray, flag RayFlag) Eval {
	raw := s.AllHits(ray)
	result := Eval{Energy: 1, GeometricHits: len(raw)}

	// entered tracks closed solids whose entry energy event already happened.
	entered := map[*Material]bool{}
	for _, hit := range raw {
		mat := hit.Prim.Material
		accepted := acceptHit(flag, hit)

		if !accepted {
			// Requirement: the back of a single-sided sheet never casts a
			// solid shadow. Under the double-sided flags such a hit is still
			// geometrically present, so it gets an explicit diagnostic color.
			if !mat.DoubleSided && !hit.Front && flag != BackfaceCull {
				result.Tag = TagIgnoredBackface
			}
			continue
		}

		// Extra relative to a plain back-face-culling shadow ray.
		if !hit.Front && flag != BackfaceCull {
			result.Tag = TagExtraOccluder
		}

		// Closed solid: merge entry and exit into one per-solid energy event.
		if mat.Closure {
			if entered[mat] {
				// The exit triangle is geometrically real, but its energy was
				// already applied at entry. The reflection closure policy
				// surfaces the merge explicitly; plain double-sided mode keeps
				// the extra-hit classification while still counting once.
				if flag == ReflectionClosure {
					result.Tag = TagClosureExit
				}
				continue
			}
			entered[mat] = true
		}

		result.AcceptedHits++
		result.LastMaterial = mat.Name
		result.Energy *= mat.Transmittance
		if result.Energy <= rayEpsilon {
			break
		}
	}
	return result
}

// NaiveDoubleCount re-applies transmittance for every triangle of the closed
// solid instead of once per solid. It demonstrates the double-counted energy
// that enabling double-sided hits naively would produce.
func (s *Scene) NaiveDoubleCount(ray Ray) float64 {
	energy := 1.0
	for _, hit := range s.AllHits(ray) {
		if hit.Prim.Material.Closure && hit.Prim.Material.DoubleSided {
			energy *= hit.Prim.Material.Transmittance
		}
	}
	return energy
}
