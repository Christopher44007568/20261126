package main

import "testing"

func testCamera() Camera {
	return Camera{Width: 960, Height: 720, Fx: 880, Fy: 905, Cx: 486, Cy: 354, K1: -0.18, K2: 0.04, P1: 0.0015, P2: -0.001}
}

func testPose() Pose {
	return PoseFromRV(Vec3{X: 0.10, Y: -0.16, Z: 0.05}, Vec3{X: -245, Y: -160, Z: 780})
}

func TestDetectSyntheticSmoke(t *testing.T) {
	cam := testCamera()
	pose := testPose()
	spec := CheckerSpec{CornersX: 9, CornersY: 6, Square: 60}
	img := RenderCheckerboard(cam, pose, spec, RenderOptions{Width: cam.Width, Height: cam.Height, SSAA: 2, Noise: 1.5, Seed: 7})
	d := DetectCheckerboard(img, spec)
	t.Logf("raw=%d spacing=%.2f found=%d/%d rmse=%.3f complete=%v usable=%v", d.RawCornerCount, d.Spacing, d.FoundCount, spec.CornersX*spec.CornersY, d.RMSE, d.Complete, d.Usable)
	if d.FoundCount < spec.CornersX*spec.CornersY-4 {
		for _, p := range d.Points {
			if !p.Found {
				t.Logf("missing id=%d i=%d j=%d", p.ID, p.I, p.J)
			}
		}
		t.Fatalf("found only %d corners", d.FoundCount)
	}
}
