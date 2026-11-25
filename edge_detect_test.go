package main

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func occludeDisk(img *image.RGBA, cx, cy, radius float64, col color.RGBA) {
	r2 := radius * radius
	b := img.Bounds()
	for y := int(math.Max(0, cy-radius)); y <= int(math.Min(float64(b.Dy()-1), cy+radius)); y++ {
		for x := int(math.Max(0, cx-radius)); x <= int(math.Min(float64(b.Dx()-1), cx+radius)); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy <= r2 {
				i := img.PixOffset(x, y)
				img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = col.R, col.G, col.B, 255
			}
		}
	}
}

func TestDetectMissingEdges(t *testing.T) {
	cam := testCamera()
	pose := testPose()
	spec := CheckerSpec{CornersX: 9, CornersY: 6, Square: 60}
	img := RenderCheckerboard(cam, pose, spec, RenderOptions{Width: cam.Width, Height: cam.Height, SSAA: 2, Noise: 1.2, Seed: 99})
	bg := color.RGBA{R: 178, G: 184, B: 190, A: 255}
	var edge []Vec2
	for j := 0; j < spec.CornersY; j++ {
		p := cam.Project(Vec3{X: float64(spec.CornersX-1) * 60, Y: float64(j) * 60}, pose)
		edge = append(edge, p)
		occludeDisk(img, p.X, p.Y, 27, bg)
		if j > 0 {
			prev := edge[j-1]
			steps := int(math.Ceil(math.Hypot(p.X-prev.X, p.Y-prev.Y) / 4))
			for s := 0; s <= steps; s++ {
				q := Vec2{X: prev.X + (p.X-prev.X)*float64(s)/float64(steps), Y: prev.Y + (p.Y-prev.Y)*float64(s)/float64(steps)}
				occludeDisk(img, q.X, q.Y, 15, bg)
			}
		}
	}
	d := DetectCheckerboard(img, spec)
	t.Logf("edge-miss found=%d/%d complete=%v edgeComplete=%v usable=%v", d.FoundCount, spec.CornersX*spec.CornersY, d.Complete, d.EdgeComplete, d.Usable)
	if d.Width != cam.Width || d.Height != cam.Height {
		t.Fatalf("detection resolution changed to %dx%d", d.Width, d.Height)
	}
	if d.Complete {
		t.Fatalf("occluded edge was incorrectly reported complete")
	}
	if d.EdgeComplete {
		t.Fatalf("edge completeness should be false")
	}
	if d.Usable {
		t.Fatalf("edge-missed detection must not be marked usable for final calibration")
	}
	if d.FoundCount > spec.CornersX*spec.CornersY-4 {
		t.Fatalf("expected several missed edge corners, found %d", d.FoundCount)
	}
}
