package main

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestMeasurePatchDetectsNewClipping(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	target := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fillRect(source, source.Bounds(), color.RGBA{R: 128, G: 128, B: 128, A: 255})
	fillRect(target, target.Bounds(), color.RGBA{R: 255, G: 128, B: 128, A: 255})
	got := measurePatch(source, target, image.Rect(0, 0, 4, 4), 254, 1)
	if math.Abs(got.NewClip-1) > 1e-9 {
		t.Fatalf("new clip = %.4f, want 1.0", got.NewClip)
	}
	if math.Abs(got.DeltaL) < 1 {
		t.Fatalf("delta L = %.4f, expected a visible brightness shift", got.DeltaL)
	}
}

func TestExampleAveragePassesButLocalFailuresRemain(t *testing.T) {
	m, err := loadManifest("examples/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := renderSwatches(m, "source")
	if err != nil {
		t.Fatal(err)
	}
	target, err := renderSwatches(m, "target")
	if err != nil {
		t.Fatal(err)
	}
	result := analyze(m, source, target)
	if !result.Summary.AveragePass {
		t.Fatalf("mean deltaE %.3f should pass aggregate limit %.3f",
			result.Summary.MeanDeltaE, m.Thresholds.DeltaE)
	}
	if result.Summary.LocalPass || result.Summary.Status != "fail" {
		t.Fatalf("local verdict = %t, status = %s; want local fail", result.Summary.LocalPass, result.Summary.Status)
	}
	if result.Summary.Failed != 4 {
		t.Fatalf("failed patches = %d, want 4", result.Summary.Failed)
	}
	joined := strings.Join(result.Summary.LocalFailures, "\n")
	for _, id := range []string{"SKIN-03", "HL-01", "N-01", "N-02"} {
		if !strings.Contains(joined, id) {
			t.Fatalf("local failures missing %s: %s", id, joined)
		}
	}
	if result.Groups[0].Pass {
		t.Fatal("neutral ramp monotonicity should fail")
	}
}

func TestMonotonicityMarksBothSidesOfBrokenStep(t *testing.T) {
	m := &Manifest{
		Title:  "ramp",
		Canvas: CanvasSpec{Width: 64, Height: 32},
		Groups: []GroupSpec{{ID: "ramp", Direction: "increasing"}},
		Thresholds: Thresholds{
			DeltaE:             100,
			Brightness:         100,
			ClipRatio:          1,
			MonotonicityDeltaL: 0.4,
			ClipHigh:           254,
			ClipLow:            1,
		},
		Patches: []PatchSpec{
			{ID: "A", Group: "ramp", Order: 0, Rect: RectSpec{0, 0, 16, 16}, Source: "#333333", Target: "#555555"},
			{ID: "B", Group: "ramp", Order: 1, Rect: RectSpec{16, 0, 16, 16}, Source: "#666666", Target: "#444444"},
			{ID: "C", Group: "ramp", Order: 2, Rect: RectSpec{32, 0, 16, 16}, Source: "#999999", Target: "#AAAAAA"},
		},
	}
	m.applyDefaults()
	source, err := renderSwatches(m, "source")
	if err != nil {
		t.Fatal(err)
	}
	target, err := renderSwatches(m, "target")
	if err != nil {
		t.Fatal(err)
	}
	result := analyze(m, source, target)
	if len(result.Groups) != 1 || result.Groups[0].Pass {
		t.Fatalf("groups = %+v, want one failed ramp", result.Groups)
	}
	for _, p := range result.Patches {
		if p.ID == "A" || p.ID == "B" {
			if !hasFailure(p, "monotonicity") {
				t.Fatalf("patch %s should be marked for monotonicity: %+v", p.ID, p.Failures)
			}
		}
	}
}

func hasFailure(p PatchResult, code string) bool {
	for _, f := range p.Failures {
		if f.Code == code {
			return true
		}
	}
	return false
}
