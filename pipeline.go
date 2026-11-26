package main

import (
	"image"
	"image/color"
	"path/filepath"
)

type PipelineReport struct {
	Pass                 bool              `json:"pass"`
	Width                int               `json:"width"`
	Height               int               `json:"height"`
	ProbeRays            int               `json:"probe_rays_per_scene"`
	PixelSuites          []SuiteReport     `json:"pixel_suites"`
	RaySuites            []RayProbeReport  `json:"ray_suites"`
	FaultControl         SuiteReport       `json:"fault_control"`
	FaultControlDetected bool              `json:"fault_control_detected"`
	Outputs              map[string]string `json:"outputs"`
}

func RunPipeline(out string, width, height, randomProbeCount int) (PipelineReport, error) {
	scenes := AllScenes(width, height)
	report := PipelineReport{
		Pass:      true,
		Width:     width,
		Height:    height,
		ProbeRays: randomProbeCount,
		Outputs:   map[string]string{},
	}

	var references, simplified, counts, testCounts, heatmaps, masks, comparison []image.Image

	for _, scene := range scenes {
		accel := NewSimplifiedAccel(scene.Triangles)
		rendered := RenderSuite(scene, accel)
		probes := DefaultProbeSuite(scene, randomProbeCount)
		rayReport := ValidateRays(scene, accel, probes)

		report.PixelSuites = append(report.PixelSuites, rendered.Report)
		report.RaySuites = append(report.RaySuites, rayReport)
		report.Pass = report.Pass && rendered.Report.Pass && rayReport.Pass

		suiteDir := filepath.Join(out, "suites", scene.Name)
		outputs := []struct {
			name string
			img  *image.RGBA
		}{
			{"reference.png", rendered.Reference},
			{"simplified.png", rendered.Simplified},
			{"intersection_count.png", rendered.Intersections},
			{"traversal_tests.png", rendered.TestCounts},
			{"error_heatmap.png", rendered.Heatmap},
			{"error_mask.png", rendered.Mask},
		}
		for _, output := range outputs {
			path := filepath.Join(suiteDir, output.name)
			if err := SavePNG(path, output.img); err != nil {
				return report, err
			}
		}

		row := HorizontalSheet([]image.Image{
			rendered.Reference, rendered.Simplified, rendered.Intersections, rendered.Heatmap,
		}, 5, borderForSuite(scene.Name))
		if err := SavePNG(filepath.Join(suiteDir, "side_by_side.png"), row); err != nil {
			return report, err
		}

		references = append(references, rendered.Reference)
		simplified = append(simplified, rendered.Simplified)
		counts = append(counts, rendered.Intersections)
		testCounts = append(testCounts, rendered.TestCounts)
		heatmaps = append(heatmaps, rendered.Heatmap)
		masks = append(masks, rendered.Mask)
		comparison = append(comparison,
			rendered.Reference, rendered.Simplified, rendered.Intersections, rendered.Heatmap)
	}

	faultScene := standardScene("fault_injection", width, height)
	faultID := faultScene.AddTriangle("deliberately-omitted",
		V(-1.7, 0.35, 0.1), V(1.7, 0.35, 0.1), V(0.0, 3.25, -0.15),
		V(1.0, 0.34, 0.16))
	faultTracer := OmitTracer{Scene: &faultScene, Omitted: map[int]bool{faultID: true}}
	faultResult := RenderSuite(&faultScene, faultTracer)
	report.FaultControl = faultResult.Report
	report.FaultControlDetected = !faultResult.Report.Pass && faultResult.Report.MissedPixels > 0
	if !report.FaultControlDetected {
		report.Pass = false
	}

	faultDir := filepath.Join(out, "fault_injection")
	faultOutputs := []struct {
		name string
		img  *image.RGBA
	}{
		{"reference.png", faultResult.Reference},
		{"faulty_simplified.png", faultResult.Simplified},
		{"intersection_count.png", faultResult.Intersections},
		{"error_heatmap.png", faultResult.Heatmap},
		{"error_mask.png", faultResult.Mask},
	}
	for _, output := range faultOutputs {
		if err := SavePNG(filepath.Join(faultDir, output.name), output.img); err != nil {
			return report, err
		}
	}
	faultSheet := HorizontalSheet([]image.Image{
		faultResult.Reference, faultResult.Simplified,
		faultResult.Intersections, faultResult.Heatmap,
	}, 5, color.RGBA{R: 255, G: 70, B: 90, A: 255})
	if err := SavePNG(filepath.Join(faultDir, "side_by_side.png"), faultSheet); err != nil {
		return report, err
	}

	rootOutputs := []struct {
		name string
		img  *image.RGBA
	}{
		{"reference.png", GridSheet(references, 1, 6)},
		{"simplified.png", GridSheet(simplified, 1, 6)},
		{"intersection_count.png", GridSheet(counts, 1, 6)},
		{"traversal_tests.png", GridSheet(testCounts, 1, 6)},
		{"error_heatmap.png", GridSheet(heatmaps, 1, 6)},
		{"error_mask.png", GridSheet(masks, 1, 6)},
		{"side_by_side.png", GridSheet(comparison, 4, 6)},
	}
	for _, output := range rootOutputs {
		path := filepath.Join(out, output.name)
		if err := SavePNG(path, output.img); err != nil {
			return report, err
		}
		report.Outputs[output.name] = path
	}

	report.Outputs["report.json"] = filepath.Join(out, "report.json")
	report.Outputs["index.html"] = filepath.Join(out, "index.html")
	if err := SaveJSON(report.Outputs["report.json"], report); err != nil {
		return report, err
	}
	if err := WriteHTMLReport(report.Outputs["index.html"], report); err != nil {
		return report, err
	}
	return report, nil
}

func borderForSuite(name string) color.RGBA {
	switch name {
	case "grazing":
		return color.RGBA{R: 255, G: 176, B: 64, A: 255}
	case "parallel":
		return color.RGBA{R: 64, G: 210, B: 255, A: 255}
	default:
		return color.RGBA{R: 150, G: 255, B: 130, A: 255}
	}
}
