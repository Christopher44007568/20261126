package mipchain

import (
	"bytes"
	"image"
	"testing"

	"texturelab/colorspace"
	"texturelab/texcontainer"
)

func TestBuildThreeLevelChainSizesAndPacking(t *testing.T) {
	source := Albedo(64)
	images, err := BuildChain(source, 3, colorspace.SRGB, false)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int{{64, 64}, {32, 32}, {16, 16}}
	for i, dimensions := range want {
		bounds := images[i].Bounds()
		if bounds.Dx() != dimensions[0] || bounds.Dy() != dimensions[1] {
			t.Fatalf("level %d = %dx%d, want %v", i, bounds.Dx(), bounds.Dy(), dimensions)
		}
	}

	levels, err := ToContainerLevels(images)
	if err != nil {
		t.Fatal(err)
	}
	texture := &texcontainer.Texture{Format: texcontainer.FormatRGBA8, Space: colorspace.SRGB, Levels: levels}
	if err := texture.ValidateMipChain(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationIsDeterministic(t *testing.T) {
	first, err := BuildChain(NormalMap(64, 3.2), 3, colorspace.Linear, true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildChain(NormalMap(64, 3.2), 3, colorspace.Linear, true)
	if err != nil {
		t.Fatal(err)
	}
	for level := range first {
		if !bytes.Equal(first[level].Pix, second[level].Pix) {
			t.Fatalf("normal mip level %d is not deterministic", level)
		}
	}
}

func TestSRGBDownsampleUsesLinearLight(t *testing.T) {
	source := twoByTwoRGBA()
	level, err := Downsample2x(source, colorspace.SRGB, false)
	if err != nil {
		t.Fatal(err)
	}
	// Averaging encoded values would be around 191; linear-light averaging and
	// re-encoding yields a darker correct midpoint around 188.
	if got := level.Pix[0]; got < 184 || got > 191 {
		t.Fatalf("linear-light sRGB average R = %d, want 184..191", got)
	}

	linear, err := Downsample2x(source, colorspace.Linear, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := linear.Pix[0]; got != 128 {
		t.Fatalf("encoded/raw average R = %d, want 128", got)
	}
}

func twoByTwoRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	values := []byte{0, 0, 0, 255, 255, 255, 255, 255}
	copy(img.Pix[:4], values[:4])
	copy(img.Pix[4:8], values[4:])
	copy(img.Pix[8:12], values[:4])
	copy(img.Pix[12:16], values[4:])
	return img
}
