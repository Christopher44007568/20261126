package audit

import (
	"math"
	"testing"
)

func TestSRGBTransferKnownValues(t *testing.T) {
	tests := []struct {
		encoded float64
		linear  float64
	}{
		{0, 0},
		{0.04045, 0.04045 / 12.92},
		{0.18, 0.027211781},
		{0.5, 0.214041140},
		{1, 1},
	}
	for _, test := range tests {
		actual := SRGBToLinear(test.encoded)
		if math.Abs(actual-test.linear) > 1e-8 {
			t.Fatalf("SRGBToLinear(%v) = %.12f, want %.12f", test.encoded, actual, test.linear)
		}
	}
}

func TestSRGBRoundTrip(t *testing.T) {
	for _, value := range []float64{0, 0.003, 0.04045, 0.2, 0.5, 0.9, 1} {
		roundTrip := LinearToSRGB(SRGBToLinear(value))
		if math.Abs(roundTrip-value) > 1e-6 {
			t.Fatalf("round trip of %v produced %v", value, roundTrip)
		}
	}
}
