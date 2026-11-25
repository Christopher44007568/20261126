package rayao

import (
	"math"
	"testing"
)

func testMaterial() Material {
	return Material{Name: "matte", Color: Vec3{0.8, 0.8, 0.8}}
}

func TestZeroDistanceStartIsAccepted(t *testing.T) {
	tri := NewTriangle(Vec3{X: 0, Y: 0, Z: 0}, Vec3{X: 1, Z: 0}, Vec3{Y: 1}, 0)
	scene := NewScene("zero", []Triangle{tri}, []Material{testMaterial()})

	ray := Ray{
		Origin:    Vec3{},
		Direction: Vec3{Z: 1},
		MinT:      0,
		MaxT:      math.Inf(1),
	}
	result := scene.Query(DefaultRayQuery(ray))
	if result.SceneVersion != scene.Version {
		t.Fatalf("result version %q != scene %q", result.SceneVersion, scene.Version)
	}
	if !result.HasClosest || !result.HasAny {
		t.Fatalf("zero-distance start missed triangle: %+v", result)
	}
	if result.Closest.Primitive != 0 || result.Closest.T != 0 {
		t.Fatalf("zero hit = primitive %d t %.17g, want 0 at t=0", result.Closest.Primitive, result.Closest.T)
	}
	if result.FirstAny.T != 0 {
		t.Fatalf("first any t = %.17g, want zero", result.FirstAny.T)
	}
}

func TestOriginInsideTriangleAtZeroDistance(t *testing.T) {
	tri := NewTriangle(Vec3{}, Vec3{X: 1}, Vec3{Y: 1}, 0)
	scene := NewScene("inside-triangle", []Triangle{tri}, []Material{testMaterial()})
	ray := Ray{
		Origin:    Vec3{X: 0.25, Y: 0.25},
		Direction: Vec3{Z: 1},
		MinT:      0,
		MaxT:      math.Inf(1),
	}

	hit, ok := scene.ClosestHit(DefaultRayQuery(ray))
	if !ok {
		t.Fatal("ray starting in triangle interior missed zero-distance hit")
	}
	if hit.T != 0 || math.Abs(hit.U-0.25) > 1e-9 || math.Abs(hit.V-0.25) > 1e-9 {
		t.Fatalf("hit = t %.17g u %.6g v %.6g, want t=0 u=v=0.25", hit.T, hit.U, hit.V)
	}
}

func TestOriginInsideClosedSurfaceHitsExit(t *testing.T) {
	cube := NewCube(Vec3{}, 2, 0)
	scene := NewScene("inside", cube, []Material{testMaterial()})
	ray := Ray{
		Origin:    Vec3{},
		Direction: Vec3{X: 1},
		MinT:      0,
		MaxT:      math.Inf(1),
	}

	hit, ok := scene.ClosestHit(DefaultRayQuery(ray))
	if !ok {
		t.Fatal("ray from cube center did not hit an exit face")
	}
	if math.Abs(hit.T-1) > 1e-9 {
		t.Fatalf("exit distance = %.12g, want 1", hit.T)
	}
	if math.Abs(hit.Point.X-1) > 1e-9 {
		t.Fatalf("exit point x = %.12g, want 1", hit.Point.X)
	}
	if hit.FrontFacing {
		t.Fatal("inside origin should see the +X face as back-facing")
	}
	if hit.Normal.X >= 0 {
		t.Fatalf("facing normal = %+v, want negative X toward interior ray", hit.Normal)
	}

	if _, any := scene.AnyHit(DefaultRayQuery(ray)); !any {
		t.Fatal("AnyHit missed exit face from interior origin")
	}
}

func TestDuplicateTriangleCandidateCountsOnce(t *testing.T) {
	tri := NewTriangle(Vec3{}, Vec3{X: 1}, Vec3{Y: 1}, 0)
	scene := NewScene("duplicate-candidate", []Triangle{tri}, []Material{testMaterial()})
	ray := Ray{Origin: Vec3{}, Direction: Vec3{Z: 1}, MinT: 0, MaxT: math.Inf(1)}

	result := scene.QueryCandidates(DefaultRayQuery(ray), []int{0, 0})
	if result.DuplicateCandidates != 1 {
		t.Fatalf("duplicates = %d, want 1", result.DuplicateCandidates)
	}
	if result.Checked != 1 || result.Accepted != 1 {
		t.Fatalf("checked=%d accepted=%d, want each exactly once", result.Checked, result.Accepted)
	}
	if !result.HasClosest || !result.HasAny || result.Closest.Primitive != result.FirstAny.Primitive {
		t.Fatalf("closest/any disagree: %+v", result)
	}
}

func TestPrimaryAndSecondaryCanHitSameTriangle(t *testing.T) {
	tri := NewTriangle(Vec3{}, Vec3{X: 1}, Vec3{Y: 1}, 0)
	scene := NewScene("same-triangle", []Triangle{tri}, []Material{testMaterial()})

	primary := Ray{
		Origin:    Vec3{X: 0.2, Y: 0.2, Z: 1},
		Direction: Vec3{Z: -1},
		MinT:      0,
		MaxT:      math.Inf(1),
	}
	primaryHit, ok := scene.ClosestHit(DefaultRayQuery(primary))
	if !ok {
		t.Fatal("primary ray missed triangle")
	}
	if math.Abs(primaryHit.T-1) > 1e-9 {
		t.Fatalf("primary t = %.12g, want 1", primaryHit.T)
	}

	// A raw ray launched at the reconstructed hit point can report the same
	// primitive at t=0. This is the self-intersection case bias/masking solves.
	selfRay := Ray{
		Origin:    primaryHit.Point,
		Direction: Vec3{Z: 1},
		MinT:      0,
		MaxT:      math.Inf(1),
	}
	raw, ok := scene.AnyHit(DefaultRayQuery(selfRay))
	if !ok {
		t.Fatal("zero-distance secondary query should see same triangle")
	}
	if raw.Primitive != primaryHit.Primitive || raw.T != 0 {
		t.Fatalf("secondary hit = prim %d t %.17g; want prim %d t=0",
			raw.Primitive, raw.T, primaryHit.Primitive)
	}

	maskedQuery := RayQuery{Ray: selfRay, IgnorePrimitive: primaryHit.Primitive}
	if witness, ok := scene.AnyHit(maskedQuery); ok {
		t.Fatalf("masked self-hit unexpectedly returned primitive %d", witness.Primitive)
	}
	maskedResult := scene.Query(maskedQuery)
	if maskedResult.IgnoredPrimitive != 1 || maskedResult.HasAny {
		t.Fatalf("masked query result = %+v", maskedResult)
	}
}
