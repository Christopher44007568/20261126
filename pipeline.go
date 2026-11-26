package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func run(outDir string) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		absOut = outDir
	}

	stage := BuildCabin()
	renderer := NewRenderer()
	report := &Report{
		Schema:          "variant-tree/v1",
		Asset:           stage.Name,
		VariantSet:      stage.VariantSet,
		OutputDirectory: absOut,
		GoVersion:       runtime.Version(),
		Pass:            true,
	}

	addCheck := func(name string, pass bool, detail string) {
		report.Checks = append(report.Checks, CheckReport{Name: name, Pass: pass, Detail: detail})
		if !pass {
			report.Pass = false
		}
	}

	summer, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "variant_summer.ppm", summer); err != nil {
		return report, err
	}

	if err := stage.SelectVariant(AssetPath, VariantWinter); err != nil {
		return report, err
	}
	winter, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "variant_winter.ppm", winter); err != nil {
		return report, err
	}

	if err := stage.SelectVariant(AssetPath, VariantNight); err != nil {
		return report, err
	}
	night, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "variant_night.ppm", night); err != nil {
		return report, err
	}

	hashesUnique := summer.PixelSHA256() != winter.PixelSHA256() &&
		winter.PixelSHA256() != night.PixelSHA256() &&
		summer.PixelSHA256() != night.PixelSHA256()
	addCheck("three variants render differently", hashesUnique,
		"summer/winter/night pixel hashes are pairwise distinct")

	sheet := BuildContactSheet(map[Variant]*Image{
		VariantSummer: summer,
		VariantWinter: winter,
		VariantNight:  night,
	})
	if err := report.writePPM(outDir, "variant_contact_sheet.ppm", sheet); err != nil {
		return report, err
	}

	if err := stage.SelectVariant(AssetPath, PropagationVariant); err != nil {
		return report, err
	}
	token, err := stage.Override(WindowBankPath, OverrideVariant)
	if err != nil {
		return report, err
	}
	propagation, err := stage.Resolve(LeftWindowPath)
	if err != nil {
		return report, err
	}
	propImage, err := renderer.Render(stage, RenderOptions{HighlightPath: LeftWindowPath})
	if err != nil {
		return report, err
	}
	AddPathCaption(propImage, propagation)
	if err := report.writePPM(outDir, "selection_path.ppm", propImage); err != nil {
		return report, err
	}

	leftResolution, _ := stage.Resolve(LeftWindowPath)
	rightResolution, _ := stage.Resolve(RightWindowPath)
	roofResolution, _ := stage.Resolve(RoofPath)
	addCheck("child override propagates to descendants",
		leftResolution.Variant == VariantNight && leftResolution.SourcePath == WindowBankPath &&
			rightResolution.Variant == VariantNight && rightResolution.SourcePath == WindowBankPath,
		"WindowBank=night causes Left/Right to resolve night from WindowBank")
	addCheck("parent selection propagates to siblings",
		roofResolution.Variant == VariantWinter && roofResolution.SourcePath == AssetPath,
		"Roof remains winter inherited from /Stage/Asset")
	leftBinding := stage.MustNode(LeftWindowPath).Bindings["surface"]
	addCheck("variant switch binds real material",
		leftBinding.Resolved() && leftBinding.MaterialID == "window.night" && leftBinding.Shader == "lit_window",
		fmt.Sprintf("Left surface -> %s (%s)", leftBinding.MaterialID, leftBinding.Shader))

	if err := stage.Restore(token); err != nil {
		return report, err
	}
	winterRestored, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "override_restored_winter.ppm", winterRestored); err != nil {
		return report, err
	}
	winterCompare := compareImages(winter, winterRestored)
	report.Restore.OverrideWinter = winterCompare
	addCheck("override restore matches inherited winter pixels", winterCompare.Identical,
		fmt.Sprintf("differentPixels=%d hash=%s", winterCompare.DifferentPixels, winterCompare.PixelHashBefore[:16]))

	if err := stage.SelectVariant(AssetPath, InitialVariant); err != nil {
		return report, err
	}
	current, err := stage.Resolve(LeftWindowPath)
	if err != nil {
		return report, err
	}
	summerRestored, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "restored_summer.ppm", summerRestored); err != nil {
		return report, err
	}
	currentPathImage, err := renderer.Render(stage, RenderOptions{HighlightPath: LeftWindowPath})
	if err != nil {
		return report, err
	}
	AddCurrentPathCaption(currentPathImage, current)
	if err := report.writePPM(outDir, "current_selection_path.ppm", currentPathImage); err != nil {
		return report, err
	}
	summerCompare := compareImages(summer, summerRestored)
	report.Restore.OriginalSummer = summerCompare
	addCheck("switch back to original restores every pixel", summerCompare.Identical,
		fmt.Sprintf("differentPixels=%d before=%s after=%s", summerCompare.DifferentPixels,
			summerCompare.PixelHashBefore[:16], summerCompare.PixelHashAfter[:16]))

	if err := stage.ChangeMaterialNameOnly(RoofPath, "surface", "roof.winter"); err != nil {
		return report, err
	}
	_, nameOnlyErr := renderer.Render(stage, RenderOptions{})
	addCheck("name-only material edit is rejected", nameOnlyErr != nil,
		"renderer refuses stale/nil Material pointer until Rebind")

	if err := stage.Rebind(); err != nil {
		return report, err
	}
	reboundImage, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		return report, err
	}
	if err := report.writePPM(outDir, "rebind_repaired_summer.ppm", reboundImage); err != nil {
		return report, err
	}
	rebindCompare := compareImages(summer, reboundImage)
	report.Restore.RebindRepair = rebindCompare
	addCheck("rebind after name-only break restores pixels", rebindCompare.Identical,
		fmt.Sprintf("differentPixels=%d", rebindCompare.DifferentPixels))

	report.CurrentPath = current
	report.PropagationPath = propagation
	report.Selections = stage.SnapshotSelections()
	report.Bindings = bindingReports(stage)

	pathText := selectionPathText(current, propagation, stage, report.Restore)
	textReport, err := writeText(outDir, "current_selection_path.txt", pathText)
	if err != nil {
		return report, err
	}
	report.Outputs = append(report.Outputs, textReport)

	pathDocument := map[string]any{
		"currentSelectionPath": report.CurrentPath,
		"propagationDemoPath":  report.PropagationPath,
		"authoredSelections":   report.Selections,
		"restore":              report.Restore,
	}
	if err := report.writeJSON(outDir, "current_selection_path.json", pathDocument); err != nil {
		return report, err
	}
	if err := report.writeJSON(outDir, "variant_report.json", report); err != nil {
		return report, err
	}

	if !report.Pass {
		return report, fmt.Errorf("variant verification failed")
	}
	return report, nil
}
