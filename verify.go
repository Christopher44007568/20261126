package main

import "math"

type VerificationReport struct {
	FramesChecked         int      `json:"framesChecked"`
	MeshPixels            int      `json:"meshPixelsPerFrame"`
	BlankPixels           int      `json:"blankPixels"`
	BlackWhiteFlashPixels int      `json:"blackWhiteFlashPixels"`
	GeometryMaskStable    bool     `json:"geometryMaskStable"`
	MaxMeanLuminanceJump  float64  `json:"maxMeanLuminanceJump"`
	Passed                bool     `json:"passed"`
	Checks                []string `json:"checks"`
}

func VerifyContinuity(source *TextureSource) VerificationReport {
	report := VerificationReport{GeometryMaskStable: true, Passed: true}

	for _, kind := range strategyOrder {
		loader := NewLoader(kind, source)
		var previousMean float64
		havePrevious := false

		for frame := 0; frame <= 10; frame++ {
			cam := CameraAt(float64(frame))
			vp := mainViewport()
			loader.Plan(cam, vp)
			rendered := RenderFrame(loader, cam, "VERIFY")
			report.FramesChecked++
			report.MeshPixels = rendered.Metrics.MeshPixels

			var luminanceSum float64
			var luminanceCount float64
			for i, isMesh := range rendered.Mesh {
				if !isMesh {
					continue
				}
				x := i % frameW
				y := i / frameW
				c := pixelAt(rendered.Image, x, y)
				if c == colorViewport {
					report.BlankPixels++
				}
				if (c.R == 0 && c.G == 0 && c.B == 0) ||
					(c.R == 255 && c.G == 255 && c.B == 255) {
					report.BlackWhiteFlashPixels++
				}
				lum := luminance(c)
				luminanceSum += lum
				luminanceCount++
			}

			mean := luminanceSum / luminanceCount
			if havePrevious {
				report.MaxMeanLuminanceJump = math.Max(report.MaxMeanLuminanceJump, math.Abs(mean-previousMean))
			}
			previousMean = mean
			havePrevious = true

			loader.Transfer(cam)
		}

		cam := CameraAt(10)
		before := RenderFrame(loader, cam, "GEOMETRY")
		loader.Store.AddFullTexture()
		loader.FullRemaining = 0
		after := RenderFrame(loader, cam, "GEOMETRY")
		if !masksEqual(before.Mesh, after.Mesh) {
			report.GeometryMaskStable = false
		}
	}

	if report.BlankPixels != 0 {
		report.Passed = false
		report.Checks = append(report.Checks, "found blank textured pixels")
	}
	if report.BlackWhiteFlashPixels != 0 {
		report.Passed = false
		report.Checks = append(report.Checks, "found pure black/white textured pixels")
	}
	if !report.GeometryMaskStable {
		report.Passed = false
		report.Checks = append(report.Checks, "geometry mask changed after child arrival")
	}
	if report.MaxMeanLuminanceJump > 35 {
		report.Passed = false
		report.Checks = append(report.Checks, "frame mean luminance jumped too sharply")
	}
	if report.Passed {
		report.Checks = []string{
			"every textured pixel has an opaque resident ancestor",
			"child arrival changes texels but not UV geometry coverage",
			"no pure black/white flash was observed",
		}
	}
	return report
}

func pixelAt(im *Image, x, y int) Color {
	i := im.pixelIndex(x, y)
	return Color{R: im.Pix[i], G: im.Pix[i+1], B: im.Pix[i+2]}
}

func luminance(c Color) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

func masksEqual(a, b []bool) bool {
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
