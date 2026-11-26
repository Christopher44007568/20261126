package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nsmigrate/report"
)

func TestRunEndToEndArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir); err != nil {
		t.Fatalf("run: %v", err)
	}

	files := []string{
		"migration_before.ppm", "migration_before.png",
		"migration_after.ppm", "migration_after.png",
		"naive_string_replace_after.ppm", "naive_string_replace_after.png",
		"red_boxes.json", "migration_report.json", "migration_report.md",
	}
	for _, name := range files {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.Size() == 0 {
			t.Fatalf("artifact %s missing or empty: %v", name, err)
		}
	}

	// PPM must be binary P6.
	head, err := os.ReadFile(filepath.Join(dir, "migration_after.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(head), "P6\n") {
		t.Fatal("after image is not a P6 PPM")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "red_boxes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest report.RedBoxesManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("bad red_boxes.json: %v", err)
	}
	if manifest.BoxColor != [3]int{255, 0, 0} {
		t.Fatalf("box color=%v", manifest.BoxColor)
	}
	if len(manifest.Images) != 3 {
		t.Fatalf("images=%d want 3", len(manifest.Images))
	}

	// Every recorded box must correspond to pure-red pixels in its PNG.
	for _, im := range manifest.Images {
		pngName := strings.TrimSuffix(im.Image, ".ppm") + ".png"
		f, err := os.Open(filepath.Join(dir, pngName))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", pngName, err)
		}
		if len(im.Boxes) == 0 {
			t.Fatalf("%s has no red boxes", im.Image)
		}
		for _, b := range im.Boxes {
			r, g, bl, _ := img.At(b.X+4, b.Y+2).RGBA()
			if uint8(r>>8) != 255 || uint8(g>>8) != 0 || uint8(bl>>8) != 0 {
				t.Fatalf("%s box %s border not red: rgb=(%d,%d,%d)",
					im.Image, b.ID, uint8(r>>8), uint8(g>>8), uint8(bl>>8))
			}
		}

		// The same top-edge sample must be pure red in the P6 PPM itself.
		ppmPath := filepath.Join(dir, im.Image)
		pw, ph, pix := parseP6(t, ppmPath)
		if pw != im.Width || ph != im.Height {
			t.Fatalf("%s PPM dims %dx%d vs manifest %dx%d", im.Image, pw, ph, im.Width, im.Height)
		}
		for _, b := range im.Boxes {
			off := (b.Y+2)*pw*3 + (b.X+4)*3
			if pix[off] != 255 || pix[off+1] != 0 || pix[off+2] != 0 {
				t.Fatalf("%s PPM box %s not red: %v", im.Image, b.ID, pix[off:off+3])
			}
		}
	}

	md, err := os.ReadFile(filepath.Join(dir, "migration_report.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(md)
	concreteOldPaths := []string{
		"/Show/OldNamespace/Assets/ChairRig",
		"/Show/OldNamespace/Materials/RedPlastic",
		"/Show/OldNamespace/Materials/BlueMetal",
		"/Show/OldNamespace/Rig/FocusTarget",
	}
	for _, p := range concreteOldPaths {
		if !strings.Contains(text, p) {
			t.Errorf("markdown report missing concrete old path %q", p)
		}
	}
}

// parseP6 is a minimal binary PPM parser for the files produced by raster.
func parseP6(t *testing.T, path string) (int, int, []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pos := 0
	readToken := func() string {
		for pos < len(data) && (data[pos] == ' ' || data[pos] == '\n' || data[pos] == '\t' || data[pos] == '\r') {
			pos++
		}
		start := pos
		for pos < len(data) && data[pos] != ' ' && data[pos] != '\n' && data[pos] != '\t' && data[pos] != '\r' {
			pos++
		}
		return string(data[start:pos])
	}
	readInt := func() int {
		tok := readToken()
		n := 0
		for _, ch := range tok {
			n = n*10 + int(ch-'0')
		}
		return n
	}
	if magic := readToken(); magic != "P6" {
		t.Fatalf("%s magic=%q want P6", path, magic)
	}
	w, h, maxVal := readInt(), readInt(), readInt()
	if maxVal != 255 {
		t.Fatalf("maxval=%d", maxVal)
	}
	// Exactly one separator whitespace follows the maxval token.
	pos++
	if pos+w*h*3 > len(data) {
		t.Fatalf("pixel data truncated")
	}
	return w, h, data[pos : pos+w*h*3]
}
