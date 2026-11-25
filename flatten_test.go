package main

import (
	"image/color"
	"testing"
)

// TestFlattenRoundTrip renders the demo scene directly and via a
// save/reload cycle of its flattened form; the images must be
// pixel-identical.
func TestFlattenRoundTrip(t *testing.T) {
	scene := buildDemoScene()
	orig := RenderScene(scene)
	flat := Flatten(scene)
	path := t.TempDir() + "/flat.json"
	if err := SaveFlatScene(flat, path); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadFlatScene(path)
	if err != nil {
		t.Fatal(err)
	}
	_, maxDelta, changed := DiffImages(orig, RenderFlat(reloaded))
	if maxDelta != 0 || changed != 0 {
		t.Fatalf("renders differ: maxDelta=%d changed=%d", maxDelta, changed)
	}
}

// TestHiddenPreserved checks that hidden leaves, groups and layers
// survive flattening with Hidden=true and are not rendered.
func TestHiddenPreserved(t *testing.T) {
	flat := Flatten(buildDemoScene())
	want := map[string]bool{
		"mid/secret-group/secret-a": true,
		"mid/secret-group/secret-b": true,
		"mid/ghost":                 true,
		"hidden-layer/never-seen":   true,
		"mid/wheel/rim":             false,
		"mid/badges/badge-a":        false,
	}
	got := map[string]bool{}
	for _, s := range flat.Shapes {
		got[s.Name] = s.Hidden
	}
	for name, hidden := range want {
		v, ok := got[name]
		if !ok {
			t.Errorf("shape %q lost during flattening", name)
		} else if v != hidden {
			t.Errorf("shape %q hidden=%v, want %v", name, v, hidden)
		}
	}
}

// TestOverrideBaked checks that material overrides are resolved into
// the flattened materials (gold fill from the group, white stroke from
// the deeper group) without clobbering untouched fields.
func TestOverrideBaked(t *testing.T) {
	flat := Flatten(buildDemoScene())
	byName := map[string]FlatShape{}
	for _, s := range flat.Shapes {
		byName[s.Name] = s
	}
	// The "mid" layer has Opacity 0.92, so baked alpha is 255*0.92 = 235.
	gold := color.NRGBA{R: 240, G: 190, B: 60, A: 235}
	a := byName["mid/badges/badge-a"]
	if a.Material.Fill != gold {
		t.Errorf("badge-a fill = %+v, want gold override", a.Material.Fill)
	}
	if a.Material.StrokeWidth != 8 {
		t.Errorf("badge-a stroke width = %v, want 8 (untouched by override)", a.Material.StrokeWidth)
	}
	b := byName["mid/badges/badge-b-group/badge-b"]
	white := color.NRGBA{R: 245, G: 245, B: 245, A: 235}
	if b.Material.Stroke != white {
		t.Errorf("badge-b stroke = %+v, want deep white override", b.Material.Stroke)
	}
	if b.Material.Fill != gold {
		t.Errorf("badge-b fill = %+v, want inherited gold override", b.Material.Fill)
	}
}

// TestTransformBaked checks that parent-child transform chains are
// composed into world-space geometry.
func TestTransformBaked(t *testing.T) {
	scene := &Scene{
		Width: 100, Height: 100,
		Layers: []*Layer{{Name: "l", Root: &Node{Children: []*Node{
			{
				Name:      "parent",
				Transform: Translate(10, 20),
				Children: []*Node{
					{
						Name:      "child",
						Kind:      KindRect,
						W:         4,
						H:         4,
						Transform: Translate(1, 2).Mul(Scale(2, 2)),
						Material:  Material{Fill: color.NRGBA{A: 255}, HasFill: true},
					},
				},
			},
		}}}},
	}
	flat := Flatten(scene)
	if len(flat.Shapes) != 1 {
		t.Fatalf("got %d shapes, want 1", len(flat.Shapes))
	}
	// Local corner (2,2) -> scaled (4,4) -> child translate (5,6)
	// -> parent translate (15,26).
	got := flat.Shapes[0].Points[2]
	if got != (Point{X: 15, Y: 26}) {
		t.Errorf("baked corner = %+v, want {15 26}", got)
	}
}
