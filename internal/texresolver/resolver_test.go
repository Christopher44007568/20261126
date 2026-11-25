package texresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUsesPriorityAndRecordsConflicts(t *testing.T) {
	base := t.TempDir()
	assetDir := filepath.Join(base, "assets")
	cacheDir := filepath.Join(base, "cache")
	fallbackDir := filepath.Join(base, "fallback")
	for _, dir := range []string{assetDir, cacheDir, fallbackDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(assetDir, "wood.png"), "asset")
	mustWrite(t, filepath.Join(cacheDir, "wood.png"), "cache")
	mustWrite(t, filepath.Join(fallbackDir, "wood.png"), "fallback")

	resolver := Resolver{Roots: []Root{
		{ID: "assets", Label: "Assets", Kind: "asset", Path: assetDir, Priority: 1},
		{ID: "cache", Label: "Cache", Kind: "cache", Path: cacheDir, Priority: 2},
		{ID: "fallback", Label: "Fallback", Kind: "fallback", Path: fallbackDir, Priority: 3},
	}}
	result, err := resolver.Resolve("wood.png")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.Selected == nil {
		t.Fatal("expected a resolved texture")
	}
	if result.Selected.RootID != "assets" {
		t.Fatalf("selected root = %q, want assets", result.Selected.RootID)
	}
	if got := filepath.ToSlash(result.Source); !strings.HasSuffix(got, "assets/wood.png") {
		t.Fatalf("source = %q, want the assets candidate", got)
	}
	if len(result.Existing) != 3 {
		t.Fatalf("existing candidates = %d, want 3", len(result.Existing))
	}
	if len(result.Conflicts) != 2 {
		t.Fatalf("conflicts = %d, want 2", len(result.Conflicts))
	}
	if result.Conflicts[0].RootID != "cache" || result.Conflicts[1].RootID != "fallback" {
		t.Fatalf("conflicts not in priority order: %#v", result.Conflicts)
	}
	if len(result.Trace) != 3 {
		t.Fatalf("trace length = %d, want 3", len(result.Trace))
	}
}

func TestResolveMissingUsesProceduralPlaceholder(t *testing.T) {
	dir := t.TempDir()
	resolver := Resolver{Roots: []Root{
		{ID: "assets", Label: "Assets", Kind: "asset", Path: dir, Priority: 1},
	}}
	result, err := resolver.Resolve("missing.png")
	if err != nil {
		t.Fatal(err)
	}
	if result.Found || result.Selected != nil {
		t.Fatal("missing texture should not select a file")
	}
	if !result.Placeholder {
		t.Fatal("missing texture should use the procedural placeholder")
	}
	if result.Source != "procedural://checkerboard" {
		t.Fatalf("source = %q", result.Source)
	}
}

func TestResolveRejectsPathTraversal(t *testing.T) {
	resolver := Resolver{}
	for _, name := range []string{"../secret.png", "dir/file.png", `dir\file.png`, ""} {
		if _, err := resolver.Resolve(name); err == nil {
			t.Fatalf("Resolve(%q) accepted an invalid filename", name)
		}
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
