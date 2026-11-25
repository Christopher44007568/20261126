package diagnose

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"texturelab/colorspace"
	"texturelab/texcontainer"
)

func TestNormalMapSRGBMislabelLocatesLevel(t *testing.T) {
	normal := uniformNormalTexture()
	originalSpace := normal.Space

	report, err := DiagnoseNormalMap("unit-normal", normal, DefaultThreshold)
	if err != nil {
		t.Fatal(err)
	}
	if normal.Space != originalSpace {
		t.Fatal("diagnostic mutated the source texture marker")
	}
	if !report.Mismatch || report.FirstFailingLevel != 0 || report.WorstLevel != 0 {
		t.Fatalf("unexpected report summary: mismatch=%v first=%d worst=%d",
			report.Mismatch, report.FirstFailingLevel, report.WorstLevel)
	}
	if len(report.Levels) != 2 {
		t.Fatalf("got %d level reports", len(report.Levels))
	}

	expectedDelta := 128.0/255.0 - colorspace.SRGBToLinear(128)
	for _, level := range report.Levels {
		if !level.Detected || level.Width != level.Height {
			t.Fatalf("level %d not detected: %+v", level.Level, level)
		}
		if math.Abs(level.MaxAbsDelta-expectedDelta) > 1e-9 {
			t.Fatalf("level %d delta %.9f, want %.9f", level.Level, level.MaxAbsDelta, expectedDelta)
		}
		if level.ChangedTexels != level.TotalTexels || level.ChangedBounds == nil {
			t.Fatalf("level %d did not locate all uniform texels: %+v", level.Level, level)
		}
		if level.DeltaFingerprint == "" {
			t.Fatalf("level %d missing stable fingerprint", level.Level)
		}
	}

	locationErr := report.LocateFirstMismatch()
	if locationErr == nil {
		t.Fatal("expected locatable mismatch error")
	}
	message := locationErr.Error()
	for _, fragment := range []string{"mip level 0", "4x4", "texel (0,0)", "channel R"} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("location error %q does not contain %q", message, fragment)
		}
	}
}

func TestCorrectMarkerHasNoInterpretationDelta(t *testing.T) {
	normal := uniformNormalTexture()
	report, err := CompareInterpretations("correct", normal, normal, DefaultThreshold)
	if err != nil {
		t.Fatal(err)
	}
	if report.Mismatch || report.FirstFailingLevel != -1 {
		t.Fatalf("correct linear marker produced false mismatch: %+v", report)
	}
}

func uniformNormalTexture() *texcontainer.Texture {
	texel := []byte{128, 128, 255, 255}
	pixels4 := bytes.Repeat(texel, 4*4)
	pixels2 := bytes.Repeat(texel, 2*2)
	return &texcontainer.Texture{
		Format: texcontainer.FormatRGBA8,
		Space:  colorspace.Linear,
		Levels: []texcontainer.Level{
			{Width: 4, Height: 4, Pixels: pixels4},
			{Width: 2, Height: 2, Pixels: pixels2},
		},
	}
}
