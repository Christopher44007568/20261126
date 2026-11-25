package main

import "testing"

func TestPackedImageUsesOnlyTwoChannels(t *testing.T) {
	samples, _ := BuildSphere(32, 8)
	img := renderPackedChannels(samples, 8)
	bounds := img.Bounds()
	insidePixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !samples[y][x].Inside {
				continue
			}
			insidePixels++
			pixel := img.RGBA64At(x, y)
			if pixel.B != 0 || pixel.A != 65535 {
				t.Fatalf("packed pixel (%d,%d) has B=%d A=%d", x, y, pixel.B, pixel.A)
			}
		}
	}

	if insidePixels == 0 {
		t.Fatal("no packed sphere pixels were rendered")
	}
}

func TestLightingComparisonHasBothSpherePanels(t *testing.T) {
	samples, _ := BuildSphere(32, 8)
	img := renderLightingComparison(samples)
	bounds := img.Bounds()

	if bounds.Dx() != 64 || bounds.Dy() != 32 {
		t.Fatalf("comparison size = %dx%d, want 64x32", bounds.Dx(), bounds.Dy())
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		pixel := img.RGBA64At(32, y)
		if pixel.R != 65535 || pixel.G != 65535 || pixel.B != 65535 {
			t.Fatalf("panel divider at y=%d is not white", y)
		}
	}
}
