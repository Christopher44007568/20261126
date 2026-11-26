package render

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateArtifacts(t *testing.T) {
	dir := t.TempDir()
	files, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(expectedFiles) {
		t.Fatalf("files = %d, want %d", len(files), len(expectedFiles))
	}
	for _, name := range files {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	checkPNGSize(t, filepath.Join(dir, "bit_gradient_puzzle.png"), puzzleW, puzzleH)
	checkPNGSize(t, filepath.Join(dir, "sheet_12bit_to_8bit.png"), sheetW, sheetH)
	checkPNGSize(t, filepath.Join(dir, "master_12bit_16bit.png"), masterW, masterH)
}

func checkPNGSize(t *testing.T, path string, wantW, wantH int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width != wantW || img.Height != wantH {
		t.Fatalf("%s = %dx%d, want %dx%d", path, img.Width, img.Height, wantW, wantH)
	}
}

func TestMasterIs16Bit(t *testing.T) {
	dir := t.TempDir()
	if _, err := Generate(dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "master_10bit_16bit.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	switch img.(type) {
	case *image.NRGBA64, *image.RGBA64:
	default:
		t.Fatalf("decoded master type = %T, want a 16-bit NRGBA/RGBA image", img)
	}
}
