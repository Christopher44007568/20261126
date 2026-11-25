package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "out", "directory for PPM, JSON, and DOT artifacts")
	width := flag.Int("width", defaultWidth, "render width")
	height := flag.Int("height", defaultHeight, "render height")
	flag.Parse()

	sim, err := RunSimulation(*width, *height)
	if err != nil {
		fmt.Fprintln(os.Stderr, "simulation:", err)
		os.Exit(1)
	}
	report := sim.BuildReport()
	if err := WriteArtifacts(report, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "write artifacts:", err)
		os.Exit(1)
	}
	if err := VerifyReport(report); err != nil {
		fmt.Fprintln(os.Stderr, "safety check:", err)
		os.Exit(1)
	}

	fmt.Printf("frames: %d, ghost pixels: %d\n", report.Metrics.FramesRendered, report.Metrics.GhostPixels)
	fmt.Printf("safe slot reuses: %d/%d, dangling refs: %d, premature releases: %d\n",
		report.Metrics.SafeReuseEvents, report.Metrics.ReuseEvents,
		report.Metrics.DanglingRefs, report.Metrics.PrematureReleases)
	for i := range report.Frames {
		fmt.Printf("  frame %d: %s\n", i+1, filepath.Join(*outDir, fmt.Sprintf("frame_%03d.ppm", i+1)))
	}
	fmt.Printf("  timeline: %s\n", filepath.Join(*outDir, "reuse_intervals.ppm"))
	fmt.Printf("  reference graph: %s\n", filepath.Join(*outDir, "dangling_references.ppm"))
	fmt.Printf("  report: %s\n", filepath.Join(*outDir, "report.json"))
}
