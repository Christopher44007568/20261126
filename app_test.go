package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildArtifactsWritesCompleteSet(t *testing.T) {
	outDir := t.TempDir()
	output, err := buildArtifacts(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Images) != len(RenderModes) {
		t.Fatalf("images = %d, want %d", len(output.Images), len(RenderModes))
	}
	if len(output.Provenance) != len(RenderModes) {
		t.Fatalf("provenance = %d, want %d", len(output.Provenance), len(RenderModes))
	}
	expected := append([]string{output.ContactSheet, filepath.Join(outDir, "audit.json")}, output.Images...)
	expected = append(expected, output.Provenance...)
	for _, path := range expected {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing artifact %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("empty artifact %s", path)
		}
	}
	if err := validateAudit(output.Audit); err != nil {
		t.Fatal(err)
	}
}
