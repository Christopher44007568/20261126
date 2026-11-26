package viz_test

import (
	"testing"

	"nsmigrate/audit"
	"nsmigrate/migrate"
	"nsmigrate/raster"
	"nsmigrate/scene"
	"nsmigrate/viz"
)

func renderCorrect(t *testing.T) *viz.Poster {
	t.Helper()
	s := scene.BuildDemoStage()
	m := scene.FixtureMapping()
	next, _ := migrate.MigrateCorrect(s, m)
	rep := audit.Audit("correct", next, m)
	return viz.Render(viz.Options{
		ImageName: "migration_after.ppm", Mode: "correct",
		Title: "t", Subtitle: "s", Stage: next, Report: rep, Mapping: m,
	})
}

func TestPosterRecordsBothRedBoxCategories(t *testing.T) {
	p := renderCorrect(t)
	var hasBroken, hasDefault bool
	if len(p.Boxes) == 0 {
		t.Fatal("expected red boxes")
	}
	for _, b := range p.Boxes {
		switch b.Category {
		case "broken_link":
			hasBroken = true
		case "retained_default":
			hasDefault = true
		default:
			t.Fatalf("unknown box category %q", b.Category)
		}
		if b.OldPath == "" {
			t.Errorf("box %s (%s/%s) must carry the concrete old path", b.ID, b.Category, b.Part)
		}
	}
	if !hasBroken || !hasDefault {
		t.Fatalf("need both categories, broken=%v default=%v", hasBroken, hasDefault)
	}
}

func TestRedBoxBordersArePureRedPixels(t *testing.T) {
	p := renderCorrect(t)
	red := raster.Color{R: 255, G: 0, B: 0}
	for _, b := range p.Boxes {
		// The 4px frame's top edge must be pure (255,0,0), proving the manifest
		// rectangle corresponds to actual red pixels in the PPM/PNG.
		if got := p.Canvas.At(b.X+4, b.Y+2); got != red {
			t.Fatalf("box %s top edge color=%+v want red", b.ID, got)
		}
		if got := p.Canvas.At(b.X+2, b.Y+4); got != red {
			t.Fatalf("box %s left edge color=%+v want red", b.ID, got)
		}
	}
}

func TestStringOnlyPosterHasMoreBoxesAndFails(t *testing.T) {
	s := scene.BuildDemoStage()
	m := scene.FixtureMapping()
	naive := migrate.MigrateStringOnly(s, m)
	rep := audit.Audit("string-only", naive, m)
	if rep.InvariantOK {
		t.Fatal("naive migration must fail invariant")
	}
	p := viz.Render(viz.Options{Mode: "string-only", Title: "t", Stage: naive, Report: rep, Mapping: m})
	if len(p.Boxes) < 12 {
		t.Fatalf("expected many red boxes under string-only edit, got %d", len(p.Boxes))
	}
}
