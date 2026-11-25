package main

import (
	"math"
	"testing"
)

func TestWeightHat(t *testing.T) {
	cases := map[int]float64{0: 0, 1: 1, 127: 127, 128: 127, 255: 0, 200: 55}
	for z, want := range cases {
		if got := Weight(z); got != want {
			t.Errorf("Weight(%d) = %v, want %v", z, got, want)
		}
	}
}

func TestSolveGaussian(t *testing.T) {
	a := [][]float64{
		{2, 1, -1},
		{-3, -1, 2},
		{-2, 1, 2},
	}
	b := []float64{8, -11, -3}
	got, err := SolveGaussian(a, b)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{2, 3, -1}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("x[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSolveGaussianSingular(t *testing.T) {
	a := [][]float64{{1, 2}, {2, 4}}
	if _, err := SolveGaussian(a, []float64{3, 6}); err == nil {
		t.Fatal("expected singular matrix error")
	}
}

func buildPipeline(t *testing.T, noise float64, glitch bool, seed int64) (
	irradiance, logTimes []float64, images []*Gray8) {
	t.Helper()
	irradiance = SyntheticIrradiance(seed)
	times := ExposureSet()
	logTimes = make([]float64, len(times))
	for i, tv := range times {
		logTimes[i] = math.Log(tv)
	}
	images = ShootExposures(irradiance, times, noise, glitch, seed+1)
	return irradiance, logTimes, images
}

func TestCleanRecoveryAccuracy(t *testing.T) {
	irradiance, logTimes, images := buildPipeline(t, 0.5, false, 42)
	g, err := RecoverResponse(images, logTimes, 2000)
	if err != nil {
		t.Fatal(err)
	}
	nonMonotonic, extrapolated := AnalyzeResponse(g, images)
	reds := 0
	for _, v := range nonMonotonic {
		if v {
			reds++
		}
	}
	if reds != 0 {
		t.Fatalf("clean response has %d non-monotonic levels, want 0", reds)
	}
	logE, ok := ReconstructIrradiance(images, g, logTimes, nonMonotonic, extrapolated)
	rep := Evaluate(logE, ok, irradiance, nonMonotonic, extrapolated)
	if rep.LogRMSEAligned > 0.05 {
		t.Fatalf("aligned log RMSE = %v, want < 0.05", rep.LogRMSEAligned)
	}
	if rep.LinearRelRMSE > 0.05 {
		t.Fatalf("linear relative RMSE = %v, want < 0.05", rep.LinearRelRMSE)
	}
}

func TestGlitchDetected(t *testing.T) {
	_, logTimes, images := buildPipeline(t, 1.5, true, 99)
	g, err := RecoverResponse(images, logTimes, 2000)
	if err != nil {
		t.Fatal(err)
	}
	nonMonotonic, _ := AnalyzeResponse(g, images)
	// The injected defect remaps the 148..168 band downward, so at least one
	// level inside that neighborhood must be flagged non-monotonic.
	hit := false
	for z := 140; z <= 180; z++ {
		if nonMonotonic[z] {
			hit = true
		}
	}
	if !hit {
		t.Fatal("no non-monotonic level detected in the defect band")
	}
}

func TestExtrapolationFlagsSaturatedTails(t *testing.T) {
	_, logTimes, images := buildPipeline(t, 0, false, 7)
	g, err := RecoverResponse(images, logTimes, 2000)
	if err != nil {
		t.Fatal(err)
	}
	_, extrapolated := AnalyzeResponse(g, images)
	if !extrapolated[0] || !extrapolated[255] {
		t.Fatal("clipped endpoints 0 and 255 must be marked extrapolated")
	}
}

func TestOracleReconstruction(t *testing.T) {
	irradiance, logTimes, images := buildPipeline(t, 0, false, 42)
	gTrue := make([]float64, 256)
	for z := range gTrue {
		gTrue[z] = InverseTrueResponse(z)
	}
	none := make([]bool, 256)
	logE, ok := ReconstructIrradiance(images, gTrue, logTimes, none, none)
	rep := Evaluate(logE, ok, irradiance, none, none)
	if rep.LogRMSEAligned > 0.01 {
		t.Fatalf("oracle aligned log RMSE = %v, want < 0.01", rep.LogRMSEAligned)
	}
}
