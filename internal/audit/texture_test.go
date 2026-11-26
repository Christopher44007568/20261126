package audit

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestAlbedoFollowsDeclaredInterpretation(t *testing.T) {
	texture := &Texture{
		Role:               RoleAlbedo,
		DeclaredColorSpace: ColorSpaceSRGB,
		Channel:            "rgb",
		Width:              1,
		Height:             1,
		Pixels:             []Pixel{{R: 0.5, G: 0.5, B: 0.5, A: 1}},
	}
	linear := texture.SampleAlbedo(0.5, 0.5, ColorSpaceLinear)
	encoded := texture.SampleAlbedo(0.5, 0.5, ColorSpaceSRGB)
	if math.Abs(linear.X-0.5) > 1e-12 {
		t.Fatalf("linear interpretation changed value to %v", linear.X)
	}
	if math.Abs(encoded.X-SRGBToLinear(0.5)) > 1e-12 {
		t.Fatalf("sRGB interpretation produced %v, want %v", encoded.X, SRGBToLinear(0.5))
	}
}

func TestNormalAndRoughnessNeverApplyTransfer(t *testing.T) {
	raw := Pixel{R: 0.75, G: 0.5, B: 0.25, A: 1}
	normal := &Texture{
		Role:               RoleNormal,
		DeclaredColorSpace: ColorSpaceSRGB,
		Channel:            "rgb",
		Width:              1,
		Height:             1,
		Pixels:             []Pixel{raw},
	}
	roughness := &Texture{
		Role:               RoleRoughness,
		DeclaredColorSpace: ColorSpaceSRGB,
		Channel:            "r",
		Width:              1,
		Height:             1,
		Pixels:             []Pixel{raw},
	}

	sampledNormal := normal.SampleNormal(0.5, 0.5)
	if sampledNormal != (Vec3{0.75, 0.5, 0.25}) {
		t.Fatalf("normal data was transformed: %+v", sampledNormal)
	}
	if sampledRoughness := roughness.SampleRoughness(0.5, 0.5); sampledRoughness != 0.75 {
		t.Fatalf("roughness data was transformed: %v", sampledRoughness)
	}
}

func TestLoadTextureUsesDeclarationNotFilename(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "surface_looks_linear.png")
	source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, source); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	declaration := TextureDeclaration{
		Path:       path,
		ColorSpace: ColorSpaceSRGB,
		Channel:    "rgb",
	}
	texture, err := loadTexture(RoleAlbedo, declaration, path)
	if err != nil {
		t.Fatal(err)
	}
	value := texture.SampleAlbedo(0.5, 0.5, texture.DeclaredColorSpace)
	expected := SRGBToLinear(128.0 / 255.0)
	if math.Abs(value.X-expected) > 1e-7 {
		t.Fatalf("declared sRGB decode = %v, want %v", value.X, expected)
	}
}

func TestRoughnessChannelAndLuma(t *testing.T) {
	texture := &Texture{
		Role:    RoleRoughness,
		Channel: "g",
		Width:   1,
		Height:  1,
		Pixels:  []Pixel{{R: 0.1, G: 0.6, B: 0.2, A: 1}},
	}
	if actual := texture.SampleRoughness(0.5, 0.5); math.Abs(actual-0.6) > 1e-7 {
		t.Fatalf("green channel roughness = %v, want 0.6", actual)
	}
	texture.Channel = "luma"
	expected := 0.2126*0.1 + 0.7152*0.6 + 0.0722*0.2
	if actual := texture.SampleRoughness(0.5, 0.5); math.Abs(actual-expected) > 1e-7 {
		t.Fatalf("luma roughness = %v, want %v", actual, expected)
	}
}
