package main

import "testing"

func demoPoses() []Pose {
	return []Pose{
		PoseFromRV(Vec3{X: 0.10, Y: -0.16, Z: 0.05}, Vec3{X: -245, Y: -160, Z: 790}),
		PoseFromRV(Vec3{X: -0.08, Y: 0.14, Z: -0.08}, Vec3{X: -235, Y: -150, Z: 835}),
		PoseFromRV(Vec3{X: 0.16, Y: 0.06, Z: 0.10}, Vec3{X: -255, Y: -168, Z: 850}),
		PoseFromRV(Vec3{X: -0.05, Y: -0.20, Z: -0.04}, Vec3{X: -230, Y: -158, Z: 825}),
	}
}

func buildCalibInput(t *testing.T, cam Camera, poses []Pose, spec CheckerSpec) CalibrationInput {
	t.Helper()
	in := CalibrationInput{Width: cam.Width, Height: cam.Height}
	for f, pose := range poses {
		img := RenderCheckerboard(cam, pose, spec, RenderOptions{Width: cam.Width, Height: cam.Height, SSAA: 2, Noise: 1.2, Seed: int64(100 + f)})
		d := DetectCheckerboard(img, spec)
		t.Logf("frame %d found=%d complete=%v rmse=%.3f spacing=%.1f", f, d.FoundCount, d.Complete, d.RMSE, d.Spacing)
		if !d.Complete {
			t.Fatalf("frame %d incomplete: %d corners", f, d.FoundCount)
		}
		in.Frames = append(in.Frames, CalibFrame{Object: d.ObjectPoints(), Observed: d.Observed()})
	}
	return in
}

func TestCalibrationRecoversCamera(t *testing.T) {
	cam := testCamera()
	spec := CheckerSpec{CornersX: 9, CornersY: 6, Square: 60}
	in := buildCalibInput(t, cam, demoPoses(), spec)
	got, err := Calibrate(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rmse=%.4f iter=%d fx=%.2f fy=%.2f cx=%.2f cy=%.2f k1=%.4f k2=%.4f", got.RMSE, got.Iterations, got.Camera.Fx, got.Camera.Fy, got.Camera.Cx, got.Camera.Cy, got.Camera.K1, got.Camera.K2)
	if got.RMSE > 1.8 {
		t.Fatalf("rmse too high: %.3f", got.RMSE)
	}
	if relErr(got.Camera.Fx, cam.Fx) > 0.04 || relErr(got.Camera.Fy, cam.Fy) > 0.04 {
		t.Fatalf("focal length not recovered: got %.1f,%.1f want %.1f,%.1f", got.Camera.Fx, got.Camera.Fy, cam.Fx, cam.Fy)
	}
	if abs(got.Camera.Cx-cam.Cx) > 18 || abs(got.Camera.Cy-cam.Cy) > 18 {
		t.Fatalf("principal point not recovered: %.1f,%.1f", got.Camera.Cx, got.Camera.Cy)
	}
}

func relErr(a, b float64) float64 { return abs(a-b) / abs(b) }
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
