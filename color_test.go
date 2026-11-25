package main

import (
	"math"
	"testing"
)

func TestDeltaE2000ReferencePairs(t *testing.T) {
	cases := []struct {
		name string
		a    Lab
		b    Lab
		want float64
	}{
		{"sharma-01", Lab{50, 2.6772, -79.7751}, Lab{50, 0, -82.7485}, 2.0425},
		{"sharma-02", Lab{50, 3.1571, -77.2803}, Lab{50, 0, -82.7485}, 2.8615},
		{"sharma-03", Lab{50, 2.8361, -74.0200}, Lab{50, 0, -82.7485}, 3.4412},
		{"sharma-04", Lab{50, -1.3802, -84.2814}, Lab{50, 0, -82.7485}, 1.0000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deltaE2000(tc.a, tc.b)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Fatalf("deltaE2000 = %.6f, want %.4f", got, tc.want)
			}
		})
	}
}

func TestRGBToLabNeutralAxis(t *testing.T) {
	white := RGB{255, 255, 255}.lab()
	black := RGB{0, 0, 0}.lab()
	if math.Abs(white.L-100) > 0.02 || math.Abs(white.A) > 0.02 || math.Abs(white.B) > 0.02 {
		t.Fatalf("white Lab = %+v, want L=100 and a=b=0", white)
	}
	if math.Abs(black.L) > 0.02 || math.Abs(black.A) > 0.02 || math.Abs(black.B) > 0.02 {
		t.Fatalf("black Lab = %+v, want all zero", black)
	}
}

func TestParseHexColor(t *testing.T) {
	r, g, b, err := parseHexColor("#1a2B3c")
	if err != nil {
		t.Fatal(err)
	}
	if r != 26 || g != 43 || b != 60 {
		t.Fatalf("got #%02X%02X%02X", r, g, b)
	}
	r, g, b, err = parseHexColor("#fa0")
	if err != nil {
		t.Fatal(err)
	}
	if r != 255 || g != 170 || b != 0 {
		t.Fatalf("short hex got #%02X%02X%02X", r, g, b)
	}
	if _, _, _, err := parseHexColor("red"); err == nil {
		t.Fatal("expected invalid color error")
	}
}
