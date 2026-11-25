package colorspace

import (
	"math"
	"testing"
)

func TestSRGBKnownMidpoint(t *testing.T) {
	linear := SRGBToLinear(128)
	if math.Abs(linear-0.2158605001) > 1e-9 {
		t.Fatalf("SRGBToLinear(128)=%.9f, want about 0.215860500", linear)
	}
	if got := LinearToSRGB(linear); got != 128 {
		t.Fatalf("LinearToSRGB round trip = %d, want 128", got)
	}
}

func TestLinearizeKeepsAlphaOutOfTransferCurve(t *testing.T) {
	srgb := LinearizeRGBA(SRGB, 0, 255, 128, 64)
	linear := LinearizeRGBA(Linear, 0, 255, 128, 64)
	if srgb[3] != 64.0/255.0 || linear[3] != 64.0/255.0 {
		t.Fatalf("alpha changed: srgb=%v linear=%v", srgb[3], linear[3])
	}
	if math.Abs(srgb[2]-0.2158605001) > 1e-9 {
		t.Fatalf("unexpected sRGB blue linearization: %.9f", srgb[2])
	}
	if linear[2] != 128.0/255.0 {
		t.Fatalf("linear data was transformed: %.9f", linear[2])
	}
}

func TestParseMarker(t *testing.T) {
	cases := map[string]Space{
		"linear": Linear,
		"sRGB":   SRGB,
		"SRGB":   SRGB,
	}
	for input, want := range cases {
		got, err := Parse(input)
		if err != nil || got != want {
			t.Fatalf("Parse(%q)=(%v,%v), want %v", input, got, err, want)
		}
	}
	if _, err := Parse("gamma2.2"); err == nil {
		t.Fatal("expected unsupported marker error")
	}
}
