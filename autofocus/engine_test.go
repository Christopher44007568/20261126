package autofocus_test

import (
	"image"
	"image/color"
	"testing"

	"autofocus/autofocus"
	"autofocus/sim"
)

func TestSharpnessPeaksNearSubjectDepth(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	var bestPos int
	var bestScore float64
	for pos := cfg.MinPosition; pos <= cfg.MaxPosition; pos++ {
		frame := scene.Render(40, pos)
		m, _ := autofocus.CalculateMeasurement(frame, cfg.ROI, nil)
		if m.Sharpness > bestScore {
			bestScore = m.Sharpness
			bestPos = pos
		}
	}
	if delta := bestPos - scene.SubjectDepth; delta < -2 || delta > 2 {
		t.Fatalf("sharpness peak at %d, want near %d", bestPos, scene.SubjectDepth)
	}
}

func TestSimulatedAcquisitionLocksWithoutHunting(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	engine, err := autofocus.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	lens := cfg.MinPosition
	var lockFrame int
	firstMoveFrame := -1
	var movesAfterLock int
	for frame := 0; frame < 120; frame++ {
		img := scene.Render(frame, lens)
		decision := engine.Update(img, lens)
		if decision.MoveTo >= 0 {
			if firstMoveFrame < 0 {
				firstMoveFrame = frame
			}
			if lockFrame > 0 && decision.MoveTo != lens {
				movesAfterLock++
			}
			lens = decision.MoveTo
		}
		if decision.State == autofocus.StateLocked && lockFrame == 0 {
			lockFrame = frame + 1
		}
	}

	snapshot := engine.Snapshot()
	if snapshot.State != autofocus.StateLocked {
		t.Fatalf("final state = %s, want locked", snapshot.State)
	}
	if delta := snapshot.LockedPosition - scene.SubjectDepth; delta < -2 || delta > 2 {
		t.Fatalf("locked at %d, want near %d", snapshot.LockedPosition, scene.SubjectDepth)
	}
	if movesAfterLock != 0 || snapshot.MovesAfterLock != 0 {
		t.Fatalf("hunting after lock: decisions=%d snapshot=%d", movesAfterLock, snapshot.MovesAfterLock)
	}
	if snapshot.Reversals > 2 {
		t.Fatalf("too many direction reversals: %d", snapshot.Reversals)
	}
	if firstMoveFrame <= scene.EnterFrame {
		t.Fatalf("search began on frame %d before subject entered on frame %d", firstMoveFrame, scene.EnterFrame)
	}
}

func TestWaitsForSubjectMotion(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	engine, err := autofocus.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	blank := image.NewRGBA(image.Rect(0, 0, scene.Width, scene.Height))
	moved := false
	for frame := 0; frame < 3; frame++ {
		decision := engine.Update(blank, cfg.MinPosition)
		if decision.MoveTo >= 0 {
			moved = true
		}
	}
	if moved || engine.Snapshot().State != autofocus.StateWaiting {
		t.Fatal("engine searched without a subject")
	}
}

func TestExternalSubjectNotificationStartsSearch(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	engine, err := autofocus.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	still := scene.Render(40, cfg.MinPosition)
	if decision := engine.Update(still, cfg.MinPosition); decision.MoveTo >= 0 {
		t.Fatal("engine moved before subject notification")
	}
	engine.NotifySubject(true)
	decision := engine.Update(still, cfg.MinPosition)
	if !decision.SubjectPresent || decision.State != autofocus.StateCoarse {
		t.Fatalf("external subject did not arm search: state=%s present=%v",
			decision.State, decision.SubjectPresent)
	}
}

func TestHysteresisIgnoresModerateSharpnessDrop(t *testing.T) {
	scene := sim.DefaultScene()
	cfg := autofocus.DefaultConfig()
	engine, err := autofocus.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	lens := cfg.MinPosition
	for frame := 0; frame < 120; frame++ {
		decision := engine.Update(scene.Render(frame, lens), lens)
		if decision.MoveTo >= 0 {
			lens = decision.MoveTo
		}
	}
	snapshot := engine.Snapshot()
	if snapshot.State != autofocus.StateLocked {
		t.Fatalf("setup did not lock: %s", snapshot.State)
	}

	dimmed := dimImage(scene.Render(120, lens), 0.95)
	for i := 0; i < 20; i++ {
		decision := engine.Update(dimmed, lens)
		if decision.State != autofocus.StateLocked || decision.MoveTo >= 0 {
			t.Fatalf("moderate drop broke hysteresis on sample %d: state=%s move=%d",
				i, decision.State, decision.MoveTo)
		}
	}
}

func dimImage(src image.Image, factor float64) *image.Gray {
	b := src.Bounds()
	dst := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := src.At(x, y).RGBA()
			l := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 257
			dst.SetGray(x-b.Min.X, y-b.Min.Y, color.Gray{Y: uint8(l * factor)})
		}
	}
	return dst
}
