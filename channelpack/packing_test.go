package main

import "testing"

func TestRoundTripInvariants(t *testing.T) {
	src := buildSource(96, 96)
	for _, l := range allLayouts() {
		got := l.Decode(l.Encode(src), src.W, src.H)
		stats := compareMaps(src, got)
		for _, st := range stats {
			if len(st.Name) >= 6 && st.Name[:6] == "ALBEDO" && st.MaxAbs != 0 {
				t.Fatalf("%s: %s must be byte-exact, max=%d", l.Name, st.Name, st.MaxAbs)
			}
			if len(st.Name) >= 6 && st.Name[:6] == "NORMAL" && st.MaxAbs != 0 {
				t.Fatalf("%s: %s must be byte-exact, max=%d", l.Name, st.Name, st.MaxAbs)
			}
		}
		if angle := maxNormalAngleDeg(src, got); angle > 1e-4 {
			t.Fatalf("%s: normal angle error %v exceeds tolerance", l.Name, angle)
		}
		if !scalarIndependence(l, src) {
			t.Fatalf("%s: scalar changes leaked into albedo or normals", l.Name)
		}
	}
}

func TestCompactScalarBound(t *testing.T) {
	src := buildSource(96, 96)
	got := decodeCompact(buildCompact(src), src.W, src.H)
	for _, st := range compareMaps(src, got) {
		if (st.Name == "ROUGH" || st.Name == "AO") && st.MaxAbs > 8 {
			t.Fatalf("%s quantization error %d exceeds one 4-bit bin", st.Name, st.MaxAbs)
		}
	}
}

func TestMisalignedIsDetectable(t *testing.T) {
	src := buildSource(96, 96)
	mis := decodeMisaligned(buildPacked(src), src.W, src.H)
	if diff := countAlbedoDifferences(src, mis); diff == 0 {
		t.Fatal("misaligned decoder produced no detectable difference")
	}
}
