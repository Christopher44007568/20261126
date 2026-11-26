package main

import (
	"path/filepath"
	"testing"
)

func TestPPMRoundTrip(t *testing.T) {
	img := NewImage(3, 2)
	img.Set(0, 0, RGB{255, 0, 255})
	img.Set(1, 0, RGB{255, 244, 92})
	img.Set(2, 1, RGB{17, 128, 211})

	path := filepath.Join(t.TempDir(), "roundtrip.ppm")
	if err := WritePPM(path, img); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPPM(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != img.Width || got.Height != img.Height {
		t.Fatalf("size = %dx%d, want %dx%d", got.Width, got.Height, img.Width, img.Height)
	}
	for i := range img.Pixels {
		if got.Pixels[i] != img.Pixels[i] {
			t.Fatalf("pixel %d = %v, want %v", i, got.Pixels[i], img.Pixels[i])
		}
	}
}
