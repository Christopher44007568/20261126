package main

import (
	"math"
	"testing"
)

func TestSyntheticLightsMoveInExpectedDirections(t *testing.T) {
	lights := lightSequence()
	sunset := lights[0].RGB
	fluorescent := lights[14].RGB
	shadow := lights[28].RGB

	if sunset.R <= sunset.G || sunset.G <= sunset.B {
		t.Fatalf("sunset should be warm: %+v", sunset)
	}
	if fluorescent.G <= fluorescent.R || fluorescent.G <= fluorescent.B {
		t.Fatalf("fluorescent spike should be green biased: %+v", fluorescent)
	}
	if shadow.B <= shadow.R {
		t.Fatalf("shadow should be cool: %+v", shadow)
	}
}

func TestFixedLightProducesNoPersistentDrift(t *testing.T) {
	scene, mask := buildScene()
	lights := lightSequence()
	frames := make([]*Image, len(lights))
	for i, light := range lights {
		frames[i] = renderFrame(scene, light.RGB)
	}
	report := analyzeFrames(frames, lights, mask)

	for _, segment := range [][2]int{{1, 9}, {15, 23}, {29, 39}} {
		base := report.Frames[segment[0]]
		for i := segment[0] + 1; i <= segment[1]; i++ {
			current := report.Frames[i]
			if gainsStep(base.GrayGains, current.GrayGains) > 1e-9 {
				t.Fatalf("gray world drifted in fixed light at frame %d", i)
			}
			if gainsStep(base.WhiteGains, current.WhiteGains) > 1e-9 {
				t.Fatalf("white point drifted in fixed light at frame %d", i)
			}
			if gainsStep(base.SmoothGains, current.SmoothGains) > 1e-9 {
				t.Fatalf("temporal smoothing drifted in fixed light at frame %d", i)
			}
		}
	}
}

func TestTemporalSmootherAllowsShortTransitionThenLocks(t *testing.T) {
	smoother := TemporalSmoother{}
	first := Gains{R: 0.5, G: 1, B: 2}
	second := Gains{R: 2, G: 1, B: 1.2}

	initial := smoother.Update(first, 0.4)
	if gainsStep(initial, first) > 1e-9 {
		t.Fatalf("initial smoother output should equal first estimate: %+v", initial)
	}

	transition := smoother.Update(second, 0.4)
	if transition.R <= first.R || transition.B >= first.B {
		t.Fatalf("transition should move partially toward new light: %+v", transition)
	}

	locked := smoother.Update(second, 0.4)
	if gainsStep(locked, second) > 1e-9 {
		t.Fatalf("stable estimate should lock without continued drift: %+v", locked)
	}
}

func TestCorrectedNeutralErrorsAreSmall(t *testing.T) {
	scene, mask := buildScene()
	lights := lightSequence()
	frames := make([]*Image, len(lights))
	for i, light := range lights {
		frames[i] = renderFrame(scene, light.RGB)
	}
	report := analyzeFrames(frames, lights, mask)

	for i, frame := range report.Frames {
		if frame.GrayError > 0.01 {
			t.Fatalf("gray world neutral RMSE too high at frame %d: %.5f", i, frame.GrayError)
		}
		whiteLimit := 0.012
		if (frame.Index >= 10 && frame.Index <= 13) || (frame.Index >= 24 && frame.Index <= 27) {
			whiteLimit = 0.015
		}
		if frame.WhiteError > whiteLimit {
			t.Fatalf("white point neutral RMSE too high at frame %d: %.5f", i, frame.WhiteError)
		}
		if frame.SmoothError > 0.15 {
			t.Fatalf("temporal neutral RMSE too high at frame %d: %.5f", i, frame.SmoothError)
		}
	}
}

func gainsStep(a, b Gains) float64 {
	return math.Max(math.Abs(a.R-b.R), math.Abs(a.B-b.B))
}
