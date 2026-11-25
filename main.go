package main

import (
	"flag"
	"fmt"
	"os"

	"atlaspad/internal/demo"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "demo" {
		args = args[1:]
	}

	fs := flag.NewFlagSet("atlaspad", flag.ContinueOnError)
	out := fs.String("out", "artifacts/generated", "directory for PNG, JSON, and Markdown outputs")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	result, err := demo.Run(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atlaspad: %v\n", err)
		if result == nil {
			os.Exit(1)
		}
	}

	fmt.Printf("atlaspad demo outputs: %s\n", *out)
	fmt.Printf("tight violations: %d\n", result.TightViolations)
	fmt.Printf("safe violations:  %d (max distance %.2f)\n", result.SafeViolations, result.MaxDistanceSafe)
	fmt.Printf("texture utilization: %.2f%%\n", result.Utilization.TextureUtilization*100)
	fmt.Printf("cell utilization:    %.2f%%\n", result.Utilization.CellUtilization*100)

	if err != nil {
		os.Exit(1)
	}
}
