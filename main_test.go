package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"texpath/internal/texresolver"
)

func TestBuildSceneAggregatesThreeSourceGroups(t *testing.T) {
	base := t.TempDir()
	roots := []texresolver.Root{
		{ID: "assets", Label: "本地资产", Kind: "asset", Path: filepath.Join(base, "assets"), Priority: 1},
		{ID: "cache", Label: "运行时缓存", Kind: "cache", Path: filepath.Join(base, "cache"), Priority: 2},
		{ID: "fallback", Label: "最终回退", Kind: "fallback", Path: filepath.Join(base, "fallback"), Priority: 3},
	}
	for _, root := range roots {
		if err := os.MkdirAll(root.Path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string][]string{
		roots[0].Path: {"oak_albedo.png", "oak_normal.png", "oak_roughness.png"},
		roots[1].Path: {"oak_albedo.png", "brass_albedo.png", "brass_normal.png"},
		roots[2].Path: {"brass_albedo.png", "brass_roughness.png", "clay_normal.png"},
	}
	for dir, names := range files {
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("texture"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	application := &app{
		cfg:      config{SearchOrder: roots},
		resolver: texresolver.Resolver{Roots: roots},
	}
	scene, err := application.buildScene()
	if err != nil {
		t.Fatal(err)
	}

	if scene.Summary.Total != 9 || scene.Summary.Resolved != 7 || scene.Summary.Placeholder != 2 {
		t.Fatalf("summary = %#v", scene.Summary)
	}
	if scene.Summary.ConflictMaps != 2 || scene.Summary.ConflictFiles != 2 {
		t.Fatalf("conflict summary = %#v", scene.Summary)
	}
	if len(scene.Groups) != 3 {
		t.Fatalf("groups = %d, want 3", len(scene.Groups))
	}
	if scene.Groups[0].Count != 3 || scene.Groups[1].Count != 2 {
		t.Fatalf("asset/cache counts = %d/%d", scene.Groups[0].Count, scene.Groups[1].Count)
	}
	if scene.Groups[2].Count != 4 || scene.Groups[2].FileCount != 2 || scene.Groups[2].PlaceholderCount != 2 {
		t.Fatalf("fallback group = %#v", scene.Groups[2])
	}
	if scene.SearchOrder[0].Hits != 3 || scene.SearchOrder[1].Duplicates != 1 || scene.SearchOrder[2].Duplicates != 1 {
		t.Fatalf("search order = %#v", scene.SearchOrder)
	}

	oak := scene.Materials[0].Maps[0]
	if !strings.HasSuffix(oak.Source, "assets/oak_albedo.png") {
		t.Fatalf("oak source = %q", oak.Source)
	}
	if len(oak.Conflicts) != 1 || !strings.HasSuffix(oak.Conflicts[0].Path, "cache/oak_albedo.png") {
		t.Fatalf("oak conflicts = %#v", oak.Conflicts)
	}
	if oak.URL != "/texture/assets/oak_albedo.png" {
		t.Fatalf("oak URL = %q", oak.URL)
	}

	missing := scene.Materials[2].Maps[0]
	if !missing.Placeholder || missing.URL != "/texture/_placeholder.png" {
		t.Fatalf("missing map = %#v", missing)
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	t.Setenv("TEXPATH_CONFIG", filepath.Join(t.TempDir(), "absent.json"))
	cfg, source, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if source != "built-in defaults" || len(cfg.SearchOrder) != 3 {
		t.Fatalf("config = %#v source=%q", cfg, source)
	}
}
