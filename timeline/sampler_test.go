package timeline

import (
	"math"
	"testing"
)

func TestNoSamplesNodeAlwaysUsesOwnBindPose(t *testing.T) {
	bind := Pose{OffY: 7, Lean: -5, ArmL: -18, ArmR: 22}
	node := Node{ID: "buddy", Kind: "buddy", Bind: bind}

	for _, mode := range []ReadMode{ModeStepHold, ModeLinear} {
		for _, at := range []float64{-10, 0, 0.25, 0.38, 1.04, 1.5, 99} {
			got := node.Evaluate(at, mode)
			if got.Status != StatusStatic {
				t.Fatalf("%s @%v status=%q want static", mode, at, got.Status)
			}
			if got.Pose != bind {
				t.Fatalf("%s @%v pose=%+v want bind %+v", mode, at, got.Pose, bind)
			}
			if got.Lo != -1 || got.Hi != -1 || got.ShownKey != -1 {
				t.Fatalf("%s @%v indices=(%d,%d,%d), want all -1", mode, at, got.Lo, got.Hi, got.ShownKey)
			}
		}
	}
}

func TestStaticNodeDoesNotBorrowNeighborTrack(t *testing.T) {
	hero := Node{ID: "hero", Samples: []Sample{
		{Time: 0, Pose: Pose{ArmR: 100}},
		{Time: 1, Pose: Pose{ArmR: -100}},
	}}
	staticBind := Pose{ArmR: 22, Lean: -5}
	scene := &Scene{
		PrimaryID: "hero",
		Nodes: []Node{
			hero,
			{ID: "beacon", Bind: Pose{}},
			{ID: "buddy", Bind: staticBind},
		},
	}
	if err := scene.Prepare(); err != nil {
		t.Fatal(err)
	}
	spec := SheetSpec{PlaybackFPS: 6, FrameCount: 12, Retime: Retime{Rate: 1}, ReadMode: ModeLinear}
	frames := scene.BuildFrames(spec)
	if len(frames) != 12 {
		t.Fatalf("frames=%d", len(frames))
	}
	for _, frame := range frames {
		buddy, ok := frame.Node("buddy")
		if !ok {
			t.Fatal("buddy missing")
		}
		if buddy.Pose != staticBind || buddy.Status != StatusStatic {
			t.Fatalf("buddy borrowed neighbor data: %+v", buddy)
		}
	}
}

func TestStepHoldAndRangeClamps(t *testing.T) {
	node := Node{ID: "hero", Samples: []Sample{
		{Time: 0.25, Name: "idle", Pose: Pose{ArmR: 19}},
		{Time: 0.38, Name: "peak", Signature: true, Pose: Pose{ArmR: 118}},
		{Time: 0.44, Name: "settle", Pose: Pose{ArmR: 74}},
		{Time: 1.5, Name: "land", Pose: Pose{OffY: -20}},
	}}

	before := node.Evaluate(0, ModeStepHold)
	if before.Status != StatusBefore || before.ShownKey != 0 || before.Pose.ArmR != 19 {
		t.Fatalf("before clamp wrong: %+v", before)
	}

	held := node.Evaluate(0.40, ModeStepHold)
	if held.Status != StatusIn || held.Lo != 1 || held.ShownKey != 1 || held.Pose.ArmR != 118 {
		t.Fatalf("step hold should hold lower/left sample: %+v", held)
	}

	after := node.Evaluate(2, ModeLinear)
	if after.Status != StatusAfter || after.ShownKey != 3 || after.Pose.OffY != -20 {
		t.Fatalf("after clamp wrong: %+v", after)
	}
}

func TestLinearInterpolatesBracket(t *testing.T) {
	node := Node{ID: "hero", Samples: []Sample{
		{Time: 0, Pose: Pose{ArmR: 0}},
		{Time: 1, Pose: Pose{ArmR: 100}},
	}}
	got := node.Evaluate(0.25, ModeLinear)
	if got.Status != StatusIn || got.Lo != 0 || got.Hi != 1 {
		t.Fatalf("bracket wrong: %+v", got)
	}
	if math.Abs(got.Alpha-0.25) > 1e-12 || math.Abs(got.Pose.ArmR-25) > 1e-12 {
		t.Fatalf("interpolation wrong alpha=%v arm=%v", got.Alpha, got.Pose.ArmR)
	}
	if got.ShownKey != -1 {
		t.Fatalf("blended frame should not claim an exact key, got %d", got.ShownKey)
	}
}

func TestRetimeMapsSlowMotionClock(t *testing.T) {
	rt := Retime{Rate: 0.5}
	cases := map[float64]float64{0.5: 0.25, 5.0 / 6.0: 5.0 / 12.0, 3.0: 1.5}
	for playback, wantSource := range cases {
		if got := rt.SourceTime(playback); math.Abs(got-wantSource) > 1e-12 {
			t.Fatalf("source(%v)=%v want %v", playback, got, wantSource)
		}
	}
}

func TestFrameCountForRangeAddsTwoOutOfRangeFrames(t *testing.T) {
	if got := FrameCountForRange(6, 1, 1.5, 2); got != 12 {
		t.Fatalf("normal count=%d want 12", got)
	}
	if got := FrameCountForRange(6, 0.5, 1.5, 2); got != 21 {
		t.Fatalf("slow count=%d want 21", got)
	}
}
