package main

import (
	"path/filepath"
	"testing"
)

func TestPPMRoundTrip(t *testing.T) {
	img := NewRGB8(4, 3)
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			i := 3 * (y*4 + x)
			img.Pix[i] = uint8((i * 7) % 256)
			img.Pix[i+1] = uint8((i * 13) % 256)
			img.Pix[i+2] = uint8((i * 29) % 256)
		}
	}
	path := filepath.Join(t.TempDir(), "out.ppm")
	if err := img.WritePPM(path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPPM(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.W != 4 || got.H != 3 || len(got.Pix) != len(img.Pix) {
		t.Fatalf("dims mismatch: %dx%d", got.W, got.H)
	}
	for i := range img.Pix {
		if got.Pix[i] != img.Pix[i] {
			t.Fatalf("pixel byte %d: got %d want %d", i, got.Pix[i], img.Pix[i])
		}
	}
}
