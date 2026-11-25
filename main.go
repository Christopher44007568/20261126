package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"aelab/internal/export"
	"aelab/internal/pipeline"
	"aelab/internal/scene"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for frames, CSV curves and report")
	quality := flag.Int("jpeg-quality", 72, "thumbnail JPEG quality")
	flag.Parse()

	if err := run(*outDir, *quality); err != nil {
		fmt.Fprintln(os.Stderr, "aelab:", err)
		os.Exit(1)
	}
}

func run(outDir string, quality int) error {
	framesRoot := filepath.Join(outDir, "frames")
	if err := os.MkdirAll(framesRoot, 0o755); err != nil {
		return err
	}

	world := scene.RenderWorld()

	fmt.Println("Rendering 10 s scene at", scene.FPS, "fps with three AE responses...")
	result, err := pipeline.Run(world, func(profileID string, frameNumber int, img *image.RGBA) error {
		path := filepath.Join(framesRoot, profileID, fmt.Sprintf("frame_%04d.jpg", frameNumber))
		return export.SaveJPEG(path, img, quality)
	})
	if err != nil {
		return err
	}

	if err := writeOutputs(outDir, result); err != nil {
		return err
	}
	fmt.Println("Done. Open", filepath.Join(outDir, "report.html"))
	return nil
}

func writeOutputs(outDir string, result *pipeline.Result) error {
	if err := export.WriteFrameCSV(filepath.Join(outDir, "exposure_values.csv"), result); err != nil {
		return err
	}
	if err := export.WriteOvershootCSV(filepath.Join(outDir, "overshoot_curves.csv"), result); err != nil {
		return err
	}
	if err := export.WriteMetricsCSV(filepath.Join(outDir, "ae_metrics.csv"), result); err != nil {
		return err
	}
	if err := export.WriteExposureSVG(filepath.Join(outDir, "exposure_curves.svg"), result); err != nil {
		return err
	}
	if err := export.WriteExposurePNG(filepath.Join(outDir, "exposure_curves.png"), result); err != nil {
		return err
	}
	if err := export.WriteOvershootSVG(filepath.Join(outDir, "overshoot_curves.svg"), result); err != nil {
		return err
	}
	if err := export.WriteOvershootPNG(filepath.Join(outDir, "overshoot_curves.png"), result); err != nil {
		return err
	}

	for _, profile := range result.Profiles {
		allPaths := make([]string, 0, scene.FrameCount)
		for frame := 1; frame <= scene.FrameCount; frame++ {
			allPaths = append(allPaths, filepath.Join(outDir, "frames", profile.Profile.ID, fmt.Sprintf("frame_%04d.jpg", frame)))
		}
		sampledPaths := make([]string, 0, 30)
		for i := 0; i < len(allPaths); i += 10 {
			sampledPaths = append(sampledPaths, allPaths[i])
		}
		sheetPath := filepath.Join(outDir, "contact_sheet_"+profile.Profile.ID+".jpg")
		if err := export.ContactSheetFromFiles(sheetPath, sampledPaths, 10, 240, 135); err != nil {
			return err
		}
		allSheetPath := filepath.Join(outDir, "all_frames_sheet_"+profile.Profile.ID+".jpg")
		if err := export.ContactSheetFromFiles(allSheetPath, allPaths, 15, 240, 135); err != nil {
			return err
		}
	}
	return export.WriteReport(filepath.Join(outDir, "report.html"), result)
}
