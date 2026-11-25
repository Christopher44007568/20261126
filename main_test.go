package main

import "testing"

func imagesEqual(a, b *Image) bool {
	if a.W != b.W || a.H != b.H {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

// Clearing one mask bit of one instance must only change the matching
// channel; the other channels must stay pixel-identical.
func TestMaskChangeIsolatesChannels(t *testing.T) {
	const w, h = 160, 100
	cam := NewCamera(w, h)

	base := buildScene()
	chBase := Render(base, cam, w, h)

	// Remove the reflection bit from instance 0.
	mod := buildScene()
	mod.Instances[0].Mask &^= MaskReflection
	chMod := Render(mod, cam, w, h)

	if imagesEqual(chBase.Reflection, chMod.Reflection) {
		t.Fatal("reflection channel should change after clearing MaskReflection")
	}
	if !imagesEqual(chBase.Shadow, chMod.Shadow) {
		t.Fatal("shadow channel must be unaffected by MaskReflection change")
	}

	// Remove the shadow bit from instance 0 instead.
	mod2 := buildScene()
	mod2.Instances[0].Mask &^= MaskShadow
	chMod2 := Render(mod2, cam, w, h)

	if imagesEqual(chBase.Shadow, chMod2.Shadow) {
		t.Fatal("shadow channel should change after clearing MaskShadow")
	}
	if !imagesEqual(chBase.Reflection, chMod2.Reflection) {
		t.Fatal("reflection channel must be unaffected by MaskShadow change")
	}
}

// A camera-invisible instance must not appear in the primary channel
// but must still cast shadows and appear in reflections.
func TestPerRayTypeFiltering(t *testing.T) {
	s := &Scene{Light: Vec3{0, 5, 0}, Ambient: 0.2}
	hidden := &Instance{Center: Vec3{0, 0, -5}, Radius: 1, Color: Vec3{1, 0, 0}, Mask: MaskShadow | MaskReflection}
	s.Add(hidden)

	// Primary ray straight at it: no hit.
	if _, excl, ok := s.Intersect(Ray{Orig: Vec3{}, Dir: Vec3{0, 0, -1}}, RayPrimary); ok {
		t.Fatal("primary ray must miss MaskVisible-cleared instance")
	} else if len(excl) != 1 || excl[0].Inst != hidden {
		t.Fatalf("expected hidden instance reported as excluded, got %+v", excl)
	}
	// Shadow ray: hit.
	if _, _, ok := s.Intersect(Ray{Orig: Vec3{}, Dir: Vec3{0, 0, -1}}, RayShadow); !ok {
		t.Fatal("shadow ray must hit instance with MaskShadow set")
	}
	// Reflection ray: hit.
	if _, _, ok := s.Intersect(Ray{Orig: Vec3{}, Dir: Vec3{0, 0, -1}}, RayReflection); !ok {
		t.Fatal("reflection ray must hit instance with MaskReflection set")
	}
}

// An excluded instance must never occlude an admitted instance behind it.
func TestExcludedDoesNotOcclude(t *testing.T) {
	s := &Scene{}
	ghost := &Instance{Center: Vec3{0, 0, -3}, Radius: 1, Color: Vec3{1, 0, 0}, Mask: MaskVisible} // no shadow bit
	solid := &Instance{Center: Vec3{0, 0, -6}, Radius: 1, Color: Vec3{0, 1, 0}, Mask: MaskAll}
	s.Add(ghost)
	s.Add(solid)

	hit, excl, ok := s.Intersect(Ray{Orig: Vec3{}, Dir: Vec3{0, 0, -1}}, RayShadow)
	if !ok || hit.Inst != solid {
		t.Fatalf("shadow ray must pass through ghost and hit solid, got %+v", hit)
	}
	if len(excl) != 1 || excl[0].Inst != ghost {
		t.Fatalf("ghost should be excluded, got %+v", excl)
	}
}
