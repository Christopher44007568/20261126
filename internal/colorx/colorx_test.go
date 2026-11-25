package colorx

import "testing"

func TestSRGBRoundTrip(t *testing.T) {
	for _, lin := range []float64{0, 0.001, 0.01, 0.18, 0.5, 0.9, 1} {
		enc := SRGBEncode(lin)
		back := SRGBDecode(float64(enc) / 255)
		tol := 1.0/255 + 1e-3
		if d := absF(back - lin); d > tol {
			t.Fatalf("round trip %.4f -> %d -> %.4f (err %.4f)", lin, enc, back, d)
		}
	}
	if SRGBEncode(0) != 0 {
		t.Fatal("black must encode to 0")
	}
	if SRGBEncode(1) != 255 {
		t.Fatal("unity must encode to 255")
	}
	if SRGBEncode(2) != 255 {
		t.Fatal("above 1 must clamp to 255")
	}
}

func TestReinhardMonotonic(t *testing.T) {
	prev := -1.0
	for x := 0.0; x <= 12; x += 0.1 {
		v := Reinhard(x, 1)
		if v < prev || v >= 1 {
			t.Fatalf("reinhard not monotonic/bounded at %.1f -> %.4f", x, v)
		}
		prev = v
	}
	if Reinhard(1, 1) != 0.5 {
		t.Fatalf("reinhard(1) = %v, want 0.5", Reinhard(1, 1))
	}
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
