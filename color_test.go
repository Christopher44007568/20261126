package main

import (
	"math"
	"testing"
)

func approxEq(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.6f, want %.6f", name, got, want)
	}
}

func TestMatrixInverseIdentity(t *testing.T) {
	m := rgbToXYZ(sRGBPrimaries, WP_D65)
	inv := m.inverse()
	prod := m.mul(inv)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			want := 0.0
			if i == j {
				want = 1.0
			}
			approxEq(t, "inverse product", prod[i][j], want, 1e-9)
		}
	}
}

func TestSRGBRoundTrip(t *testing.T) {
	for _, v := range []float64{0.001, 0.05, 0.3, 0.8, 1.0} {
		approxEq(t, "sRGB roundtrip", sRGBDecode(sRGBEncode(v)), v, 1e-12)
	}
}

func TestPQRoundTrip(t *testing.T) {
	// PQ code value 1.0 == 10000 cd/m^2 reference white.
	for _, v := range []float64{0.0, 0.01, 0.1, 0.5, 1.0} {
		approxEq(t, "PQ roundtrip", pqDecode(pqEncode(v)), v, 1e-12)
	}
	approxEq(t, "PQ peak", pqEncode(1.0), 1.0, 1e-12)
}

func TestD65AdaptationIsIdentity(t *testing.T) {
	a := adaptation(WP_D65, WP_D65)
	white := WP_D65.xyz()
	got := a.apply(white)
	approxEq(t, "D65 X", got.R, white.R, 1e-9)
	approxEq(t, "D65 Y", got.G, white.G, 1e-9)
	approxEq(t, "D65 Z", got.B, white.B, 1e-9)
}

func TestWhitePointPreservesLuminance(t *testing.T) {
	// Chromatic adaptation must not change the Y (luminance) component.
	src := xyY(WP_D65, 150)
	for _, wp := range []Chromaticity{WP_D93, WP_5K} {
		got := adaptation(WP_D65, wp).apply(src)
		approxEq(t, "adapted Y", got.G, 150, 1e-6)
	}
}

func TestDisplaySwitchKeepsExposure(t *testing.T) {
	scene := NewScene()
	scene.Exposure = 0.75
	exposed := scene.Exposed()

	displays := DefaultDisplays()
	first := make([]*Rendered, len(displays))
	for i, d := range displays {
		first[i] = d.Render(exposed, scene.W, scene.H)
	}
	// Re-render in shuffled order with the identical shared buffer.
	shuffled := []int{2, 0, 1}
	for _, i := range shuffled {
		again := displays[i].Render(exposed, scene.W, scene.H)
		if again.ClipPct != first[i].ClipPct {
			t.Fatalf("display %d clip pct changed: %.4f vs %.4f",
				i, again.ClipPct, first[i].ClipPct)
		}
		for p := range again.Preview {
			if again.Preview[p] != first[i].Preview[p] ||
				again.Mask[p] != first[i].Mask[p] {
				t.Fatalf("display %d pixel %d changed on re-render", i, p)
			}
		}
	}
}

func TestDimDisplayClipsMore(t *testing.T) {
	scene := NewScene()
	exposed := scene.Exposed()
	rates := map[string]float64{}
	for _, d := range DefaultDisplays() {
		rates[d.Name] = d.Render(exposed, scene.W, scene.H).ClipPct
	}
	if !(rates["LOW BRIGHTNESS"] > rates["SDR WIDE GAMUT"] &&
		rates["SDR WIDE GAMUT"] > rates["PQ HDR"]) {
		t.Fatalf("expected clip ordering dim>SDR>HDR, got %v", rates)
	}
}
