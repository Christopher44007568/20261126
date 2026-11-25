package internal

import "math"

// ReflectedRay builds the mirror/rough-reflection ray and transfers the parent
// ray differentials through the reflection event.
//
// For R = reflect(D, n), differentiating gives (Igehy 1999):
//
//	dn = dD.Dot(n) + D.Dot(dN)
//	dR = dD - 2*(dn*n + (D.Dot n)*dN)
//
// A perfect mirror keeps the pixel footprint tight. A rough conductor scatters
// into a lobe whose angular width grows with roughness; widening the reflected
// ray's direction differentials by a matching factor makes distant reflection
// hits select coarser mip levels instead of reusing the primary footprint.
func ReflectedRay(r Ray, p, n, dpx, dpy, dnx, dny Vec3, roughness float64) Ray {
	dir := Reflect(r.Dir, n).Normalize()

	out := Ray{
		Origin:  p,
		Dir:     dir,
		HasDiff: r.HasDiff,
	}
	if !r.HasDiff {
		return out
	}

	// Origin differential is the surface footprint directly.
	out.DOriginX = dpx
	out.DOriginY = dpy

	drdx := reflectDDir(r.Dir, n, r.DDirX, dnx)
	drdy := reflectDDir(r.Dir, n, r.DDirY, dny)

	// Roughness broadening: alpha = roughness^2 is the GGX linear roughness.
	// The factor stays 1 at roughness 0 and grows smoothly, keeping LOD
	// continuous as roughness changes. A clamp guards against runaway footprints
	// (e.g. grazing edge rays) so the screen border cannot pick a bogus level.
	alpha := roughness * roughness
	spread := 1 + 64*alpha*alpha
	spread = Clamp(spread, 1, 1<<14)

	tan, bitan := TangentSpace(n)
	widen := func(dr Vec3) Vec3 {
		normalComp := dr.Dot(n) * 1
		tanComp := dr.Dot(tan) * spread
		bitComp := dr.Dot(bitan) * spread
		return n.Scale(normalComp).
			Add(tan.Scale(tanComp)).
			Add(bitan.Scale(bitComp))
	}
	out.DDirX = widen(drdx)
	out.DDirY = widen(drdy)
	return out
}

func reflectDDir(d, n, dd, dn Vec3) Vec3 {
	dnDot := dd.Dot(n) + d.Dot(dn)
	return dd.Sub(n.Scale(2 * dnDot)).
		Sub(dn.Scale(2 * d.Dot(n)))
}

// SpreadsRoughness is exposed for tests/documentation of the mapping.
func SpreadsRoughness(roughness float64) float64 {
	alpha := roughness * roughness
	return math.Min(1+64*alpha*alpha, 1<<14)
}
