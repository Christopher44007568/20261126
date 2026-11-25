package report_test

import (
	"image"
	"testing"

	"autofocus/autofocus"
	"autofocus/report"
	"autofocus/sim"
)

func TestTenFramesAlwaysReturnsTen(t *testing.T) {
	frames := make([]report.Frame, 23)
	for i := range frames {
		frames[i].Frame = i
	}
	got := report.TenFrames(frames)
	if len(got) != 10 {
		t.Fatalf("got %d frames, want 10", len(got))
	}
	if got[0].Frame != 0 || got[9].Frame != 22 {
		t.Fatalf("endpoints not preserved: %d..%d", got[0].Frame, got[9].Frame)
	}
}

func TestContactSheetAndCurveHaveExpectedCanvas(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	img := scene.Render(40, scene.SubjectDepth)
	roi := image.Rect(80, 48, 240, 192)
	capture := report.Frame{
		Image: img, Frame: 40, Position: scene.SubjectDepth,
		State: autofocus.StateLocked, Sharpness: 0.08, ROI: roi,
	}
	frames := make([]report.Frame, 23)
	for i := range frames {
		frames[i] = capture
	}
	sheet := report.ContactSheet(frames, cfg)
	if sheet.Bounds().Dx() != 1272 || sheet.Bounds().Dy() != 508 {
		t.Fatalf("contact sheet = %v, want 1272x508", sheet.Bounds())
	}
	curve := report.SharpnessCurve([]autofocus.Sample{
		{Position: 40, Sharpness: 0.08, Phase: "coarse"},
		{Position: 44, Sharpness: 0.09, Phase: "fine"},
	}, cfg, autofocus.Snapshot{
		State: autofocus.StateLocked, BestPosition: 44, LockedPosition: 44,
		LockScore: 0.09, BestSharpness: 0.09, Moves: 4, Reversals: 2,
	})
	if curve.Bounds().Dx() != 1000 || curve.Bounds().Dy() != 600 {
		t.Fatalf("curve = %v, want 1000x600", curve.Bounds())
	}
	overlay := report.FocusOverlay(capture, cfg, autofocus.Snapshot{
		State: autofocus.StateLocked, Position: 44, BestPosition: 44,
		LockedPosition: 44, LockScore: 0.09,
	})
	if overlay.Bounds().Dx() != 1000 || overlay.Bounds().Dy() != 640 {
		t.Fatalf("overlay = %v, want 1000x640", overlay.Bounds())
	}
}
