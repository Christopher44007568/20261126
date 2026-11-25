package main

import (
	"fmt"
	"image"
	"math"
	"path/filepath"
)

type parameterEffect struct {
	Name     string  `json:"name"`
	File     string  `json:"file"`
	RMSE     float64 `json:"rmse_px"`
	MaxError float64 `json:"max_error_px"`
	ScalePx  float64 `json:"heatmap_scale_px"`
	Rejected bool    `json:"rejected_as_update"`
	Reason   string  `json:"reason,omitempty"`
}

func buildEdgeDiagnostic(cam Camera, pose Pose, spec CheckerSpec, seed int64) (*image.RGBA, Detection) {
	img := RenderCheckerboard(cam, pose, spec, RenderOptions{
		Width: cam.Width, Height: cam.Height, SSAA: 2, Noise: 1.2, Seed: seed,
	})
	OccludeRightEdge(img, cam, pose, spec)
	d := DetectCheckerboard(img, spec)
	return img, d
}

func runParameterDiagnostics(outDir string, keeper *Keeper, good *CalibrationResult, src *image.RGBA,
	spec CheckerSpec, object []Vec3, observed []Vec2) error {
	wrongDist := good.Camera
	wrongDist.K1, wrongDist.K2, wrongDist.K3 = 0, 0, 0
	wrongDist.P1, wrongDist.P2 = 0, 0

	wrongFocal := good.Camera
	wrongFocal.Fx *= 1.16
	wrongFocal.Fy *= 1.16

	wrongPrincipal := good.Camera
	wrongPrincipal.Cx += 46
	wrongPrincipal.Cy -= 38

	effects := []struct {
		name  string
		file  string
		cam   Camera
		scale float64
	}{
		{"distortion_zeroed", "reprojection_distortion_error.png", wrongDist, 10},
		{"focal_length_16pct_too_large", "reprojection_focal_error.png", wrongFocal, 48},
		{"principal_point_shifted", "reprojection_principal_error.png", wrongPrincipal, 48},
	}
	var report []parameterEffect
	for _, e := range effects {
		residuals := ReprojectionResiduals(e.cam, []Pose{good.Poses[0]}, [][]Vec3{object}, [][]Vec2{observed})
		grid := NewErrorGrid(spec, residuals, e.scale)
		img := RenderResidualHeatmap(src, grid)
		path := filepath.Join(outDir, e.file)
		if err := SavePNG(path, img); err != nil {
			return err
		}
		candidate := CalibrationResult{
			Camera: e.cam, Poses: good.Poses, RMSE: grid.RMSE, Points: len(residuals),
			Iterations: good.Iterations,
		}
		_, pubErr := keeper.Publish(candidate, nil)
		pe := parameterEffect{
			Name: e.name, File: e.file, RMSE: grid.RMSE, MaxError: grid.MaxErr,
			ScalePx: e.scale, Rejected: pubErr != nil,
		}
		if pubErr != nil {
			pe.Reason = pubErr.Error()
		}
		if pubErr == nil {
			return fmt.Errorf("unsafe %s parameter update was accepted", e.name)
		}
		report = append(report, pe)
		fmt.Printf("%s: rmse=%.2f max=%.2f px, rejected and last good retained\n", e.name, grid.RMSE, grid.MaxErr)
	}
	return SaveJSON(filepath.Join(outDir, "parameter_effects.json"), report)
}

func runBadParameterFallback(keeper *Keeper, good CalibrationResult) {
	badFocal := good
	badFocal.Camera.Fx = -880
	badFocal.RMSE = 999
	_, _ = keeper.Publish(badFocal, nil)

	badNaN := good
	badNaN.Camera.Cx = math.NaN()
	badNaN.RMSE = math.Inf(1)
	_, _ = keeper.Publish(badNaN, nil)
}
