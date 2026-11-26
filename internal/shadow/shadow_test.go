package shadow

import "testing"

func evalAt(scene *Scene, x, z float64, flag RayFlag) Eval {
	return scene.Evaluate(ShadowRay(Vec3{X: x, Y: ReceiverY, Z: z}), flag)
}

func evalUnder(scene *Scene, objectCenter Vec3, flag RayFlag) Eval {
	ground := PointUnderObject(objectCenter)
	return scene.Evaluate(ShadowRay(ground), flag)
}

// A single-sided sheet whose back faces the receiver must never cast a solid
// shadow under any ray flag.
func TestSingleSidedBackfaceNeverOccludes(t *testing.T) {
	scene := BuildScene()
	for _, flag := range []RayFlag{BackfaceCull, DoubleSidedHit, ReflectionClosure} {
		up := evalUnder(scene, Vec3{-4.5, sheetY, 2.5}, flag)
		if up.Energy != 1 {
			t.Fatalf("flag %v: back-facing single-sided sheet blocked energy, got %.2f", flag, up.Energy)
		}
		if up.AcceptedHits != 0 {
			t.Fatalf("flag %v: expected zero accepted hits, got %d", flag, up.AcceptedHits)
		}
		if flag != BackfaceCull && up.Tag != TagIgnoredBackface {
			t.Fatalf("flag %v: rejected back face must be tagged, got %v", flag, up.Tag)
		}

		down := evalUnder(scene, Vec3{-4.5, sheetY, -1.5}, flag)
		if down.Energy != 0 {
			t.Fatalf("flag %v: front-facing single-sided sheet should fully block, got %.2f", flag, down.Energy)
		}
	}
}

// Back-face culling rejects the back of a double-sided sheet; the other two
// flags accept it and therefore cast an extra, color-tagged shadow.
func TestDoubleSidedSheetBackFace(t *testing.T) {
	scene := BuildScene()

	culled := evalUnder(scene, Vec3{0, sheetY, -1.5}, BackfaceCull)
	if culled.Energy != 1 || culled.Tag != TagNone {
		t.Fatalf("backface cull: expected full energy, got %+v", culled)
	}

	for _, flag := range []RayFlag{DoubleSidedHit, ReflectionClosure} {
		hit := evalUnder(scene, Vec3{0, sheetY, -1.5}, flag)
		if hit.Energy != 0 || hit.AcceptedHits != 1 {
			t.Fatalf("flag %v: expected the back face to block, got %+v", flag, hit)
		}
		if hit.Tag != TagExtraOccluder {
			t.Fatalf("flag %v: expected extra-occluder tag, got %v", flag, hit.Tag)
		}
	}
}

// Enabling double-sided hits must not make a closed solid apply its
// transmittance twice: all three flags agree, naive counting does not.
func TestClosedSolidEnergyCountedOnce(t *testing.T) {
	scene := BuildScene()
	// Aim through the top face center so the ray clearly enters and exits.
	ray := ShadowRay(PointUnderObject(Vec3{4.5, cubeY + cubeHalf, 0.5}))

	var energies []float64
	for _, flag := range []RayFlag{BackfaceCull, DoubleSidedHit, ReflectionClosure} {
		eval := scene.Evaluate(ray, flag)
		if eval.AcceptedHits != 1 {
			t.Fatalf("flag %v: solid should produce one energy event, got %d", flag, eval.AcceptedHits)
		}
		energies = append(energies, eval.Energy)
	}
	if energies[0] != 0.6 || energies[1] != 0.6 || energies[2] != 0.6 {
		t.Fatalf("all flags must agree at 0.6, got %v", energies)
	}
	if naive := scene.NaiveDoubleCount(ray); naive != 0.36 {
		t.Fatalf("naive per-triangle counting should reach 0.36, got %.2f", naive)
	}

	closure := scene.Evaluate(ray, ReflectionClosure)
	if closure.Tag != TagClosureExit {
		t.Fatalf("exit hit must be tagged as closure exit, got %v", closure.Tag)
	}

	plain := scene.Evaluate(ray, DoubleSidedHit)
	if plain.Energy != closure.Energy {
		t.Fatalf("double-sided and closure energy must match, got %.2f vs %.2f",
			plain.Energy, closure.Energy)
	}
	if plain.Tag != TagExtraOccluder {
		t.Fatalf("plain double-sided mode keeps extra-hit tag, got %v", plain.Tag)
	}
}

func TestFrontBackClassification(t *testing.T) {
	normal := Vec3{0, -1, 0}
	frontRay := Ray{Direction: Vec3{0, 1, 0}}
	backRay := Ray{Direction: Vec3{0, -1, 0}}
	if frontRay.Direction.Dot(normal) >= 0 {
		t.Fatal("upward ray should be a front hit against a downward normal")
	}
	if backRay.Direction.Dot(normal) <= 0 {
		t.Fatal("downward ray should be a back hit against a downward normal")
	}
}
