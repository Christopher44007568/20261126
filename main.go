package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory for PNG/JSON artifacts")
	width := flag.Int("width", 960, "native checkerboard image width")
	height := flag.Int("height", 720, "native checkerboard image height")
	frames := flag.Int("frames", 5, "number of calibration views")
	seed := flag.Int64("seed", 20261126, "deterministic synthetic seed")
	flag.Parse()

	if err := runDemo(*outDir, *width, *height, *frames, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runDemo(outDir string, width, height, frameCount int, seed int64) error {
	if width < 320 || height < 240 || frameCount < 2 {
		return fmt.Errorf("require width>=320 height>=240 frames>=2")
	}
	spec := DefaultSpec()
	cam := DefaultCamera()
	cam.Width, cam.Height = width, height
	if width != 960 || height != 720 {
		scale := float64(width) / 960
		cam.Fx *= scale
		cam.Fy *= scale
		cam.Cx = float64(width) / 2
		cam.Cy = float64(height) / 2
	}
	poses := DefaultPoses()
	if frameCount > len(poses) {
		frameCount = len(poses)
	}
	poses = poses[:frameCount]

	var images []*image.RGBA
	var detections []Detection
	var input CalibrationInput
	input.Width, input.Height = width, height
	for f, pose := range poses {
		img := RenderCheckerboard(cam, pose, spec, RenderOptions{
			Width: width, Height: height, SSAA: 2, Noise: 1.3,
			Seed: seed + int64(f*17),
		})
		d := DetectCheckerboard(img, spec)
		fmt.Printf("frame %d: detected %d/%d corners (rmse %.2f px)\n", f, d.FoundCount, spec.CornersX*spec.CornersY, d.RMSE)
		if !d.Complete {
			return fmt.Errorf("demo frame %d is incomplete; use edge diagnostic path for missed corners", f)
		}
		images = append(images, img)
		detections = append(detections, d)
		input.Frames = append(input.Frames, CalibFrame{Object: d.ObjectPoints(), Observed: d.Observed()})
		_ = SavePNG(filepath.Join(outDir, "frames", fmt.Sprintf("frame_%d_detection.png", f)), renderDetectionOnly(img, d))
	}

	fit, err := Calibrate(input)
	if err != nil {
		return fmt.Errorf("calibrate: %w", err)
	}
	fmt.Printf("calibration rmse=%.3f px fx=%.1f fy=%.1f cx=%.1f cy=%.1f k1=%.4f k2=%.4f\n",
		fit.RMSE, fit.Camera.Fx, fit.Camera.Fy, fit.Camera.Cx, fit.Camera.Cy, fit.Camera.K1, fit.Camera.K2)

	d0, img0 := detections[0], images[0]
	objects0, observed0 := d0.ObjectPoints(), d0.Observed()
	goodRes := ReprojectionResiduals(fit.Camera, []Pose{fit.Poses[0]}, [][]Vec3{objects0}, [][]Vec2{observed0})
	original := cloneRGBA(img0)
	DrawDetectionGrid(original, d0)
	DrawReprojectionErrors(original, goodRes)

	corrected := UndistortImage(img0, fit.Camera)
	DrawCorrectedGrid(corrected, d0, fit.Camera, fit, 0)
	rectified := RenderRectifiedGrid(img0, fit.Camera, fit.Poses[0], spec)

	goodGrid := NewErrorGrid(spec, goodRes, 2.5)
	heatmapImg := RenderResidualHeatmap(img0, goodGrid)

	edgeImg, edgeDetection := buildEdgeDiagnostic(cam, poses[0], spec, seed+999)
	edgeOverlay := cloneRGBA(edgeImg)
	DrawDetectionGrid(edgeOverlay, edgeDetection)

	keeper := NewKeeper(outDir, 2.0)
	if err := keeper.LoadLastGood(); err != nil {
		fmt.Println("ignoring unusable previous result:", err)
	}
	accepted, err := keeper.Publish(*fit, map[string]*image.RGBA{
		"original_grid.png":              original,
		"corrected_grid.png":             corrected,
		"rectified_metric_grid.png":      rectified,
		"residual_heatmap.png":           heatmapImg,
		"edge_missing_original_grid.png": edgeOverlay,
	})
	if err != nil {
		return fmt.Errorf("publish calibrated result: %w", err)
	}
	fmt.Println("published usable calibration generation", accepted.Generation)
	for name, img := range map[string]*image.RGBA{
		"original_grid.ppm":              original,
		"corrected_grid.ppm":             corrected,
		"rectified_metric_grid.ppm":      rectified,
		"residual_heatmap.ppm":           heatmapImg,
		"edge_missing_original_grid.ppm": edgeOverlay,
	} {
		if err := SavePPM(filepath.Join(outDir, name), img); err != nil {
			return err
		}
	}

	beforeHashes := keeper.hashCanonicalFiles()
	if err := runParameterDiagnostics(outDir, keeper, fit, img0, spec, objects0, observed0); err != nil {
		return err
	}
	runBadParameterFallback(keeper, *fit)
	afterHashes := keeper.hashCanonicalFiles()

	report := map[string]any{
		"ground_truth_camera": cam,
		"ground_truth_poses":  poses,
		"pattern":             spec,
		"detections":          detections,
		"edge_detection":      edgeDetection,
		"calibration":         fit,
		"frame0_rmse":         goodGrid.RMSE,
		"frame0_max_error":    goodGrid.MaxErr,
		"accepted_event":      accepted,
		"acceptance_rmse_px":  keeper.MaxRMSE,
		"resolution_policy": map[string]any{
			"downsampled":     false,
			"native_width":    width,
			"native_height":   height,
			"detection_scale": 1.0,
			"missing_corners": "rendered at native resolution with predicted markers",
		},
		"canonical_hashes_before_rejection": beforeHashes,
		"canonical_hashes_after_rejection":  afterHashes,
		"previous_result_retained":          equalStringSlices(beforeHashes, afterHashes),
	}
	if err := SaveJSON(filepath.Join(outDir, "calibration_report.json"), report); err != nil {
		return err
	}
	fmt.Println("wrote artifacts to", outDir)
	return nil
}

func renderDetectionOnly(src *image.RGBA, d Detection) *image.RGBA {
	img := cloneRGBA(src)
	DrawDetectionGrid(img, d)
	return img
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
