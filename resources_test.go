package main

import (
	"image"
	"image/color"
	"testing"
)

func solidImage(size int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func containsCameraVersion(versions []CameraVersion, want CameraVersion) bool {
	for _, version := range versions {
		if version == want {
			return true
		}
	}
	return false
}

func TestCameraInstallDoesNotOverwritePinnedRevision(t *testing.T) {
	m1 := CameraMatrix(100, 100, 40, 30, 0, 0)
	m2 := CameraMatrix(220, 180, 80, 70, 0.1, 0.01)
	cameras := NewCameraPool(m1, "CAM.V1")
	v1 := cameras.Current()
	if _, ok := cameras.Acquire(v1.Version); !ok {
		t.Fatal("acquire v1 failed")
	}

	v2 := cameras.Install(m2, "CAM.V2")
	if v2.Version == v1.Version {
		t.Fatal("install reused the old camera version")
	}
	got, ok := cameras.Get(v1.Version)
	if !ok {
		t.Fatal("pinned camera v1 disappeared")
	}
	if !got.Matrix.equal(m1) {
		t.Fatal("camera install overwrote the pinned v1 matrix")
	}
	if !cameras.Current().Matrix.equal(m2) {
		t.Fatal("current camera was not advanced to v2")
	}

	if removed := cameras.GC(); containsCameraVersion(removed, v1.Version) {
		t.Fatal("GC removed a camera with a live reference")
	}
	cameras.Release(v1.Version)
	if removed := cameras.GC(); !containsCameraVersion(removed, v1.Version) {
		t.Fatal("GC did not collect the unreferenced camera")
	}
}

func TestTextureInstallCopiesAndKeepsOldPixels(t *testing.T) {
	store := NewTextureStore()
	source := solidImage(8, color.RGBA{R: 11, G: 22, B: 33, A: 255})
	first := store.Install(source, "TEX.V1")
	source.SetRGBA(0, 0, color.RGBA{R: 250, G: 0, B: 0, A: 255})

	_, pixels, ok := store.Get(first.Version)
	if !ok {
		t.Fatal("first texture disappeared")
	}
	if got := pixels.RGBAAt(0, 0); got.R != 11 || got.G != 22 || got.B != 33 {
		t.Fatalf("published texture was mutated through caller pixels: %#v", got)
	}

	second := store.Install(solidImage(8, color.RGBA{R: 44, G: 55, B: 66, A: 255}), "TEX.V2")
	_, oldPixels, _ := store.Get(first.Version)
	_, newPixels, _ := store.Get(second.Version)
	if oldPixels.RGBAAt(0, 0) == newPixels.RGBAAt(0, 0) {
		t.Fatal("second texture install changed the old texture")
	}
}
