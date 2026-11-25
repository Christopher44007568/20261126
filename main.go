package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PPM frames and JSON/Markdown report")
	flag.Parse()

	artifacts, err := Generate(*outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "texture-stream: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("texture-stream: wrote %d PPM frames to %s\n", len(artifacts.Paths)-2, *outDir)
	for _, path := range artifacts.Paths {
		fmt.Println(path)
	}

	v := artifacts.Report.Verification
	fmt.Printf(
		"verify: frames=%d blank=%d bw_flash=%d geometry_stable=%t max_luma_jump=%.2f passed=%t\n",
		v.FramesChecked,
		v.BlankPixels,
		v.BlackWhiteFlashPixels,
		v.GeometryMaskStable,
		v.MaxMeanLuminanceJump,
		v.Passed,
	)
	if !v.Passed {
		os.Exit(2)
	}
}
