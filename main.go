package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	manifestPath := flag.String("manifest", "", "path to the patch manifest JSON")
	sourcePath := flag.String("source", "", "optional original image (PNG/JPEG/GIF)")
	targetPath := flag.String("target", "", "optional target image (PNG/JPEG/GIF)")
	outPath := flag.String("out", "comparison.png", "comparison chart output path")
	reportPath := flag.String("report", "", "report JSON path (default: <out>.report.json)")
	failOnError := flag.Bool("fail-on-error", false, "exit with status 2 when local checks fail")
	flag.Parse()

	if *manifestPath == "" {
		flag.Usage()
		fatalf("-manifest is required")
	}
	m, err := loadManifest(*manifestPath)
	if err != nil {
		fatalf("%v", err)
	}

	source, target, err := loadFrames(m, *sourcePath, *targetPath)
	if err != nil {
		fatalf("%v", err)
	}
	result := analyze(m, source, target)
	chart := renderChart(m, result, m.Output.Amplification)

	if err := writePNG(*outPath, chart); err != nil {
		fatalf("write comparison: %v", err)
	}
	if *reportPath == "" {
		*reportPath = defaultReportPath(*outPath)
	}
	report := buildReport(m, result, *outPath, m.Output.Amplification)
	if err := writeReport(*reportPath, report); err != nil {
		fatalf("write report: %v", err)
	}

	printSummary(result, *outPath, *reportPath)
	if *failOnError && !result.Summary.LocalPass {
		os.Exit(2)
	}
}

func loadFrames(m *Manifest, sourcePath, targetPath string) (image.Image, image.Image, error) {
	if (sourcePath == "") != (targetPath == "") {
		return nil, nil, errors.New("-source and -target must be supplied together")
	}
	if sourcePath == "" {
		source, err := renderSwatches(m, "source")
		if err != nil {
			return nil, nil, err
		}
		target, err := renderSwatches(m, "target")
		if err != nil {
			return nil, nil, err
		}
		return source, target, nil
	}
	source, err := loadImage(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load source: %w", err)
	}
	target, err := loadImage(targetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load target: %w", err)
	}
	if err := validateImageSize(source, m, "source"); err != nil {
		return nil, nil, err
	}
	if err := validateImageSize(target, m, "target"); err != nil {
		return nil, nil, err
	}
	return cropToCanvas(source, m), cropToCanvas(target, m), nil
}

func writePNG(path string, img image.Image) error {
	if err := ensureParent(path); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(f, img)
}

func ensureParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func defaultReportPath(out string) string {
	ext := filepath.Ext(out)
	if ext == "" {
		return strings.TrimRight(out, "./\\") + ".report.json"
	}
	return strings.TrimSuffix(out, ext) + ".report.json"
}

func printSummary(a *Analysis, comparison, report string) {
	fmt.Printf("status: %s\n", strings.ToUpper(a.Summary.Status))
	fmt.Printf("mean deltaE: %.2f (%d/%d patches pass)\n", a.Summary.MeanDeltaE, a.Summary.Passed, a.Summary.PatchCount)
	if a.Summary.AveragePass && !a.Summary.LocalPass {
		fmt.Println("average: pass; local verdict: FAIL (failure frame retained in comparison)")
	}
	for _, failure := range a.Summary.LocalFailures {
		fmt.Printf("local failure: %s\n", failure)
	}
	fmt.Printf("comparison: %s\n", comparison)
	fmt.Printf("report: %s\n", report)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "patchcheck: "+format+"\n", args...)
	os.Exit(1)
}
