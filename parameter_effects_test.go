package main

import "testing"

func TestParameterErrorsVisibleInReprojection(t *testing.T) {
	cam := testCamera()
	pose := testPose()
	spec := DefaultSpec()
	var object []Vec3
	var observed []Vec2
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			obj := Vec3{X: float64(i) * spec.Square, Y: float64(j) * spec.Square}
			object = append(object, obj)
			observed = append(observed, cam.Project(obj, pose))
		}
	}
	good := ReprojectionResiduals(cam, []Pose{pose}, [][]Vec3{object}, [][]Vec2{observed})
	if rmse := RMSEResidual(good); rmse > 1e-9 {
		t.Fatalf("ideal projection rmse = %v", rmse)
	}

	wrongFocal := cam
	wrongFocal.Fx *= 1.16
	focalRes := ReprojectionResiduals(wrongFocal, []Pose{pose}, [][]Vec3{object}, [][]Vec2{observed})
	if RMSEResidual(focalRes) < 10 {
		t.Fatalf("focal error not visible: rmse %.3f", RMSEResidual(focalRes))
	}

	wrongPrincipal := cam
	wrongPrincipal.Cx += 46
	ppRes := ReprojectionResiduals(wrongPrincipal, []Pose{pose}, [][]Vec3{object}, [][]Vec2{observed})
	if RMSEResidual(ppRes) < 30 {
		t.Fatalf("principal error not visible: rmse %.3f", RMSEResidual(ppRes))
	}

	wrongDist := cam
	wrongDist.K1 = 0
	distRes := ReprojectionResiduals(wrongDist, []Pose{pose}, [][]Vec3{object}, [][]Vec2{observed})
	if RMSEResidual(distRes) < 1 {
		t.Fatalf("distortion error not visible: rmse %.3f", RMSEResidual(distRes))
	}
}
