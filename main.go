package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
)

func main() {
	size := flag.Int("size", 512, "square preview size in pixels")
	bits := flag.Int("bits", 8, "bits used for each of the two stored channels")
	outputDir := flag.String("out", "artifacts", "output directory")
	flag.Parse()

	if *size < 16 || *size > 4096 {
		fmt.Fprintln(os.Stderr, "size must be between 16 and 4096")
		os.Exit(2)
	}
	if *bits < 1 || *bits > 16 {
		fmt.Fprintln(os.Stderr, "bits must be between 1 and 16")
		os.Exit(2)
	}

	samples, metrics := BuildSphere(*size, *bits)
	outputs := []struct {
		name string
		img  image.Image
	}{
		{"original_normal_sphere.png", renderNormalSphere(samples, false)},
		{"compressed_normal_sphere.png", renderNormalSphere(samples, true)},
		{"angular_error_heatmap.png", renderErrorHeatmap(samples, metrics.MaxRadians)},
		{"lighting_comparison.png", renderLightingComparison(samples)},
		{"packed_octahedral_rg.png", renderPackedChannels(samples, *bits)},
	}

	for _, output := range outputs {
		path := filepath.Join(*outputDir, output.name)
		if err := writePNG(path, output.img); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	fmt.Printf("samples=%d bits=%d/channels=2\n", metrics.Samples, *bits)
	fmt.Printf("mean_angular_error_deg=%.5f\n", metrics.MeanRadians*180/math.Pi)
	fmt.Printf("max_angular_error_deg=%.5f\n", metrics.MaxRadians*180/math.Pi)
	worst := metrics.MaxSample
	fmt.Printf("worst_original=(%.6f, %.6f, %.6f) compressed=(%.6f, %.6f, %.6f) packed=(%d,%d)\n",
		worst.Original.X, worst.Original.Y, worst.Original.Z,
		worst.Compressed.X, worst.Compressed.Y, worst.Compressed.Z,
		worst.Packed.U, worst.Packed.V)
}
