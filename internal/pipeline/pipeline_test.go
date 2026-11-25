package pipeline

import (
	"math"
	"testing"

	"aelab/internal/scene"
)

func TestFullTenSecondPipeline(t *testing.T) {
	result, err := Run(scene.RenderWorld(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FPS != scene.FPS || result.Duration != scene.Duration {
		t.Fatalf("unexpected timing: %d fps, %.1f s", result.FPS, result.Duration)
	}
	if len(result.Profiles) != len([]string{"rapid_breathing", "fast_smooth", "gentle_slow"}) {
		t.Fatalf("profile count = %d", len(result.Profiles))
	}

	for _, profile := range result.Profiles {
		if len(profile.Records) != scene.FrameCount {
			t.Fatalf("%s records = %d, want %d", profile.Profile.ID, len(profile.Records), scene.FrameCount)
		}
		for _, record := range profile.Records {
			if math.IsNaN(record.ActualEV) || math.IsInf(record.ActualEV, 0) {
				t.Fatalf("%s frame %d has invalid EV", profile.Profile.ID, record.Frame)
			}
			if math.Abs(record.EVStep) > profile.Profile.MaxEVStep+1e-9 {
				t.Fatalf("%s frame %d EV step %.4f exceeds cap %.4f", profile.Profile.ID, record.Frame, record.EVStep, profile.Profile.MaxEVStep)
			}
		}
	}

	fast := findProfile(t, result, "fast_smooth")
	if !fast.Metrics.PassesNoFlash {
		t.Fatalf("fast_smooth no-flash metrics failed: max luma jump %.4f, EV step %.4f", fast.Metrics.MaxLumaJump, fast.Metrics.MaxEVStep)
	}
	if fast.Metrics.MaxOvershootEV > 0.06 {
		t.Fatalf("fast_smooth overshoot %.3f EV is too large", fast.Metrics.MaxOvershootEV)
	}
	if fast.Metrics.SettleTimeS <= 0 || fast.Metrics.SettleTimeS > 2.2 {
		t.Fatalf("fast_smooth settle time = %.3f", fast.Metrics.SettleTimeS)
	}
	if fast.Metrics.WindowEdge < 0.025 || fast.Metrics.WindowClipHigh > 0.15 {
		t.Fatalf("window detail metrics outside range: edge %.4f clip %.3f", fast.Metrics.WindowEdge, fast.Metrics.WindowClipHigh)
	}
	if fast.Metrics.ShadowMean < 0.16 || fast.Metrics.ShadowStd < 0.06 {
		t.Fatalf("shadow detail metrics outside range: mean %.3f std %.3f", fast.Metrics.ShadowMean, fast.Metrics.ShadowStd)
	}
}

func findProfile(t *testing.T, result *Result, id string) ProfileResult {
	t.Helper()
	for _, profile := range result.Profiles {
		if profile.Profile.ID == id {
			return profile
		}
	}
	t.Fatalf("missing profile %s", id)
	return ProfileResult{}
}
