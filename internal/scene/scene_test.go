package scene

import "testing"

func TestSceneContainsHighlights(t *testing.T) {
	m := Build(160, 100)
	s := m.Compute()
	if s.OverWhite == 0 {
		t.Fatal("expected pixels above SDR white (linear 1.0)")
	}
	if s.MaxLum <= 1 {
		t.Fatalf("peak luminance %.3f must exceed 1.0", s.MaxLum)
	}
	if s.MinLum < 0 {
		t.Fatalf("negative radiance %.3f", s.MinLum)
	}
	if s.MeanLum <= 0 || s.MeanLum > s.MaxLum {
		t.Fatalf("implausible mean luminance %.3f", s.MeanLum)
	}
}
