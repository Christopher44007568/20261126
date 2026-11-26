package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for generated frames and report")
	flag.Parse()

	result, err := RunDemo(*outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "descriptorlab:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *outDir)
	fmt.Printf("  normal: camera v%d, texture v%d, descriptor v%d\n",
		result.Report.NormalBinding.Camera.Version,
		result.Report.NormalBinding.TextureVersion,
		result.Report.NormalBinding.DescriptorVersion)
	fmt.Printf("  stale:  camera v%d, texture v%d, descriptor v%d (%s)\n",
		result.Report.StaleBinding.Camera.Version,
		result.Report.StaleBinding.TextureVersion,
		result.Report.StaleBinding.DescriptorVersion,
		result.Report.StaleBinding.Invalidations)
	fmt.Printf("  projection delta: %.2f px; changed pixels: %.2f%%\n",
		result.Report.ProjectionDelta,
		result.Report.FrameDifference.Percent)
}
