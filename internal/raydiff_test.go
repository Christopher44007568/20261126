package internal

import (
	"math"
	"testing"
)

func approxEq(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s = %.6f, want %.6f", name, got, want)
	}
}

func TestPrimaryDifferentials(t *testing.T) {
	cam := NewCamera(V(0, 0, 0), V(0, 0, -1), 90, 100, 100)
	r := cam.GenerateRay(50, 50)
	if !r.HasDiff {
		t.Fatal("primary ray must carry differentials")
	}
	// Pinhole: shared origin => zero origin derivatives.
	if r.DOriginX.Length2()+r.DOriginY.Length2() != 0 {
		t.Fatalf("pinhole origin derivatives must be zero")
	}
	// fov 90 => tan(45)=1, dD/dx magnitude = 2/width along right.
	approxEq(t, "ddirx", r.DDirX.Length(), 2.0/100, 1e-12)
	approxEq(t, "ddiry", r.DDirY.Length(), 2.0/100, 1e-12)
	// Center ray looks straight down -Z.
	approxEq(t, "dirz", r.Dir.Z, -1, 1e-3)
}

func TestSphereFootprintGrowsWithDistance(t *testing.T) {
	cam := NewCamera(V(0, 0, 0), V(0, 0, -1), 60, 256, 256)
	r := cam.GenerateRay(128, 128)

	near := Sphere{Center: V(0, 0, -4), Radius: 1}
	far := Sphere{Center: V(0, 0, -16), Radius: 1}

	_, _, _, dpxNear, _, _, _, okNear := near.Intersect(r)
	_, _, _, dpxFar, _, _, _, okFar := far.Intersect(r)
	if !okNear || !okFar {
		t.Fatal("expected both spheres to be hit")
	}
	if dpxFar.Length() <= dpxNear.Length() {
		t.Fatalf("far footprint %.4f must exceed near footprint %.4f",
			dpxFar.Length(), dpxNear.Length())
	}
}

func TestReflectionDifferentialMirrorConservesSpread(t *testing.T) {
	// Horizontal ray hitting a flat-ish top of a sphere: mirror reflection of a
	// tight differential must stay tight when roughness is zero.
	r := Ray{
		Origin:  V(0, 1, 0),
		Dir:     V(0, 0, -1),
		DDirX:   V(0.01, 0, 0),
		DDirY:   V(0, 0.01, 0),
		HasDiff: true,
	}
	n := V(0, 1, 0)
	p := V(0, 1, -5)
	out := ReflectedRay(r, p, n, V(0.1, 0, 0), V(0, 0.1, 0), Vec3{}, Vec3{}, 0)
	// Roughness 0 => spread factor 1, tangent magnitude preserved.
	approxEq(t, "mirror ddirx", out.DDirX.Length(), r.DDirX.Length(), 1e-12)
}

func TestRoughnessWidensFootprint(t *testing.T) {
	base := Ray{
		Origin: V(0, 1, 0), Dir: V(0, 0, -1),
		DDirX: V(0.01, 0, 0), DDirY: V(0, 0.01, 0), HasDiff: true,
	}
	n := V(0, 1, 0)
	p := V(0, 1, -5)
	glossy := ReflectedRay(base, p, n, Vec3{}, Vec3{}, Vec3{}, Vec3{}, 0.5)
	mirror := ReflectedRay(base, p, n, Vec3{}, Vec3{}, Vec3{}, Vec3{}, 0)
	if glossy.DDirX.Length() <= mirror.DDirX.Length() {
		t.Fatalf("rough reflection footprint should be wider")
	}
	// Mapping is continuous at roughness 0.
	approxEq(t, "spread at r=0", SpreadsRoughness(0), 1, 1e-12)
}

func TestLODContinuityAndClamp(t *testing.T) {
	tex := NewCheckerTexture()
	// Sweep a smoothly growing footprint: LOD must be monotonic and continuous.
	prev := -1.0
	for k := 0; k < 60; k++ {
		du := math.Pow(2, float64(k)/10.0) / float64(tex.Size)
		_, lod := tex.FootprintSample(0.3, 0.3, du, 0, 0, du)
		if lod < prev-1e-9 {
			t.Fatalf("LOD decreased: %.4f -> %.4f at k=%d", prev, lod, k)
		}
		if lod < 0 || lod > tex.MaxLOD() {
			t.Fatalf("LOD %.4f outside [0,%.0f]", lod, tex.MaxLOD())
		}
		prev = lod
	}
}

func TestEdgeDivergenceDoesNotEscapeLOD(t *testing.T) {
	tex := NewCheckerTexture()
	// Pathological divergent derivatives (NaN/Inf/huge) like a grazing edge ray.
	_, lod := tex.FootprintSample(0.5, 0.5,
		math.NaN(), math.Inf(1), 1e18, -1e18)
	approxEq(t, "clamped lod", lod, tex.MaxLOD(), 1e-9)
}

func TestSecondaryLODCoarserThanPrimaryForRoughMetal(t *testing.T) {
	// End-to-end: on rough metal, the reflected ray's angular footprint must be
	// wider than a zero-roughness reflection from the same shading point. That
	// widened footprint is what pushes distant reflection hits to coarser LOD.
	renderer := NewRenderer(128, 96, false)
	checked := 0
	for py := 0; py < 96; py++ {
		for px := 0; px < 128; px++ {
			primary := renderer.Camera.GenerateRay(px, py)
			hit, ok := renderer.trace(primary)
			if !ok || !hit.mat.Metal || hit.mat.Roughness < 0.4 {
				continue
			}
			rough := ReflectedRay(primary, hit.p, hit.n,
				hit.dpx, hit.dpy, hit.dnx, hit.dny, hit.mat.Roughness)
			mirror := ReflectedRay(primary, hit.p, hit.n,
				hit.dpx, hit.dpy, hit.dnx, hit.dny, 0)
			if rough.DDirX.Length() <= mirror.DDirX.Length() {
				t.Fatalf("rough reflection not widened at (%d,%d)", px, py)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no rough metal pixels found in scene")
	}
}
