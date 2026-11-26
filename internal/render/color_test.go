package render

import (
	"math"
	"testing"
)

func TestLadderDensityOrdering(t *testing.T) {
	got := ladderDensities()
	if got["8-BIT"] != 2 {
		t.Fatalf("8-bit densities = %d, want 2", got["8-BIT"])
	}
	if got["10 TO 8"] < 4 {
		t.Fatalf("10-bit densities = %d, want at least 4", got["10 TO 8"])
	}
	if got["12 TO 8"] != 16 {
		t.Fatalf("12-bit densities = %d, want 16", got["12 TO 8"])
	}
	if got["12 TO 8"] <= got["10 TO 8"] {
		t.Fatalf("12-bit should expose more subcodes than 10-bit: %#v", got)
	}
}

func TestDitherPreservesAverage(t *testing.T) {
	p12 := Pipelines[2]
	for i := 0; i < 16; i++ {
		code := 100 + float64(i)/16
		var sum float64
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				sum += float64(DisplayCode(code/255, p12, x, y))
			}
		}
		avg := sum / (64 * 64)
		if math.Abs(avg-code) > 0.08 {
			t.Errorf("subcode %d: average %.4f, want %.4f", i, avg, code)
		}
	}
}

func TestHuePathHasNoWrapJump(t *testing.T) {
	left := HSV(359.75, 1, 1)
	right := HSV(0.25, 1, 1)
	h1, _ := RGBHue(left)
	h2, _ := RGBHue(right)
	if d := HueDistance(h1, h2); d > 1 {
		t.Fatalf("hue wrap jump = %.3f degrees", d)
	}
}

func TestHighlightClipsRGBInOrder(t *testing.T) {
	sc := Scenes[2]
	for _, p := range Pipelines {
		r := firstSaturated(sc, p, 0, metricW, metricH)
		g := firstSaturated(sc, p, 1, metricW, metricH)
		b := firstSaturated(sc, p, 2, metricW, metricH)
		if r < 0 || g < 0 || b < 0 || !(r <= g && g <= b) {
			t.Fatalf("%s: saturation order R/G/B = %d/%d/%d", p.Name, r, g, b)
		}
	}
}

func TestDisplayCodeStaysInRange(t *testing.T) {
	for _, p := range Pipelines {
		for _, v := range []float64{-1, 0, 0.5, 1, 1.7} {
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					code := DisplayCode(v, p, x, y)
					if code < 0 || code > 255 {
						t.Fatalf("%s: code(%v) = %d", p.Name, v, code)
					}
				}
			}
		}
	}
}

func TestHueSwitchIsContinuous(t *testing.T) {
	scene := Scenes[1]
	for _, pair := range [][2]Pipeline{
		{Pipelines[0], Pipelines[1]},
		{Pipelines[1], Pipelines[2]},
		{Pipelines[0], Pipelines[2]},
	} {
		hs := compareHues(scene, pair[0], pair[1])
		if hs.Samples == 0 {
			t.Fatalf("%s to %s: no hue samples", pair[0].Name, pair[1].Name)
		}
		if hs.MaxDeg > 1 {
			t.Errorf("%s to %s: hue max %.3f degrees", pair[0].Name, pair[1].Name, hs.MaxDeg)
		}
	}
}
