package main

import (
	"flag"
	"fmt"
)

func main() {
	outputDir := flag.String("out", "output", "directory for PPM images, charts, CSV, and summary")
	flag.Parse()

	if err := run(*outputDir); err != nil {
		panic(err)
	}
	fmt.Printf("AWB comparison written to %s\n", *outputDir)
}

func run(outputDir string) error {
	scene, neutralMask := buildScene()
	lights := lightSequence()
	frames := make([]*Image, len(lights))
	for i, light := range lights {
		frames[i] = renderFrame(scene, light.RGB)
	}

	report := analyzeFrames(frames, lights, neutralMask)
	if err := osMkdirAll(outputDir); err != nil {
		return err
	}
	if err := writeImagePPMs(outputDir, frames, report); err != nil {
		return err
	}
	if err := writeCharts(outputDir, report); err != nil {
		return err
	}
	return writeReports(outputDir, report)
}
