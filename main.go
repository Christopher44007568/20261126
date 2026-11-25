package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ModeReport struct {
	Name           string        `json:"name"`
	File           string        `json:"file"`
	UpdateMillis   float64       `json:"update_millis"`
	BuildCount     int           `json:"build_count"`
	RefitCount     int           `json:"refit_count"`
	RebuildFrames  []int         `json:"rebuild_frames,omitempty"`
	FinalRebuild   bool          `json:"final_rebuild"`
	FinalMetrics   BoundsMetrics `json:"final_bounds_metrics"`
	Compare        CompareStats  `json:"compare_with_brute"`
	EmptyFallbacks int           `json:"empty_background_fallbacks"`
}

type Report struct {
	GeneratedAtUTC string            `json:"generated_at_utc"`
	Frames         int               `json:"frames"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	LeafSize       int               `json:"leaf_size"`
	Budget         BoundsBudget      `json:"bounds_budget"`
	Modes          []ModeReport      `json:"modes"`
	Timing         TimingSummary     `json:"timing"`
	Legend         map[string]string `json:"legend"`
}

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	frames := flag.Int("frames", 36, "animation frames; final frame is the stretched pose")
	width := flag.Int("width", 360, "image width")
	flag.Parse()

	config := DefaultSimulationConfig()
	config.Frames = *frames
	config.Camera.Width = *width
	config.Camera.Height = *width * 2 / 3
	if config.Camera.Height < 120 {
		config.Camera.Height = 120
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	refit := config.RunRefit()
	rebuild := config.RunRebuild()
	brute := config.RunBrute()
	adaptive := config.RunAdaptive()

	AnnotateComparison(refit.Image, refit.Truth, refit.Coverage)
	AnnotateComparison(rebuild.Image, rebuild.Truth, rebuild.Coverage)
	AnnotateComparison(adaptive.Image, adaptive.Truth, adaptive.Coverage)

	outputs := []struct {
		result ModeResult
		file   string
	}{
		{result: refit, file: "refit.ppm"},
		{result: rebuild, file: "rebuild.ppm"},
		{result: brute, file: "brute.ppm"},
		{result: adaptive, file: "adaptive.ppm"},
	}

	reports := make([]ModeReport, 0, len(outputs))
	for _, output := range outputs {
		path := filepath.Join(*outDir, output.file)
		if err := WritePPM(path, output.result.Image); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		reports = append(reports, modeReport(output.result, output.file))
	}

	report := Report{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Frames:         config.Frames,
		Width:          config.Camera.Width,
		Height:         config.Camera.Height,
		LeafSize:       config.LeafSize,
		Budget:         config.Budget,
		Modes:          reports,
		Timing:         SummarizeTiming(refit, rebuild),
		Legend: map[string]string{
			"red":  "brute-force hit, mode missed",
			"blue": "mode false hit: background or wrong triangle",
		},
	}
	reportPath := filepath.Join(*outDir, "report.json")
	writeReport(reportPath, report)
	printSummary(report)
}

func modeReport(result ModeResult, file string) ModeReport {
	return ModeReport{
		Name:           result.Name,
		File:           file,
		UpdateMillis:   float64(result.UpdateNanos) / 1e6,
		BuildCount:     result.BuildCount,
		RefitCount:     result.RefitCount,
		RebuildFrames:  result.RebuildFrames,
		FinalRebuild:   result.FinalRebuild,
		FinalMetrics:   result.FinalMetrics,
		Compare:        result.Stats,
		EmptyFallbacks: result.EmptyFallbacks,
	}
}

func writeReport(path string, report Report) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode report: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		log.Fatalf("write report: %v", err)
	}
}

func printSummary(report Report) {
	for _, mode := range report.Modes {
		fmt.Printf("%-8s %-12s hit=%5d miss=%4d false=%4d wrong=%4d update=%7.3fms rebuilds=%d fallback=%d\n",
			mode.Name,
			mode.File,
			mode.Compare.ModePixels,
			mode.Compare.MissedPixels,
			mode.Compare.FalseHitPixels,
			mode.Compare.WrongTriangle,
			mode.UpdateMillis,
			len(mode.RebuildFrames),
			mode.EmptyFallbacks,
		)
	}
	fmt.Printf("refit %.3f ms vs rebuild %.3f ms (%.2fx faster)\n",
		report.Timing.RefitMillis,
		report.Timing.RebuildMillis,
		ratio(report.Timing.RebuildMillis, report.Timing.RefitMillis),
	)
}

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
