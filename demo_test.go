package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func countColor(img *image.RGBA, target color.RGBA, tolerance int) int {
	count := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.RGBAAt(x, y)
			if absDiff(c.R, target.R) <= tolerance &&
				absDiff(c.G, target.G) <= tolerance &&
				absDiff(c.B, target.B) <= tolerance {
				count++
			}
		}
	}
	return count
}

func containsCameraID(versions []uint64, want CameraVersion) bool {
	for _, version := range versions {
		if CameraVersion(version) == want {
			return true
		}
	}
	return false
}

func TestBuildDemoCapturesStaleAndFreshVersions(t *testing.T) {
	result, err := BuildDemo()
	if err != nil {
		t.Fatal(err)
	}
	report := result.Report
	if !report.SameTexturePixels {
		t.Fatal("demo did not use identical texture pixels for both frames")
	}
	if report.NormalBinding.Stale {
		t.Fatalf("normal frame is stale: %v", report.NormalBinding.Invalidations)
	}
	if !report.StaleBinding.Stale {
		t.Fatal("stale frame is not marked stale")
	}
	if report.StaleBinding.Camera.Version == report.NormalBinding.Camera.Version {
		t.Fatal("normal and stale frames use the same camera version")
	}
	if !report.StaleBinding.CameraAlive {
		t.Fatal("stale camera was not retained")
	}
	if report.StaleBinding.CameraReferences != 1 {
		t.Fatalf("stale camera refs = %d, want 1 at report capture", report.StaleBinding.CameraReferences)
	}
	if report.StaleBinding.TextureVersion != report.NormalBinding.TextureVersion {
		t.Fatal("texture versions differ between frames")
	}
	if report.StaleBinding.TextureHash != report.NormalBinding.TextureHash {
		t.Fatal("texture hashes differ between frames")
	}
	if report.ProjectionDelta < 50 {
		t.Fatalf("projection delta too small to demonstrate the bug: %.2f", report.ProjectionDelta)
	}
	if report.FrameDifference.ChangedPixels == 0 {
		t.Fatal("normal and stale frames are pixel-identical")
	}
	if report.FrameDifference.Percent < 15 {
		t.Fatalf("frame difference too small: %.2f%%", report.FrameDifference.Percent)
	}

	if !containsCameraID(report.Lifetime.RetainedCameras, report.StaleBinding.Camera.Version) {
		t.Fatal("lifetime report lost the pinned camera")
	}
	if !containsCameraID(report.Lifetime.RetainedCameras, report.NormalBinding.Camera.Version) {
		t.Fatal("lifetime report lost the current camera")
	}
	if report.Lifetime.PinnedCameraRefs != 1 {
		t.Fatalf("pinned camera refs = %d, want 1", report.Lifetime.PinnedCameraRefs)
	}

	if result.Normal.Bounds().Dx() != canvasW || result.Normal.Bounds().Dy() != canvasH {
		t.Fatal("normal frame has the wrong dimensions")
	}
	if result.Stale.Bounds().Dx() != canvasW || result.Stale.Bounds().Dy() != canvasH {
		t.Fatal("stale frame has the wrong dimensions")
	}
	if result.Overlay.Bounds().Dx() != overlayW || result.Overlay.Bounds().Dy() != overlayH {
		t.Fatal("overlay has the wrong dimensions")
	}
	if countColor(result.Overlay, colCyan, 4) == 0 ||
		countColor(result.Overlay, colAmber, 4) == 0 ||
		countColor(result.Overlay, colMagenta, 4) == 0 {
		t.Fatal("overlay is missing a version-source color")
	}
}

func TestRunDemoWritesPngsAndReport(t *testing.T) {
	dir := t.TempDir()
	result, err := RunDemo(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{fileNormalFrame, fileStaleFrame, fileOverlay, fileReport} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() < 500 {
			t.Fatalf("%s is unexpectedly small: %d bytes", name, info.Size())
		}
	}

	for _, item := range []struct {
		name string
		w, h int
	}{
		{fileNormalFrame, canvasW, canvasH},
		{fileStaleFrame, canvasW, canvasH},
		{fileOverlay, overlayW, overlayH},
	} {
		file, err := os.Open(filepath.Join(dir, item.name))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if img.Bounds().Dx() != item.w || img.Bounds().Dy() != item.h {
			t.Fatalf("%s dimensions = %v", item.name, img.Bounds())
		}
	}

	var report DemoReport
	data, err := os.ReadFile(filepath.Join(dir, fileReport))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != 1 || report.TextureVersion != result.Report.TextureVersion {
		t.Fatalf("report does not match generated result: %+v", report)
	}
	if len(report.StaleBinding.Invalidations) == 0 {
		t.Fatal("report is missing stale invalidation flags")
	}
}
