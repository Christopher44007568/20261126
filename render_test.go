package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestComparisonRendersFullFramesAndThresholdColors(t *testing.T) {
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
	chart := renderChart(m, result, m.Output.Amplification)
	if chart.Bounds().Dx() < 1000 || chart.Bounds().Dy() < 500 {
		t.Fatalf("comparison is too small: %v", chart.Bounds())
	}

	if got := chart.RGBAAt(336, 327); got != (color.RGBA{R: 0x7D, G: 0x7D, B: 0x7D, A: 255}) {
		t.Fatalf("source frame was not retained: got %#v", got)
	}
	if got := chart.RGBAAt(716, 327); got != (color.RGBA{R: 0x82, G: 0x82, B: 0x82, A: 255}) {
		t.Fatalf("target frame was not retained: got %#v", got)
	}

	wantColors := map[string]color.RGBA{
		"delta_e":      {R: 255, G: 45, B: 141, A: 255},
		"gamut_clip":   {R: 39, G: 199, B: 255, A: 255},
		"monotonicity": {R: 255, G: 176, B: 32, A: 255},
	}
	found := make(map[string]bool)
	b := chart.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := chart.RGBAAt(x, y)
			for code, want := range wantColors {
				if c == want {
					found[code] = true
				}
			}
		}
	}
	for code := range wantColors {
		if !found[code] {
			t.Fatalf("threshold color %s is not visible", code)
		}
	}
}

func TestWriteOutputsAndReport(t *testing.T) {
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

	dir := t.TempDir()
	out := filepath.Join(dir, "nested", "comparison.png")
	reportPath := filepath.Join(dir, "nested", "comparison.report.json")
	if err := writePNG(out, renderChart(m, result, m.Output.Amplification)); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Empty() {
		t.Fatal("encoded comparison is empty")
	}

	report := buildReport(m, result, out, m.Output.Amplification)
	if err := writeReport(reportPath, report); err != nil {
		t.Fatal(err)
	}
	if !report.AveragePass || report.LocalPass || report.Status != "fail" {
		t.Fatalf("report verdict = %+v", report)
	}
	if !report.Visual.SourceRetained || !report.Visual.TargetRetained || !report.Visual.ErrorMapRendered {
		t.Fatalf("visual retention flags = %+v", report.Visual)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultReportPath(t *testing.T) {
	if got := defaultReportPath("out/chart.png"); got != "out/chart.report.json" {
		t.Fatalf("got %q", got)
	}
	if got := defaultReportPath("chart"); got != "chart.report.json" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderChartAcceptsPortraitFrames(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 120, 300))
	target := image.NewRGBA(image.Rect(0, 0, 120, 300))
	fillRect(source, source.Bounds(), color.RGBA{R: 80, G: 80, B: 80, A: 255})
	fillRect(target, target.Bounds(), color.RGBA{R: 84, G: 84, B: 84, A: 255})
	m := &Manifest{
		Title:  "PORTRAIT",
		Canvas: CanvasSpec{Width: 120, Height: 300},
		Patches: []PatchSpec{
			{ID: "P", Rect: RectSpec{0, 0, 120, 300}, Source: "#505050", Target: "#545454"},
		},
	}
	m.applyDefaults()
	result := analyze(m, source, target)
	chart := renderChart(m, result, 8)
	if chart.Bounds().Dx() < 1000 {
		t.Fatalf("portrait comparison width = %d, want at least 1000", chart.Bounds().Dx())
	}
}
