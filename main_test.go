package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesThreeImagesAndLocatesFirstError(t *testing.T) {
	outDir := t.TempDir()
	var stdout bytes.Buffer
	err := run(
		[]string{"-out", outDir, "-width", "160", "-height", "100"},
		&stdout,
	)
	if err != nil {
		t.Fatalf("run failed: %v\nstdout:\n%s", err, stdout.String())
	}

	names := []string{
		"material_index.png",
		"hit_groups.png",
		"shaded.png",
		"broken_material_index.png",
		"broken_hit_groups.png",
		"broken_shaded.png",
		"mapping_diff.png",
	}
	for _, name := range names {
		path := filepath.Join(outDir, name)
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if img.Bounds().Dx() != 160 || img.Bounds().Dy() != 100 {
			t.Fatalf("%s has size %v", name, img.Bounds())
		}
	}

	data, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report RenderReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.DeliberateChanges) != 1 {
		t.Fatalf("report has %d deliberate changes, want 1", len(report.DeliberateChanges))
	}
	if !report.FirstError.Found {
		t.Fatal("report does not contain a first error")
	}
	if report.FirstError.ObjectName != "back_stone_wall" {
		t.Fatalf("first error object = %q", report.FirstError.ObjectName)
	}
	if report.FirstError.ExpectedHitGroup != "stone" || report.FirstError.ActualHitGroup != "glass" {
		t.Fatalf("first error hit groups = %q -> %q",
			report.FirstError.ExpectedHitGroup, report.FirstError.ActualHitGroup)
	}
	if len(report.ReferenceObjectMappings) != 8 || len(report.BrokenObjectMappings) != 8 {
		t.Fatalf("report mapping tables have lengths %d and %d",
			len(report.ReferenceObjectMappings), len(report.BrokenObjectMappings))
	}
}
