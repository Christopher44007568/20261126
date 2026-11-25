package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"transientalias/internal/aliaslab"
)

func main() {
	width := flag.Int("w", 320, "image width")
	height := flag.Int("h", 180, "image height")
	outDir := flag.String("out", "artifacts", "artifact directory")
	flag.Parse()

	report, err := aliaslab.RunExperiment(*outDir, *width, *height)
	if report != nil {
		fmt.Printf("no_reuse:     %d slots, exact reference\n", report.NoReuse.SlotCount)
		fmt.Printf("valid_reuse:  %d slots, reuse=%d, exact=%t\n",
			report.ValidReuse.SlotCount, report.ValidReuse.ReuseCount,
			report.ValidReuse.ReferenceMatch)
		if report.ErrorReuse.Contamination != nil {
			fmt.Printf("error_reuse:  %d slots, violations=%d, first pollution (%d,%d)\n",
				report.ErrorReuse.SlotCount, len(report.ErrorReuse.Violations),
				report.ErrorReuse.Contamination.FirstX,
				report.ErrorReuse.Contamination.FirstY)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	for _, name := range []string{"no_reuse.ppm", "valid_reuse.ppm", "error_reuse.ppm", "result.json"} {
		fmt.Printf("wrote %s\n", filepath.Join(*outDir, name))
	}
}
