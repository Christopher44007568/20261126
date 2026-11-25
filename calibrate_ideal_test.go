package main

import "testing"

func TestCalibrateIdealPoints(t *testing.T) {
	cam := testCamera()
	spec := CheckerSpec{CornersX: 9, CornersY: 6, Square: 60}
	var obj []Vec3
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			obj = append(obj, Vec3{X: float64(i) * 60, Y: float64(j) * 60})
		}
	}
	in := CalibrationInput{Width: cam.Width, Height: cam.Height}
	for _, pose := range demoPoses() {
		var obs []Vec2
		for _, p := range obj {
			obs = append(obs, cam.Project(p, pose))
		}
		in.Frames = append(in.Frames, CalibFrame{Object: obj, Observed: obs})
	}
	got, err := Calibrate(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ideal rmse=%.4f iter=%d fx=%.2f fy=%.2f cx=%.2f cy=%.2f k1=%.4f k2=%.4f", got.RMSE, got.Iterations, got.Camera.Fx, got.Camera.Fy, got.Camera.Cx, got.Camera.Cy, got.Camera.K1, got.Camera.K2)
	if got.RMSE > 1 {
		t.Fatalf("ideal fit rmse %.3f", got.RMSE)
	}
}
