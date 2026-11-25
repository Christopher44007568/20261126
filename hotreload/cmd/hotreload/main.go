package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"hotreload"
)

type scenario struct {
	name string
	run  func() (*hotreload.Frame, *hotreload.Frame, *hotreload.Frame, hotreload.ReloadReport)
}

func main() {
	outDir := "output"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	scenarios := []scenario{
		{
			name: "rgb",
			run: func() (*hotreload.Frame, *hotreload.Frame, *hotreload.Frame, hotreload.ReloadReport) {
				return hotreload.NewEngine().ReloadRGB(hotreload.ReloadedRGBPayload())
			},
		},
		{
			name: "geometry",
			run: func() (*hotreload.Frame, *hotreload.Frame, *hotreload.Frame, hotreload.ReloadReport) {
				return hotreload.NewEngine().ReplaceGeometry(hotreload.ReplacementGeometry(), true)
			},
		},
		{
			name: "material_detached",
			run: func() (*hotreload.Frame, *hotreload.Frame, *hotreload.Frame, hotreload.ReloadReport) {
				return hotreload.NewEngine().ReplaceGeometry(hotreload.ReplacementGeometry(), false)
			},
		},
		{
			name: "rgb_error",
			run: func() (*hotreload.Frame, *hotreload.Frame, *hotreload.Frame, hotreload.ReloadReport) {
				return hotreload.NewEngine().ReloadRGB(hotreload.BrokenRGBPayload())
			},
		},
	}

	summary := map[string]hotreload.ReloadReport{}
	for _, sc := range scenarios {
		before, after, diff, report := sc.run()
		summary[sc.name] = report
		writePNG(filepath.Join(outDir, sc.name+"_before.png"), before)
		writePNG(filepath.Join(outDir, sc.name+"_after.png"), after)
		writePNG(filepath.Join(outDir, sc.name+"_diff.png"), diff)
		writeJSON(filepath.Join(outDir, sc.name+"_report.json"), report)
		fmt.Printf("%-18s ok=%-5v recomputed=%v retained=%v changed=%dpx (%.2f%%)\n",
			sc.name, report.OK, report.RecomputedNodes, report.CacheRetained,
			report.Diff.ChangedPixels, report.Diff.Percent)
	}
	writeJSON(filepath.Join(outDir, "reports.json"), summary)
	fmt.Println("wrote outputs to", outDir)
}

func writePNG(path string, f *hotreload.Frame) {
	file, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	if err := f.EncodePNG(file); err != nil {
		fatal(err)
	}
}

func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "hotreload:", err)
	os.Exit(1)
}
