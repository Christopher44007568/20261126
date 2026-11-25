package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateWritesThreeDetailComparisons(t *testing.T) {
	out := t.TempDir()
	art, err := Generate(out)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	required := []string{
		"low.ppm",
		"medium.ppm",
		"high.ppm",
		"comparison_low.ppm",
		"comparison_medium.ppm",
		"comparison_high.ppm",
		"full_low.ppm",
		"chunked_medium.ppm",
		"screen-error_high.ppm",
		"streaming_report.json",
		"streaming_report.md",
	}
	for _, name := range required {
		path := filepath.Join(out, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing artifact %s: %v", name, err)
		}
		if info.Size() <= 0 {
			t.Fatalf("artifact %s is empty", name)
		}
	}

	if len(art.FrameImages["LOW"]) != 3 {
		t.Fatalf("LOW frames = %d, want 3 strategies", len(art.FrameImages["LOW"]))
	}
}

func TestStrategyMetricsShowScreenErrorEfficiency(t *testing.T) {
	art, err := Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	metrics := make(map[string]map[string]FrameMetrics)
	for _, strategy := range art.Report.Strategies {
		metrics[strategy.Name] = make(map[string]FrameMetrics)
		for _, frame := range strategy.Frames {
			metrics[strategy.Name][frame.Detail] = frame
		}
	}

	for _, strategyName := range []string{"full", "chunked", "screen-error"} {
		for _, detail := range []string{"LOW", "MEDIUM", "HIGH"} {
			m := metrics[strategyName][detail]
			statusSum := m.ExactPixels + m.CoarsePixels + m.RootPixels
			if statusSum != m.MeshPixels {
				t.Fatalf("%s/%s fallback sum %d != mesh %d", strategyName, detail, statusSum, m.MeshPixels)
			}
			if m.LoadingPixels > m.MeshPixels {
				t.Fatalf("%s/%s loading %d > mesh %d", strategyName, detail, m.LoadingPixels, m.MeshPixels)
			}
		}
	}

	screenHigh := metrics["screen-error"]["HIGH"]
	screenMedium := metrics["screen-error"]["MEDIUM"]
	fullHigh := metrics["full"]["HIGH"]
	chunkHigh := metrics["chunked"]["HIGH"]

	if screenMedium.ExactPixels == 0 || screenMedium.CoarsePixels == 0 || screenMedium.LoadingPixels == 0 {
		t.Fatalf("SSE medium should show exact, coarse-parent and loading overlays: %+v", screenMedium)
	}
	if screenHigh.TargetCoverage < 99.9 {
		t.Fatalf("screen-error high coverage = %.1f%%, want 100%%", screenHigh.TargetCoverage)
	}
	if screenHigh.BytesLoaded >= fullHigh.BytesLoaded {
		t.Fatalf("screen-error bytes %d should be below full %d", screenHigh.BytesLoaded, fullHigh.BytesLoaded)
	}
	if screenHigh.BytesLoaded >= chunkHigh.BytesLoaded {
		t.Fatalf("screen-error bytes %d should be below chunked %d", screenHigh.BytesLoaded, chunkHigh.BytesLoaded)
	}
	if chunkHigh.TargetCoverage >= screenHigh.TargetCoverage {
		t.Fatalf("chunk coverage %.1f should be below SSE %.1f", chunkHigh.TargetCoverage, screenHigh.TargetCoverage)
	}
	if !art.Report.Verification.Passed {
		t.Fatalf("continuity verification failed: %v", art.Report.Verification.Checks)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	if _, err := Generate(first); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(second); err != nil {
		t.Fatal(err)
	}

	name := "screen-error_high.ppm"
	a, err := os.ReadFile(filepath.Join(first, name))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(second, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("%s differs between identical runs", name)
	}
}
