package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunGeneratesArtifacts(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "mips")
	err := Run([]string{"--out", outputDir, "--size", "32"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	required := []string{
		"color_mip_montage.png",
		"normal_mip_montage.png",
		"seam_zoom.png",
		"normal_seam_zoom.png",
		"edge_probe_zoom.png",
		"average-color-error.png",
		"average-color-error.csv",
		"metrics.json",
		"report.md",
		filepath.Join("levels", "color_L05_1x1.png"),
		filepath.Join("levels", "normal_L05_1x1.png"),
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("required artifact %q: %v", name, err)
		}
	}
}
