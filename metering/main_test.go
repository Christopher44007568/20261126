package main

import (
	"math"
	"testing"
)

func uniformFrame(w, h int, lum float64) *Frame {
	f := NewFrame(w, h)
	for i := 0; i < w*h; i++ {
		f.set(i, lum, lum, lum)
	}
	return f
}

func TestWeightsNormalize(t *testing.T) {
	f := uniformFrame(64, 48, 0.3)
	for _, m := range []Mode{ModeFullFrame, ModeCenterWeighted, ModeHighlightProtect, ModeSpot} {
		w := Weights(f, m, 10, 10)
		sum := 0.0
		for _, v := range w {
			sum += v
		}
		if math.Abs(sum-1) > 1e-9 {
			t.Errorf("mode %v: weights sum to %v, want 1", m, sum)
		}
	}
}

func TestWeightedPercentileUniform(t *testing.T) {
	f := uniformFrame(32, 32, 0.42)
	w := Weights(f, ModeFullFrame, 0, 0)
	p := WeightedPercentile(f, w, 0.9)
	if math.Abs(p-0.42) > 0.02 {
		t.Errorf("percentile = %v, want ~0.42", p)
	}
}

func TestSmootherLimitsJumps(t *testing.T) {
	s := NewSmoother(0.25, 0.35)
	s.Update(0)
	prev := 0.0
	// A metering point hopping between dark wall and blown window.
	raw := []float64{3.5, -3.5, 3.5, -3.5, 3.5, -3.5, 3.5, -3.5}
	for i, r := range raw {
		got := s.Update(r)
		if d := math.Abs(got - prev); d > 0.35+1e-9 {
			t.Fatalf("step %d: jump %.2f EV exceeds slew limit", i, d)
		}
		prev = got
	}
	if math.Abs(prev) > 1.0 {
		t.Errorf("smoothed EV = %.2f, should stay near 0 under oscillation", prev)
	}
}

func TestSmootherConverges(t *testing.T) {
	s := NewSmoother(0.25, 0.35)
	s.Update(0)
	got := 0.0
	for i := 0; i < 60; i++ {
		got = s.Update(-1.5)
	}
	if math.Abs(got-(-1.5)) > 0.05 {
		t.Errorf("smoothed EV = %.2f, want convergence to -1.5", got)
	}
}

func TestHighlightProtectNeverCrushesSubject(t *testing.T) {
	// Extreme case: a tiny sun-like region in a dim room with a
	// mid-tone subject at the center.
	f := uniformFrame(100, 100, 0.05)
	for y := 45; y < 55; y++ {
		for x := 45; x < 55; x++ {
			f.set(y*100+x, 0.35, 0.32, 0.28) // subject
		}
	}
	for y := 5; y < 8; y++ {
		for x := 80; x < 83; x++ {
			f.set(y*100+x, 60, 60, 60) // sun
		}
	}
	ev, _ := HighlightProtectEV(f)
	cw := Weights(f, ModeCenterWeighted, 0, 0)
	evCenter := EVForLuminance(Meter(f, cw))
	floor := evCenter - MaxHighlightUnderEV
	if ev < floor-1e-9 {
		t.Errorf("highlight EV %.2f below floor %.2f: subject would be crushed", ev, floor)
	}
	// Subject luminance after exposure must stay visible (not a black blob).
	gain := math.Exp2(ev)
	if got := 0.35 * gain; got < 0.02 {
		t.Errorf("subject luminance after exposure = %.3f, too dark", got)
	}
}

func TestHighlightProtectTargetsHighlights(t *testing.T) {
	scene := DefaultScene()
	f := scene.Render(0.5)
	ev, _ := HighlightProtectEV(f)
	// Window luminance ~3.0 should land near HighlightTarget.
	got := 3.0 * math.Exp2(ev)
	if got < 0.6 || got > 1.1 {
		t.Errorf("window lands at %.2f linear, want near %.2f", got, HighlightTarget)
	}
}
