package hotreload

import "testing"

func TestRGBReloadOnlyTouchesRGBSubtree(t *testing.T) {
	e := NewEngine()
	_, _, _, report := e.ReloadRGB(ReloadedRGBPayload())

	if !report.OK {
		t.Fatalf("expected successful reload, got errors %+v", report.Errors)
	}
	wantRecomputed := []string{"rgb.layer", "rgb.grade", "composite"}
	assertStrings(t, report.RecomputedNodes, wantRecomputed)
	wantRetained := []string{"background", "geo.layer", "surface", "outline"}
	assertStrings(t, report.CacheRetained, wantRetained)
	assertStrings(t, report.CacheHits, wantRetained)
	if len(report.ChangedEdges) != 3 {
		t.Fatalf("want 3 changed edges, got %d: %+v", len(report.ChangedEdges), report.ChangedEdges)
	}
	if report.Diff.ChangedPixels == 0 {
		t.Fatal("expected visible RGB pixel changes")
	}
}

func TestRGBErrorIsNotMaskedByCacheHit(t *testing.T) {
	e := NewEngine()
	goodKey := e.node("rgb.layer").cache.Key
	_, after, _, report := e.ReloadRGB(BrokenRGBPayload())

	if report.OK {
		t.Fatal("broken payload must fail")
	}
	if len(report.Errors) != 1 || report.Errors[0].Code != "invalid-rgb-buffer" {
		t.Fatalf("unexpected errors: %+v", report.Errors)
	}
	if e.node("rgb.layer").cache.Key == goodKey {
		t.Fatal("error frame must not retain the successful cache key")
	}
	if !containsColor(after, LeftPanel, ColorErrorBg) {
		t.Fatal("expected an error card in the RGB viewport")
	}
	for _, id := range report.CacheHits {
		if id == "rgb.layer" || id == "rgb.grade" || id == "composite" {
			t.Fatalf("failing subtree reported as cache hit: %s", id)
		}
	}
}

func containsColor(f *Frame, r Rect, want RGB) bool {
	for y := r.Y0; y < r.Y1; y++ {
		for x := r.X0; x < r.X1; x++ {
			if f.Get(x, y) == want {
				return true
			}
		}
	}
	return false
}

func TestGeometryReplaceRefreshesOutlineAndDetachesMaterial(t *testing.T) {
	e := NewEngine()
	_, _, _, report := e.ReplaceGeometry(ReplacementGeometry(), false)

	if !report.OK {
		t.Fatalf("geometry replacement failed: %+v", report.Errors)
	}
	wantRecomputed := []string{"geo.layer", "surface", "outline", "composite"}
	assertStrings(t, report.RecomputedNodes, wantRecomputed)
	wantRetained := []string{"background", "rgb.layer", "rgb.grade"}
	assertStrings(t, report.CacheRetained, wantRetained)
	assertStrings(t, report.DetachedMaterials, []string{string(MatStripes)})
	if len(report.BoundMaterials) != 0 {
		t.Fatalf("detached scenario must not bind fallback, got %v", report.BoundMaterials)
	}
	surface := e.node("surface")
	if surface.material != nil {
		t.Fatal("old material must be unbound from replacement geometry")
	}
	if surface.cache.Frame.Get(356, 80) == (RGB{}) {
		t.Fatal("surface should still render a visible fallback")
	}
}

func TestGeometryReplaceBindsCompatibleMaterial(t *testing.T) {
	e := NewEngine()
	_, _, _, report := e.ReplaceGeometry(ReplacementGeometry(), true)

	assertStrings(t, report.DetachedMaterials, []string{string(MatStripes)})
	assertStrings(t, report.BoundMaterials, []string{string(MatCheckAmber)})
	if e.node("surface").material == nil ||
		e.node("surface").material.ID != MatCheckAmber {
		t.Fatal("compatible fallback material was not bound")
	}
	if len(report.ChangedEdges) != 7 {
		t.Fatalf("want 7 changed edges including material detach, got %d", len(report.ChangedEdges))
	}
}

func TestRepeatedReloadStableCaches(t *testing.T) {
	e := NewEngine()
	firstPayload := ReloadedRGBPayload()
	e.ReloadRGB(firstPayload)
	keys := map[string]uint64{}
	for _, n := range e.Nodes() {
		keys[n.ID] = n.cache.Key
	}

	// Re-loading identical content changes revision metadata but produces the
	// same content key; every renderable node then retains its cache.
	_, _, _, report := e.ReloadRGB(append([]byte(nil), firstPayload...))
	if !report.OK {
		t.Fatalf("identical payload should validate: %+v", report.Errors)
	}
	for _, n := range e.Nodes() {
		if n.cache.Key != keys[n.ID] {
			t.Fatalf("node %s key changed for identical content", n.ID)
		}
	}
	if len(report.RecomputedNodes) != 0 {
		t.Fatalf("identical reload must be all cache hits, recomputed %v", report.RecomputedNodes)
	}
	if len(report.CacheRetained) != len(e.Nodes()) {
		t.Fatalf("want %d retained, got %v", len(e.Nodes()), report.CacheRetained)
	}
}

func TestDiffThresholdAndBounds(t *testing.T) {
	a := NewFrame(4, 2)
	b := NewFrame(4, 2)
	a.Fill(RGB{0, 0, 0})
	b.Fill(RGB{0, 0, 0})
	b.Set(3, 1, RGB{30, 0, 0})
	_, stats := ComputeDiff(a, b)
	if stats.ChangedPixels != 1 || !stats.Bounds.Valid ||
		stats.Bounds.MinX != 3 || stats.Bounds.MaxY != 1 {
		t.Fatalf("unexpected diff stats: %+v", stats)
	}
	if stats.Threshold != DiffThreshold {
		t.Fatalf("threshold = %d, want %d", stats.Threshold, DiffThreshold)
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("at %d got %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}
