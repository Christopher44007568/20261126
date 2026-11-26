package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	out := flag.String("out", "artifacts", "directory for PNG, JSON and HTML outputs")
	width := flag.Int("width", 480, "render width in pixels")
	height := flag.Int("height", 270, "render height in pixels")
	probeRays := flag.Int("probe-rays", 1024, "deterministic random probe rays per suite")
	flag.Parse()

	if *width < 16 || *height < 16 {
		fmt.Fprintln(os.Stderr, "width and height must be at least 16")
		os.Exit(2)
	}

	report, err := RunPipeline(*out, *width, *height, *probeRays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ray diagnostics failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ray diagnostics written to %s\n", *out)
	for _, suite := range report.PixelSuites {
		fmt.Printf("  %-16s pixels=%d mismatch=%d missed=%d false=%d wrong=%d count=%d maxErr=%.3g pass=%t\n",
			suite.Name, suite.Pixels, suite.MismatchPixels, suite.MissedPixels,
			suite.FalseHitPixels, suite.WrongTrianglePixels,
			suite.CountMismatchPixels, suite.MaxColorError, suite.Pass)
	}
	for _, rays := range report.RaySuites {
		fmt.Printf("  %-16s probes=%d failed=%d pass=%t\n", rays.Group, rays.Rays, len(rays.Mismatches), rays.Pass)
	}

	if !report.Pass {
		fmt.Fprintln(os.Stderr, "strict reference comparison failed")
		os.Exit(1)
	}
}
