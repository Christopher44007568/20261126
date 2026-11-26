package migrate_test

import (
	"testing"

	"nsmigrate/migrate"
	"nsmigrate/scene"
)

func TestCorrectMigrationRerootsAllPrimAndLinks(t *testing.T) {
	old := scene.BuildDemoStage()
	m := scene.FixtureMapping()

	next, rep := migrate.MigrateCorrect(old, m)

	if !rep.ParentChildClean {
		t.Fatalf("correct migration must keep parent/child paths consistent")
	}
	if rep.RewrittenPrims != len(old.Prims) {
		t.Fatalf("rewritten prims=%d want %d", rep.RewrittenPrims, len(old.Prims))
	}
	if rep.RewrittenLinks != len(old.Links) {
		t.Fatalf("rewritten links=%d want %d", rep.RewrittenLinks, len(old.Links))
	}

	if p := next.PrimByPath(scene.NewRoot + "/Materials/RedPlastic"); p == nil {
		t.Fatal("expected material at new path")
	}
	if p := next.PrimByPath(scene.OldRoot + "/Materials/RedPlastic"); p != nil {
		t.Fatal("old material path must be gone after re-root")
	}

	wantTargets := map[string]string{
		"R1": scene.NewRoot + "/Assets/ChairRig",
		"M1": scene.NewRoot + "/Materials/RedPlastic",
		"C1": scene.NewRoot + "/Rig/FocusTarget",
	}
	for id, want := range wantTargets {
		if got := next.LinkByID(id).Target; got != want {
			t.Errorf("link %s target=%q want %q", id, got, want)
		}
	}

	// Migration must not mutate the before snapshot.
	if got := old.LinkByID("M1").Target; got != scene.OldRoot+"/Materials/RedPlastic" {
		t.Fatalf("before stage mutated: M1=%q", got)
	}
}

func TestStringOnlyLeavesParentChildDesync(t *testing.T) {
	old := scene.BuildDemoStage()
	m := scene.FixtureMapping()

	naive := migrate.MigrateStringOnly(old, m)
	mismatches := migrate.CheckParentChild(naive)
	if len(mismatches) == 0 {
		t.Fatal("string-only edit must leave parent/child path mismatches")
	}

	// Root string renamed ...
	if naive.PrimByID("root").Path != scene.NewRoot {
		t.Fatalf("root path=%q", naive.PrimByID("root").Path)
	}
	// ... but the child still carries the old parent-child path.
	if p := naive.PrimByPath(scene.OldRoot + "/Materials/RedPlastic"); p == nil {
		t.Fatal("descendant should remain at its old parent-child path under naive edit")
	}
	if got := naive.LinkByID("M1").Target; got != scene.NewRoot+"/Materials/RedPlastic" {
		t.Fatalf("M1 string target=%q", got)
	}
}
