package audit

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAuditEndToEnd(t *testing.T) {
	directory := t.TempDir()
	writeSolid := func(name string, value Vec3) {
		t.Helper()
		picture := NewImage(4, 4)
		for index := range picture.Pixels {
			picture.Pixels[index] = value
		}
		if err := writePNG(filepath.Join(directory, name), picture); err != nil {
			t.Fatal(err)
		}
	}
	writeSolid("a.data", Vec3{0.5, 0.12, 0.04})
	writeSolid("b.data", Vec3{0.5, 0.5, 1})
	writeSolid("c.data", Vec3{0.35, 0.35, 0.35})

	manifest := tinyManifest()
	manifest.Textures = TextureSet{
		Albedo:    TextureDeclaration{Path: "a.data", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Normal:    TextureDeclaration{Path: "b.data", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Roughness: TextureDeclaration{Path: "c.data", ColorSpace: ColorSpaceLinear, Channel: "r"},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(directory, "scene.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	outputDirectory := filepath.Join(directory, "output")
	report, err := RunAudit(manifestPath, outputDirectory, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Invariants.NormalLinear || !report.Invariants.RoughnessLinear {
		t.Fatalf("data-space invariants are false: %+v", report.Invariants)
	}
	if report.Invariants.MisreadModifiedDataChannels {
		t.Fatalf("misread reported data-channel changes")
	}
	if report.Metrics.Max <= 0 {
		t.Fatalf("expected a non-zero albedo color-space error")
	}

	expectedFiles := []string{
		"correct.png",
		"forced-srgb-albedo.png",
		"comparison.png",
		"local-error.png",
		"report.json",
	}
	for _, name := range expectedFiles {
		info, err := os.Stat(filepath.Join(outputDirectory, name))
		if err != nil {
			t.Fatalf("missing output %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("empty output %s", name)
		}
	}

	file, err := os.Open(filepath.Join(outputDirectory, "correct.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 48 || decoded.Bounds().Dy() != 48 {
		t.Fatalf("correct render has dimensions %v", decoded.Bounds())
	}
}
