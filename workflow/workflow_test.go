package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesCompleteArtifactsAndVerifiesReload(t *testing.T) {
	out := t.TempDir()
	result, err := Run(out)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Roundtrip.NoColorShift {
		t.Fatalf("round trip reported color shift: %+v", result.Roundtrip)
	}
	if len(result.Rows) != 9 {
		t.Fatalf("metadata rows = %d, want 9 (3 assets x 3 levels)", len(result.Rows))
	}
	if len(result.Diagnostic.Levels) != 3 {
		t.Fatalf("diagnostic levels = %d, want 3", len(result.Diagnostic.Levels))
	}
	if result.Diagnostic.FirstFailingLevel != 0 {
		t.Fatalf("first failing level = %d, want 0", result.Diagnostic.FirstFailingLevel)
	}

	files := []string{
		result.Files.AlbedoContainer,
		result.Files.NormalContainer,
		result.Files.MislabeledNormalContainer,
		result.Files.AlbedoAtlas,
		result.Files.NormalAtlas,
		result.Files.DeltaAtlas,
		result.Files.MetadataCSV,
		result.Files.MetadataJSON,
		result.Files.MetadataMarkdown,
		result.Files.DiagnosticJSON,
		result.Files.DiagnosticMarkdown,
		result.Files.RoundtripJSON,
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("artifact %s is empty", path)
		}
	}

	csvData, err := os.ReadFile(result.Files.MetadataCSV)
	if err != nil {
		t.Fatal(err)
	}
	csvText := string(csvData)
	for _, fragment := range []string{"albedo_srgb.ktxc", "normal_linear.ktxc", ",2,128,128,", "true"} {
		if !strings.Contains(csvText, fragment) {
			t.Fatalf("metadata CSV missing %q", fragment)
		}
	}
}

func TestGeneratedMipSizesAreRequested(t *testing.T) {
	if len(MipSizes) != 3 || MipSizes[0] != 512 || MipSizes[1] != 256 || MipSizes[2] != 128 {
		t.Fatalf("unexpected mip sizes: %v", MipSizes)
	}
	if filepath.Base(makePaths("x").MetadataCSV) != "mip_metadata.csv" {
		t.Fatal("unexpected metadata filename")
	}
}
