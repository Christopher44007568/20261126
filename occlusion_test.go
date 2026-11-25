package main

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// A clear glass volume must never hard-occlude: the query stays non-opaque
// and transmittance is identity.
func TestClearGlassCastsNoSolidShadow(t *testing.T) {
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Slab{Min: Vec3{-1, -1, -1}, Max: Vec3{1, 1, 1}, G: &Glass{ID: 1, IOR: 1.5}},
		},
	}
	st := &Stats{}
	occ := scene.QueryOcclusion(Ray{Vec3{0, 5, 0}, Vec3{0, -1, 0}}, st)
	if occ.Opaque {
		t.Fatal("clear glass reported as opaque occlusion")
	}
	if occ.Transmitted != (Vec3{1, 1, 1}) {
		t.Fatalf("clear glass altered transmittance: %v", occ.Transmitted)
	}
	if st.OpaqueOccluded != 0 || st.PenetratingRays != 1 || st.GlassPenetrated != 1 {
		t.Fatalf("unexpected stats: %+v", st)
	}
}

// An opaque sphere behind the glass still hard-stops the ray immediately.
func TestOpaqueBehindGlassTruncates(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5, Absorption: Vec3{1, 1, 1}}
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Slab{Min: Vec3{-2, 2, -2}, Max: Vec3{2, 3, 2}, G: glass},
			&Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Albedo: Vec3{0.5, 0.5, 0.5}},
		},
	}
	st := &Stats{}
	occ := scene.QueryOcclusion(Ray{Vec3{0, 6, 0}, Vec3{0, -1, 0}}, st)
	if !occ.Opaque {
		t.Fatal("opaque sphere did not occlude")
	}
	if st.OpaqueOccluded != 1 || st.PenetratingRays != 0 {
		t.Fatalf("occluded ray must not count as penetrating: %+v", st)
	}
}

// Direct opaque occlusion with no glass in the path is a hard shadow and the
// penetrating counters stay zero.
func TestDirectHardShadow(t *testing.T) {
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Albedo: Vec3{0.5, 0.5, 0.5}},
		},
	}
	st := &Stats{}
	occ := scene.QueryOcclusion(Ray{Vec3{0, 5, 0}, Vec3{0, -1, 0}}, st)
	if !occ.Opaque || st.ClearRays != 0 || st.GlassPenetrated != 0 {
		t.Fatalf("hard shadow handling wrong: occ=%+v stats=%+v", occ, st)
	}
}

// Beer-Lambert attenuation over a known glass path length matches exp(-a*d).
func TestBeerLambertPath(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5, Absorption: Vec3{2, 1, 0}}
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Slab{Min: Vec3{-1, -1, -1}, Max: Vec3{1, 1, 1}, G: glass},
		},
	}
	st := &Stats{}
	// Ray travels downward through two units of glass (y from 1 to -1).
	occ := scene.QueryOcclusion(Ray{Vec3{0, 5, 0}, Vec3{0, -1, 0}}, st)
	want := Vec3{math.Exp(-4), math.Exp(-2), 1}
	if !approx(occ.Transmitted.X, want.X) || !approx(occ.Transmitted.Y, want.Y) ||
		!approx(occ.Transmitted.Z, want.Z) {
		t.Fatalf("transmittance %v, want %v", occ.Transmitted, want)
	}
	if occ.Crossings != 2 {
		t.Fatalf("crossings=%d, want 2", occ.Crossings)
	}
}

// Strong absorption over a thick path restores near-opaque visibility.
func TestNearOpaqueRestoresOcclusion(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5, Absorption: Vec3{6, 6, 6}}
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Slab{Min: Vec3{-1, -1, -1}, Max: Vec3{1, 1, 1}, G: glass},
		},
	}
	occ := scene.QueryOcclusion(Ray{Vec3{0, 5, 0}, Vec3{0, -1, 0}}, &Stats{})
	if Luminance(occ.Transmitted) > 0.02 {
		t.Fatalf("expected near-opaque transmittance, got %v", occ.Transmitted)
	}
}

// Every query is accounted for exactly once: blocked plus penetrating totals
// the number of rays issued.
func TestStatsInvariant(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5, Absorption: Vec3{1, 1, 1}}
	scene := &Scene{
		LightDir: Vec3{0, -1, 0},
		Primitives: []Primitive{
			&Slab{Min: Vec3{-0.5, 2, -0.5}, Max: Vec3{0.5, 3, 0.5}, G: glass},
			&Sphere{Center: Vec3{2, 0, 0}, Radius: 1, Albedo: Vec3{0.5, 0.5, 0.5}},
		},
	}
	st := &Stats{}
	for x := -4; x <= 4; x++ {
		scene.QueryOcclusion(Ray{Vec3{float64(x), 6, 0}, Vec3{0, -1, 0}}, st)
	}
	if st.ShadowRays != st.OpaqueOccluded+st.PenetratingRays {
		t.Fatalf("invariant broken: %+v", st)
	}
	if st.PenetratingRays != st.ClearRays+st.GlassPenetrated {
		t.Fatalf("penetrating breakdown wrong: %+v", st)
	}
}
