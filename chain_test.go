package main

import (
	"strings"
	"testing"
)

func buildTestManifest(t *testing.T) *Manifest {
	t.Helper()
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	m, err := app.BuildManifest()
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetWork() })
	return m
}

func TestBeforeAllReferencesResolve(t *testing.T) {
	m := buildTestManifest(t)
	for _, sc := range []SceneReport{m.Before.Live, m.Before.Archive} {
		for _, s := range []SlotReport{sc.Material, sc.Geometry, sc.Camera} {
			if s.Error != "" {
				t.Fatalf("before slot %s unexpectedly broken: %s", s.Slot, s.Error)
			}
			if s.Kind != s.Slot {
				t.Fatalf("slot %s kind = %q, want %q", s.Slot, s.Kind, s.Slot)
			}
			if len(s.Chain) != 2 || !s.Chain[0].Exists || !s.Chain[1].Exists {
				t.Fatalf("slot %s chain not fully resolved: %+v", s.Slot, s.Chain)
			}
			if !strings.Contains(s.Source, " -> ") {
				t.Fatalf("slot %s source should be a chain: %q", s.Slot, s.Source)
			}
		}
	}
}

func TestOnlyReferencingInstanceUpdates(t *testing.T) {
	m := buildTestManifest(t)

	if !m.Impact.OnlyReferencingPathsUpdate {
		t.Fatal("impact must flag onlyReferencingPathsUpdate=true")
	}
	if !strings.Contains(m.After.Live.Material.Error, "has no kind") {
		t.Fatalf("after live material should fail kind contract: %q", m.After.Live.Material.Error)
	}
	if m.After.Archive.Material.Error != "" {
		t.Fatalf("independent copy must stay healthy, got %q", m.After.Archive.Material.Error)
	}
	if m.After.Archive.Material.FinalPath == m.After.Live.Material.FinalPath {
		t.Fatal("live and archive material final paths must be distinct assets")
	}
}

func TestMissingGeometryKeepsChainAndError(t *testing.T) {
	m := buildTestManifest(t)
	geo := m.Missing.Live.Geometry
	if geo.Error == "" || !strings.Contains(geo.Error, "no such file or directory") {
		t.Fatalf("missing geometry error mismatch: %q", geo.Error)
	}
	if geo.Kind != "" {
		t.Fatalf("missing geometry must not report a kind, got %q", geo.Kind)
	}
	last := geo.Chain[len(geo.Chain)-1]
	if last.Exists || !strings.HasSuffix(last.To, "showcase_missing.json") {
		t.Fatalf("last hop must be the absent geometry file: %+v", last)
	}
	if m.Missing.Archive.Geometry.Error != "" {
		t.Fatalf("archive geometry must remain resolved: %q", m.Missing.Archive.Geometry.Error)
	}
}

func TestChangesRecordFinalPathSource(t *testing.T) {
	m := buildTestManifest(t)
	if len(m.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(m.Changes))
	}
	for _, ch := range m.Changes {
		if ch.FinalPath == "" || ch.Source == "" || len(ch.Chain) != 2 {
			t.Fatalf("change %+v missing final path/source/chain", ch)
		}
		if !strings.HasPrefix(ch.Source, "instances/live.json -> models/live.json -> ") {
			t.Fatalf("change source must walk the reference chain: %q", ch.Source)
		}
	}
	if m.Changes[0].Before != "#F06A5D" || m.Changes[0].After != "#6C7BFF" {
		t.Fatalf("material change values mismatch: %+v", m.Changes[0])
	}
}

func TestEllipsizeKeepsShortStrings(t *testing.T) {
	if got := Ellipsize("ABC", 1, 0, 100); got != "ABC" {
		t.Fatalf("short text changed: %q", got)
	}
	got := Ellipsize("ABCDEFGHIJKLMNOP", 1, 0, 20)
	if !strings.HasSuffix(got, "...") || Measure(got, 1, 0) > 20 {
		t.Fatalf("ellipsized text invalid: %q width=%d", got, Measure(got, 1, 0))
	}
}
