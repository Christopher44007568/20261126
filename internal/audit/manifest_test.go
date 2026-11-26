package audit

import (
	"strings"
	"testing"
)

func testManifest() Manifest {
	manifest := defaultManifest()
	manifest.Textures = TextureSet{
		Albedo:    TextureDeclaration{Path: "a.png", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Normal:    TextureDeclaration{Path: "b.png", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Roughness: TextureDeclaration{Path: "c.png", ColorSpace: ColorSpaceLinear, Channel: "r"},
	}
	return manifest
}

func TestManifestRejectsSRGBDataTextures(t *testing.T) {
	manifest := testManifest()
	manifest.Textures.Normal.ColorSpace = ColorSpaceSRGB
	err := manifest.Validate()
	if err == nil || !strings.Contains(err.Error(), "must be declared linear") {
		t.Fatalf("expected linear-data validation error, got %v", err)
	}
}

func TestManifestRequiresExplicitColorSpace(t *testing.T) {
	manifest := testManifest()
	manifest.Textures.Albedo.ColorSpace = ""
	err := manifest.Validate()
	if err == nil || !strings.Contains(err.Error(), "unsupported color_space") {
		t.Fatalf("expected color-space validation error, got %v", err)
	}
}

func TestManifestRequiresAllThreeRoles(t *testing.T) {
	manifest := testManifest()
	manifest.Textures.Roughness.Path = ""
	err := manifest.Validate()
	if err == nil || !strings.Contains(err.Error(), "roughness texture path is required") {
		t.Fatalf("expected missing role error, got %v", err)
	}
}

func TestManifestRejectsInvalidRoughnessChannel(t *testing.T) {
	manifest := testManifest()
	manifest.Textures.Roughness.Channel = "xy"
	err := manifest.Validate()
	if err == nil || !strings.Contains(err.Error(), "roughness channel") {
		t.Fatalf("expected channel validation error, got %v", err)
	}
}
