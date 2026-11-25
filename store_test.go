package main

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestKeeperRetainsLastGood(t *testing.T) {
	dir := t.TempDir()
	k := NewKeeper(dir, 2)
	good := CalibrationResult{
		Camera: DefaultCamera(), RMSE: 0.2, Points: 108, Iterations: 4,
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 200, B: 90, A: 255})
	files := map[string]*image.RGBA{
		"original_grid.png": img, "corrected_grid.png": img, "residual_heatmap.png": img,
	}
	if _, err := k.Publish(good, files); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "original_grid.png"))
	if err != nil {
		t.Fatal(err)
	}

	bad := good
	bad.RMSE = 88
	bad.Camera.Fx = 0
	if _, err := k.Publish(bad, map[string]*image.RGBA{"original_grid.png": image.NewRGBA(image.Rect(0, 0, 16, 16))}); err == nil {
		t.Fatal("bad calibration accepted")
	}
	after, err := os.ReadFile(filepath.Join(dir, "original_grid.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("canonical image was overwritten by bad parameters")
	}
	if k.Generation != 1 || k.LastGood == nil || k.LastGood.RMSE != 0.2 {
		t.Fatalf("last good changed: generation=%d", k.Generation)
	}
	if _, err := os.Stat(filepath.Join(dir, "fallback_events.jsonl")); err != nil {
		t.Fatalf("fallback log missing: %v", err)
	}

	nanBad := good
	nanBad.RMSE = math.Inf(1)
	nanBad.Camera.Cx = math.NaN()
	if _, err := k.Publish(nanBad, nil); err == nil {
		t.Fatal("NaN calibration accepted")
	}
	lines, err := os.ReadFile(filepath.Join(dir, "fallback_events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(string(lines)) == 0 || byteCount(lines, '\n') < 2 {
		t.Fatalf("expected finite and non-finite rejection events, got %q", lines)
	}
}

func byteCount(data []byte, b byte) int {
	n := 0
	for _, x := range data {
		if x == b {
			n++
		}
	}
	return n
}
