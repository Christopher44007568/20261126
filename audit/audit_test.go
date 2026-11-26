package audit_test

import (
	"testing"

	"nsmigrate/audit"
	"nsmigrate/migrate"
	"nsmigrate/scene"
)

func TestBeforeAuditKnowsEdgeCases(t *testing.T) {
	s := scene.BuildDemoStage()
	rep := audit.Audit("before", s, scene.FixtureMapping())
	if !rep.InvariantOK {
		t.Fatalf("before fixture should match expectations: %+v", rep.Findings)
	}
	if rep.Counts[string(audit.StatusBroken)] != 1 {
		t.Fatalf("broken count=%d want 1 (R2)", rep.Counts[string(audit.StatusBroken)])
	}
	if rep.Counts[string(audit.StatusDefault)] != 1 {
		t.Fatalf("default count=%d want 1 (M3)", rep.Counts[string(audit.StatusDefault)])
	}
}

func TestCorrectMigrationPreservesAllThreeLinkKinds(t *testing.T) {
	old := scene.BuildDemoStage()
	next, _ := migrate.MigrateCorrect(old, scene.FixtureMapping())
	rep := audit.Audit("correct", next, scene.FixtureMapping())

	if !rep.InvariantOK {
		t.Fatalf("correct migration must satisfy every expectation: %+v", rep.Findings)
	}

	wantOK := map[string]bool{"R1": true, "M1": true, "M2": true, "C1": true}
	for _, f := range rep.Findings {
		if wantOK[f.LinkID] && f.Status != audit.StatusOK {
			t.Errorf("%s (%s) must resolve after migration, got %s", f.LinkID, f.Kind, f.Status)
		}
	}

	// One healthy arc of each required kind.
	for _, kind := range []scene.LinkKind{scene.KindReference, scene.KindMaterialBinding, scene.KindCameraTarget} {
		var ok bool
		for _, f := range rep.FindingsFor(kind) {
			if f.Status == audit.StatusOK {
				ok = true
			}
		}
		if !ok {
			t.Errorf("kind %s has no resolving link after correct migration", kind)
		}
	}
}

func TestStringOnlyDiagnosticQuotesConcreteOldPaths(t *testing.T) {
	old := scene.BuildDemoStage()
	m := scene.FixtureMapping()
	naive := migrate.MigrateStringOnly(old, m)
	rep := audit.Audit("string-only", naive, m)

	if rep.InvariantOK {
		t.Fatal("string-only migration must fail the invariant")
	}

	wantOld := map[string]string{
		"R1": "/Show/OldNamespace/Assets/ChairRig",
		"M1": "/Show/OldNamespace/Materials/RedPlastic",
		"M2": "/Show/OldNamespace/Materials/BlueMetal",
		"C1": "/Show/OldNamespace/Rig/FocusTarget",
	}
	byID := map[string]audit.Finding{}
	for _, f := range rep.Findings {
		byID[f.LinkID] = f
	}
	for id, oldPath := range wantOld {
		f := byID[id]
		if f.OldPath != oldPath {
			t.Errorf("%s OldPath=%q want concrete %q", id, f.OldPath, oldPath)
		}
		if f.StalePath != oldPath {
			t.Errorf("%s StalePath=%q want stale parent/child path %q", id, f.StalePath, oldPath)
		}
		if f.Status == audit.StatusOK {
			t.Errorf("%s must be dangling under string-only edit", id)
		}
	}

	// The material bindings collapse onto retained defaults.
	var defaults int
	for _, d := range rep.Defaults {
		if d.ValueName != scene.DefaultMaterialName || d.OldPath == "" {
			t.Errorf("bad default record: %+v", d)
		}
		defaults++
	}
	if defaults < 3 {
		t.Fatalf("expected at least M1/M2/M3 retained defaults under naive edit, got %d", defaults)
	}
}
