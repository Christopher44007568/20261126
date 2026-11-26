package audit

import (
	"math"
	"testing"
)

func solidTexture(role Role, value Vec3) *Texture {
	return &Texture{
		Role:               role,
		DeclaredColorSpace: ColorSpaceLinear,
		Channel:            "rgb",
		Width:              1,
		Height:             1,
		Pixels: []Pixel{{
			R: float32(value.X),
			G: float32(value.Y),
			B: float32(value.Z),
			A: 1,
		}},
	}
}

func testTextures() MaterialTextures {
	normal := solidTexture(RoleNormal, Vec3{0.5, 0.5, 1})
	roughness := solidTexture(RoleRoughness, Vec3{0.35, 0.35, 0.35})
	roughness.Channel = "r"
	return MaterialTextures{
		Albedo:    solidTexture(RoleAlbedo, Vec3{0.5, 0.12, 0.04}),
		Normal:    normal,
		Roughness: roughness,
	}
}

func tinyManifest() Manifest {
	manifest := defaultManifest()
	manifest.Render = RenderConfig{Width: 48, Height: 48, Samples: 1, Exposure: 1}
	manifest.Material = MaterialConfig{Metallic: 0, Specular: 0.5, NormalStrength: 1}
	manifest.Light = LightConfig{AzimuthDegrees: -35, ElevationDegrees: 42, Intensity: 4, Ambient: 0.1}
	return manifest
}

func TestRenderProducesFiniteImageAndMisreadDifference(t *testing.T) {
	manifest := tinyManifest()
	textures := testTextures()
	correct := Render(manifest, textures, RenderVariant{})
	misread := Render(manifest, textures, RenderVariant{ForceSRGBAlbedo: true})
	if correct.Width != 48 || correct.Height != 48 {
		t.Fatalf("unexpected dimensions %dx%d", correct.Width, correct.Height)
	}
	difference := 0.0
	for index, value := range correct.Pixels {
		if !finite(value) || !finite(misread.Pixels[index]) {
			t.Fatalf("render produced a non-finite pixel at %d", index)
		}
		difference += math.Abs(value.X - misread.Pixels[index].X)
		difference += math.Abs(value.Y - misread.Pixels[index].Y)
		difference += math.Abs(value.Z - misread.Pixels[index].Z)
	}
	if difference < 1 {
		t.Fatalf("forced sRGB albedo produced almost no image difference: %v", difference)
	}
}

func TestNeutralNormalMapDoesNotShiftSurface(t *testing.T) {
	objectNormal := Vec3{0.2, 0.4, 0.89}.Normalize()
	result := tangentSpaceNormal(Vec3{0.5, 0.5, 1}, objectNormal, 1)
	if result.Sub(objectNormal).Length() > 1e-9 {
		t.Fatalf("neutral normal map shifted normal from %+v to %+v", objectNormal, result)
	}
}

func TestErrorMapKnownDifference(t *testing.T) {
	correct := NewImage(2, 1)
	misread := NewImage(2, 1)
	correct.Set(0, 0, Vec3{0.2, 0.2, 0.2})
	correct.Set(1, 0, Vec3{0.6, 0.6, 0.6})
	misread.Set(0, 0, Vec3{0.2, 0.2, 0.2})
	misread.Set(1, 0, Vec3{0.3, 0.3, 0.3})

	errorMap, metrics, err := BuildErrorMap(correct, misread, 2)
	if err != nil {
		t.Fatal(err)
	}
	if errorMap.Width != 2 || errorMap.Height != 1 {
		t.Fatalf("unexpected error map dimensions")
	}
	if math.Abs(metrics.Max-0.3) > 1e-12 {
		t.Fatalf("max delta = %v, want 0.3", metrics.Max)
	}
	if math.Abs(metrics.Mean-0.15) > 1e-12 {
		t.Fatalf("mean delta = %v, want 0.15", metrics.Mean)
	}
	if metrics.ChangedPixels != 1 {
		t.Fatalf("changed pixels = %d, want 1", metrics.ChangedPixels)
	}
}

func TestComparisonDimensions(t *testing.T) {
	left := NewImage(10, 8)
	right := NewImage(10, 8)
	comparison, err := BuildComparison(left, right, "LEFT", "RIGHT")
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Width != 24*2+10*2+18 || comparison.Height != 24*2+50+8 {
		t.Fatalf("unexpected comparison dimensions %dx%d", comparison.Width, comparison.Height)
	}
}

func finite(value Vec3) bool {
	return !math.IsNaN(value.X) && !math.IsInf(value.X, 0) &&
		!math.IsNaN(value.Y) && !math.IsInf(value.Y, 0) &&
		!math.IsNaN(value.Z) && !math.IsInf(value.Z, 0)
}
