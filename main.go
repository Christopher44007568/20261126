package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	output := flag.String("out", "artifacts/render", "directory for PNG outputs")
	budget := flag.Int("budget", 0, "override hit budget; zero renders full and two-hit examples")
	flag.Parse()

	if err := run(*output, *budget); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(output string, budget int) error {
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	layers := DefaultScene()
	ray := Ray{
		Origin:    Vec3{X: -2.0, Y: 0, Z: 0},
		Direction: Vec3{X: 1, Y: 0, Z: 0},
	}
	tMin, tMax := 0.0, 4.5
	initial := whiteEnergy()

	if budget > 0 {
		result, err := TraceTransparent(ray, layers, initial, tMin, tMax, budget)
		if err != nil {
			return err
		}
		files, err := Render(result, layers, ray, tMin, tMax, fmt.Sprintf("budget_%d", budget), output)
		if err != nil {
			return err
		}
		printSummary(result, files)
		return nil
	}

	full, err := TraceTransparent(ray, layers, initial, tMin, tMax, len(layers))
	if err != nil {
		return err
	}
	fullFiles, err := Render(full, layers, ray, tMin, tMax, "full", filepath.Join(output))
	if err != nil {
		return err
	}
	printSummary(full, fullFiles)

	limited, err := TraceTransparent(ray, layers, initial, tMin, tMax, 2)
	if err != nil {
		return err
	}
	limitedFiles, err := Render(limited, layers, ray, tMin, tMax, "limited_budget", output)
	if err != nil {
		return err
	}
	printSummary(limited, limitedFiles)

	invalid := cloneLayers(layers)
	invalid[1].Transmittance.R = 1.25
	invalidResult, err := TraceTransparent(ray, invalid, initial, tMin, tMax, len(invalid))
	if err != nil {
		return err
	}
	invalidFiles, err := Render(invalidResult, invalid, ray, tMin, tMax, "invalid_energy", output)
	if err != nil {
		return err
	}
	printSummary(invalidResult, invalidFiles)
	return nil
}

func cloneLayers(layers []Layer) []Layer {
	copied := make([]Layer, len(layers))
	copy(copied, layers)
	return copied
}

func printSummary(result TraceResult, files []string) {
	fmt.Printf("hits=%d complete=%t energy=%s\n", len(result.Hits), result.Complete, shortRGB(result.Energy))
	if result.Unprocessed != nil {
		fmt.Printf("unprocessed=%s remaining=%v\n", result.Unprocessed.Reason, result.RemainingLayers)
	}
	for _, file := range files {
		fmt.Println(file)
	}
}
