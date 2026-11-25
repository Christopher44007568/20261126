package main

import (
	"image"
	"image/color"
	"math"
)

func DefaultSpec() CheckerSpec {
	return CheckerSpec{CornersX: 9, CornersY: 6, Square: 60}
}

func DefaultCamera() Camera {
	return Camera{
		Width: 960, Height: 720,
		Fx: 880, Fy: 905, Cx: 486, Cy: 354,
		K1: -0.18, K2: 0.045, P1: 0.0015, P2: -0.0011,
	}
}

func DefaultPoses() []Pose {
	return []Pose{
		PoseFromRV(Vec3{X: 0.10, Y: -0.16, Z: 0.05}, Vec3{X: -245, Y: -160, Z: 790}),
		PoseFromRV(Vec3{X: -0.08, Y: 0.14, Z: -0.08}, Vec3{X: -235, Y: -150, Z: 835}),
		PoseFromRV(Vec3{X: 0.16, Y: 0.06, Z: 0.10}, Vec3{X: -255, Y: -168, Z: 850}),
		PoseFromRV(Vec3{X: -0.05, Y: -0.20, Z: -0.04}, Vec3{X: -230, Y: -158, Z: 825}),
		PoseFromRV(Vec3{X: 0.13, Y: 0.17, Z: -0.12}, Vec3{X: -250, Y: -155, Z: 865}),
	}
}

func OccludeRightEdge(img *image.RGBA, cam Camera, pose Pose, spec CheckerSpec) {
	bg := color.RGBA{R: 178, G: 184, B: 190, A: 255}
	var prev Vec2
	for j := 0; j < spec.CornersY; j++ {
		p := cam.Project(Vec3{
			X: float64(spec.CornersX-1) * spec.Square,
			Y: float64(j) * spec.Square,
		}, pose)
		fillCircle(img, p, 36, bg)
		if j > 0 {
			steps := int(math.Ceil(math.Hypot(p.X-prev.X, p.Y-prev.Y) / 4))
			for s := 0; s <= steps; s++ {
				q := Vec2{
					X: prev.X + (p.X-prev.X)*float64(s)/float64(steps),
					Y: prev.Y + (p.Y-prev.Y)*float64(s)/float64(steps),
				}
				fillCircle(img, q, 23, bg)
			}
		}
		prev = p
	}
}
