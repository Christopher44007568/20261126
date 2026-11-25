package main

import (
	"bytes"
	"testing"
)

func TestWritePPMEncodesP6HeaderAndPixels(t *testing.T) {
	im := NewImage(2, 1, rgb(1, 2, 3))
	im.Set(0, 0, rgb(10, 20, 30))
	im.Set(1, 0, rgb(40, 50, 60))

	var buf bytes.Buffer
	if err := im.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}

	prefix := []byte("P6\n2 1\n255\n")
	if !bytes.HasPrefix(buf.Bytes(), prefix) {
		t.Fatalf("PPM prefix = %q", buf.Bytes()[:len(prefix)])
	}
	pixels := buf.Bytes()[len(prefix):]
	want := []byte{10, 20, 30, 40, 50, 60}
	if !bytes.Equal(pixels, want) {
		t.Fatalf("pixels = %v, want %v", pixels, want)
	}
}

func TestEveryUVResolvesToResidentAncestor(t *testing.T) {
	source := NewTextureSource()
	for _, kind := range strategyOrder {
		loader := NewLoader(kind, source)
		for _, frame := range []float64{0, 2.5, 5, 7.5, 10} {
			cam := CameraAt(frame)
			for u := 0; u <= 10; u++ {
				for v := 0; v <= 10; v++ {
					uv := UV{U: float64(u) / 10, V: float64(v) / 10}
					info := loader.Inspect(uv, cam)
					if info.Tile == nil {
						t.Fatalf("%s frame %.1f UV %v has no tile", kind, frame, uv)
					}
					if info.ResolvedLevel < 0 || info.ResolvedLevel > maxLOD {
						t.Fatalf("invalid resolved LOD %d", info.ResolvedLevel)
					}
					color := info.Tile.sample(0.5, 0.5)
					if color.R == 0 && color.G == 0 && color.B == 0 {
						t.Fatalf("fallback tile is pure black")
					}
				}
			}
		}
	}
}

func TestGeometryMaskStableWhenChildrenArrive(t *testing.T) {
	source := NewTextureSource()
	for _, kind := range strategyOrder {
		loader := NewLoader(kind, source)
		cam := CameraAt(8)
		before := RenderFrame(loader, cam, "BEFORE")

		loader.Store.AddFullTexture()
		loader.FullRemaining = 0
		after := RenderFrame(loader, cam, "AFTER")

		if !masksEqual(before.Mesh, after.Mesh) {
			t.Fatalf("%s geometry/coverage mask changed after child load", kind)
		}
		if before.Metrics.MeshPixels == 0 || after.Metrics.MeshPixels == 0 {
			t.Fatalf("%s rendered no mesh pixels", kind)
		}
	}
}

func TestVerificationPasses(t *testing.T) {
	report := VerifyContinuity(NewTextureSource())
	if !report.Passed {
		t.Fatalf("verification failed: %+v", report)
	}
	if report.BlankPixels != 0 || report.BlackWhiteFlashPixels != 0 {
		t.Fatalf("blank=%d blackWhite=%d", report.BlankPixels, report.BlackWhiteFlashPixels)
	}
	if !report.GeometryMaskStable {
		t.Fatal("geometry mask unstable")
	}
}
