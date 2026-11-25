package main

import "testing"

func centerLuminance(c *Canvas) float64 {
	i := (c.H/2*c.W + c.W/2) * 3
	r := float64(c.Pix[i+0])
	g := float64(c.Pix[i+1])
	b := float64(c.Pix[i+2])
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func TestNearOccluderBlocksFarHighlight(t *testing.T) {
	const w, h = 96, 96
	aperture := RegularAperture(6, 0.04)
	cam := NewCamera(V(0, 0, 0), V(0, 0, 5), V(0, 1, 0), 40, 1, 5, aperture)

	rodMat := polishedMaterial(0.22, 0.27, 0.36, 0.3, 100)
	ballMat := emissiveMaterial(1, 1, 1, 12)
	both := &Scene{BackgroundColor: darkBackground}
	both.Add(&Rod{X: 0, Z: 2, Radius: 0.18, YMin: -3, YMax: 3, Mat: rodMat})
	both.Add(&Sphere{Center: V(0, 0, 6), Radius: 0.22, Mat: ballMat})

	result := Render(both, cam, w, h, RenderOptions{Samples: 64, Seed: 7})
	occluded := result.ToCanvas(1)
	if lum := centerLuminance(occluded); lum > 145 {
		t.Fatalf("rod-center luminance %.1f leaks far highlight", lum)
	}
	if d := result.Depth[(h/2)*w+w/2]; d > 3.0 {
		t.Fatalf("center depth = %.2f, want near rod", d)
	}

	ballOnly := &Scene{BackgroundColor: darkBackground}
	ballOnly.Add(&Sphere{Center: V(0, 0, 6), Radius: 0.22, Mat: ballMat})
	bright := Render(ballOnly, cam, w, h, RenderOptions{Samples: 64, Seed: 7}).ToCanvas(1)
	if lum := centerLuminance(bright); lum < 210 {
		t.Fatalf("unoccluded highlight luminance %.1f, want bright control", lum)
	}
}

func meanCanvasDifference(a, b *Canvas) float64 {
	if len(a.Pix) != len(b.Pix) {
		return 1e9
	}
	var sum int64
	for i := range a.Pix {
		d := int64(a.Pix[i]) - int64(b.Pix[i])
		if d < 0 {
			d = -d
		}
		sum += d
	}
	return float64(sum) / float64(len(a.Pix))
}

func TestSmallApertureReturnsNearPinholeImage(t *testing.T) {
	demo := highlightScene(0)
	const w, h = 120, 75
	demo.Camera.Aspect = float64(w) / float64(h)

	renderWithRadius := func(radius float64) *Canvas {
		demo.Camera.Aperture = RegularAperture(6, radius)
		return Render(demo.Scene, demo.Camera, w, h, RenderOptions{Samples: 64, Seed: 11}).ToCanvas(1)
	}

	pinhole := renderWithRadius(0)
	small := renderWithRadius(0.002)
	wide := renderWithRadius(0.12)
	smallDiff := meanCanvasDifference(pinhole, small)
	wideDiff := meanCanvasDifference(pinhole, wide)

	if smallDiff > 4.0 {
		t.Fatalf("small aperture differs from pinhole by %.2f/255", smallDiff)
	}
	if wideDiff <= smallDiff*3 {
		t.Fatalf("wide diff %.2f should be much larger than small diff %.2f", wideDiff, smallDiff)
	}
}
