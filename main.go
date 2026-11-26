package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"texaudit/internal/audit"
)

func main() {
	manifestPath := flag.String("manifest", "", "path to a JSON scene manifest")
	outputDir := flag.String("out", "audit-output", "directory for rendered audit outputs")
	generateDemo := flag.Bool("demo", false, "generate the bundled linear-albedo demo scene before auditing")
	errorGain := flag.Float64("error-gain", 2, "amplification factor for the local-error heat map")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  texaudit -manifest scene.json -out audit-output\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  texaudit -demo -out audit-output\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *generateDemo && *manifestPath != "" {
		exitWithError("choose either -manifest or -demo, not both")
	}
	if !*generateDemo && *manifestPath == "" {
		flag.Usage()
		exitWithError("-manifest is required (or use -demo)")
	}
	if *errorGain <= 0 {
		exitWithError("-error-gain must be positive")
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		exitWithError(fmt.Sprintf("create output directory: %v", err))
	}
	if *generateDemo {
		generatedManifest, err := audit.WriteDemoAssets(*outputDir)
		if err != nil {
			exitWithError(err.Error())
		}
		*manifestPath = generatedManifest
	}

	report, err := audit.RunAudit(*manifestPath, *outputDir, *errorGain)
	if err != nil {
		exitWithError(err.Error())
	}

	fmt.Printf("correct render:     %s\n", relativeOrAbsolute(report.Outputs.Correct))
	fmt.Printf("sRGB misread:       %s\n", relativeOrAbsolute(report.Outputs.Misread))
	fmt.Printf("comparison:         %s\n", relativeOrAbsolute(report.Outputs.Comparison))
	fmt.Printf("local error map:    %s\n", relativeOrAbsolute(report.Outputs.ErrorMap))
	fmt.Printf("report:             %s\n", relativeOrAbsolute(report.Outputs.Report))
	fmt.Printf("mean |delta|:       %.6f\n", report.Metrics.Mean)
	fmt.Printf("p99 |delta|:        %.6f\n", report.Metrics.P99)
	fmt.Printf("max |delta|:        %.6f\n", report.Metrics.Max)
	fmt.Printf("data channels:      normal=linear roughness=linear (unchanged in misread)\n")
}

func relativeOrAbsolute(path string) string {
	relative, err := filepath.Rel(".", path)
	if err == nil && len(relative) < len(path) {
		return relative
	}
	return path
}

func exitWithError(message string) {
	fmt.Fprintln(os.Stderr, "texaudit:", message)
	os.Exit(1)
}
