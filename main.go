// Command nsmigrate demonstrates namespace migration for a small USD-like
// scene: it proves references, material bindings and camera targets survive a
// path/parent-child-aware re-rooting, renders before/after PPM posters with
// red annotation boxes, and quotes concrete old paths for the string-only
// failure mode. Only the Go standard library is used.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"nsmigrate/audit"
	"nsmigrate/migrate"
	"nsmigrate/report"
	"nsmigrate/scene"
	"nsmigrate/viz"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := run(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type rendered struct {
	base   string
	poster *viz.Poster
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	m := scene.FixtureMapping()
	beforeStage := scene.BuildDemoStage()
	beforeReport := audit.Audit("before", beforeStage, m)

	correctStage, correctMig := migrate.MigrateCorrect(beforeStage, m)
	correctReport := audit.Audit("correct", correctStage, m)

	naiveStage := migrate.MigrateStringOnly(beforeStage, m)
	naiveReport := audit.Audit("string-only", naiveStage, m)

	// The demo's contract: before and correct-migration states match their
	// expected edge cases and preserve every in-scope arc; the naive edit must
	// provably break so its concrete-old-path diagnostic is meaningful.
	if !beforeReport.InvariantOK {
		return fmt.Errorf("before fixture does not match expectations")
	}
	if !correctReport.InvariantOK {
		return fmt.Errorf("path-aware migration broke a reference/material/camera link")
	}
	if naiveReport.InvariantOK {
		return fmt.Errorf("string-only demo unexpectedly preserved all links")
	}
	printDiagnostics(correctReport, naiveReport)

	specs := []struct {
		base string
		opt  viz.Options
	}{
		{"migration_before", viz.Options{
			ImageName: "migration_before.ppm", Mode: "before",
			Title:    "BEFORE MIGRATION - OLD NAMESPACE",
			Subtitle: "reference / material / camera arcs audited in the original layout",
			Stage:    beforeStage, Report: beforeReport, Mapping: m,
		}},
		{"migration_after", viz.Options{
			ImageName: "migration_after.ppm", Mode: "correct",
			Title:    "AFTER PATH-AWARE MIGRATION - LINKS PRESERVED",
			Subtitle: "parent/child paths re-rooted with component-aware prefix rewrite",
			Stage:    correctStage, Report: correctReport, Mapping: m,
		}},
		{"naive_string_replace_after", viz.Options{
			ImageName: "naive_string_replace_after.ppm", Mode: "string-only",
			Title:    "STRING-ONLY EDIT - PARENT/CHILD PATHS NOT UPDATED",
			Subtitle: "literal ReplaceAll moved arc strings; descendants stay on the old path",
			Stage:    naiveStage, Report: naiveReport, Mapping: m,
		}},
	}

	var renderedImages []report.ImageBoxes
	for _, s := range specs {
		poster := viz.Render(s.opt)
		ppmPath := filepath.Join(outDir, s.base+".ppm")
		pngPath := filepath.Join(outDir, s.base+".png")
		if err := poster.Canvas.WritePPMFile(ppmPath); err != nil {
			return err
		}
		if err := poster.Canvas.WritePNGFile(pngPath); err != nil {
			return err
		}
		renderedImages = append(renderedImages, report.ImageBoxes{
			Image: s.opt.ImageName, Width: poster.Width, Height: poster.Height, Boxes: poster.Boxes,
		})
		fmt.Printf("wrote %s (+ .png): %dx%d, %d red boxes\n", ppmPath, poster.Width, poster.Height, len(poster.Boxes))
	}

	boxesManifest := report.NewRedBoxesManifest(renderedImages)
	if err := report.WriteJSON(filepath.Join(outDir, "red_boxes.json"), boxesManifest); err != nil {
		return err
	}

	combined := struct {
		Mapping          scene.Mapping         `json:"mapping"`
		CorrectMigration migrate.CorrectReport `json:"correct_migration"`
		Before           *audit.Report         `json:"before"`
		Correct          *audit.Report         `json:"correct"`
		StringOnly       *audit.Report         `json:"string_only"`
		Artifacts        map[string]string     `json:"artifacts"`
	}{
		Mapping: m, CorrectMigration: correctMig,
		Before: beforeReport, Correct: correctReport, StringOnly: naiveReport,
		Artifacts: map[string]string{
			"before_ppm":      "artifacts/migration_before.ppm",
			"after_ppm":       "artifacts/migration_after.ppm",
			"naive_ppm":       "artifacts/naive_string_replace_after.ppm",
			"red_boxes":       "artifacts/red_boxes.json",
			"markdown_report": "artifacts/migration_report.md",
			"json_report":     "artifacts/migration_report.json",
		},
	}
	if err := report.WriteJSON(filepath.Join(outDir, "migration_report.json"), combined); err != nil {
		return err
	}
	md := report.BuildMarkdown(m, beforeReport, correctReport, naiveReport)
	if err := report.WriteFile(filepath.Join(outDir, "migration_report.md"), []byte(md)); err != nil {
		return err
	}

	fmt.Println("invariants: before=PASS correct=PASS string-only=FAIL (expected anti-example)")
	fmt.Println("reports: artifacts/migration_report.{json,md}, artifacts/red_boxes.json")
	return nil
}

func printDiagnostics(correct, naive *audit.Report) {
	fmt.Println("\n[correct migration] links that still resolve under the new path:")
	for _, f := range correct.Findings {
		if f.Status == audit.StatusOK {
			fmt.Printf("  OK   %-2s %-18s %s\n", f.LinkID, f.Kind, f.Target)
		}
	}

	fmt.Println("\n[string-only edit] dangling arcs - concrete OLD paths:")
	for _, f := range naive.Findings {
		fmt.Printf("  %-8s %-2s %-18s old_path=%s\n",
			f.Status, f.LinkID, f.Kind, f.OldPath)
		if f.StalePath != "" {
			fmt.Printf("                    stale parent/child path still present: %s\n", f.StalePath)
		}
	}
	fmt.Println()
}
