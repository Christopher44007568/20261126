package main

import (
	"math"
	"testing"
)

func testParams() Params {
	p := defaultParams()
	return p
}

// A stationary camera must render the rod perfectly straight for every path.
func TestRestPoseIsStraight(t *testing.T) {
	p := testParams()
	p.PanStart = 1e6 // pan never happens inside the sequence
	rest := p.RodX - p.RodHalf
	for _, s := range []Shutter{Rolling, Segmented, Global} {
		for f := 0; f < p.Frames; f++ {
			edge := rodLeftEdge(p, s, p.frameStart(f))
			if got := frameShear(edge); got > 1e-9 {
				t.Fatalf("%s frame %d rest shear = %v, want 0", s, f, got)
			}
			for y, v := range edge {
				if math.Abs(v-rest) > 1e-9 {
					t.Fatalf("%s frame %d row %d edge = %v, want %v", s, f, y, v, rest)
				}
			}
		}
	}
}

// After the pan finishes, every path must return to the straight rest pose.
func TestShapeRecoversAfterPan(t *testing.T) {
	p := testParams()
	for _, s := range []Shutter{Rolling, Segmented, Global} {
		edge := rodLeftEdge(p, s, p.frameStart(p.Frames-1))
		// Every row shares one edge value: the rod is straight again, even
		// though the whole scene has been translated by the completed pan.
		for y := 1; y < len(edge); y++ {
			if math.Abs(edge[y]-edge[0]) > 1e-6 {
				t.Fatalf("%s final frame not straight at row %d: %v vs %v",
					s, y, edge[y], edge[0])
			}
		}
	}
}

// Rolling and segmented exposure bend the rod during the pan, while the
// global shutter keeps every row at one instant and never bends it.
func TestShutterPathsDifferDuringPan(t *testing.T) {
	p := testParams()
	maxRolling, maxSegmented, maxGlobal := 0.0, 0.0, 0.0
	for f := 0; f < p.Frames; f++ {
		start := p.frameStart(f)
		if d := frameShear(rodLeftEdge(p, Rolling, start)); d > maxRolling {
			maxRolling = d
		}
		if d := frameShear(rodLeftEdge(p, Segmented, start)); d > maxSegmented {
			maxSegmented = d
		}
		if d := frameShear(rodLeftEdge(p, Global, start)); d > maxGlobal {
			maxGlobal = d
		}
	}
	if maxRolling < 10 {
		t.Fatalf("rolling max shear = %v, want pronounced bending", maxRolling)
	}
	if maxSegmented < 10 {
		t.Fatalf("segmented max shear = %v, want pronounced bending", maxSegmented)
	}
	if maxGlobal != 0 {
		t.Fatalf("global max shear = %v, want exactly 0", maxGlobal)
	}
}

// Segment borders must not introduce a row jump: adjacent edge samples stay
// within the bound imposed by one row of smooth rolling-shutter readout.
func TestSegmentedBordersDoNotJump(t *testing.T) {
	p := testParams()
	rowStep := p.PanVMax * p.ReadoutT / float64(p.Height-1)
	// The feather cross-fade spreads one block step over 2*FeatherRows rows.
	bound := rowStep * float64(p.BlockRows) / (2 * p.FeatherRows) * 1.01
	for f := 0; f < p.Frames; f++ {
		edge := rodLeftEdge(p, Segmented, p.frameStart(f))
		for y := 0; y+1 < p.Height; y++ {
			d := math.Abs(edge[y+1] - edge[y])
			if d > bound {
				t.Fatalf("frame %d rows %d->%d jump %v exceeds rolling bound %v",
					f, y, y+1, d, bound)
			}
		}
	}
}

// The segmented timing map is block-constant in the middle of each block and
// continuous across the feathered borders.
func TestSegmentedTimingMap(t *testing.T) {
	p := testParams()
	start := p.frameStart(4)
	mid := 1 // rows 0..3 are inside the constant region (feather starts at row 4)
	t0 := p.segmentCenter(start, mid)
	t1 := p.segmentCenter(start, mid+1)
	if math.Abs(t1-t0) > 1e-12 {
		t.Fatalf("block interior times differ: %v vs %v", t0, t1)
	}
	rowStep := p.ReadoutT / float64(p.Height-1)
	bound := rowStep * float64(p.BlockRows) / (2 * p.FeatherRows) * 1.01
	for y := 0; y+1 < p.Height; y++ {
		a := p.segmentCenter(start, y)
		b := p.segmentCenter(start, y+1)
		if math.Abs(b-a) > bound {
			t.Fatalf("timing discontinuity at row %d: %v -> %v", y, a, b)
		}
	}
}

func TestPanEndsAtRest(t *testing.T) {
	p := testParams()
	end := p.PanStart + 2*p.PanRamp + p.PanFlat
	if v := p.panVel(end + 0.5); v != 0 {
		t.Fatalf("velocity after pan = %v, want 0", v)
	}
	// Position is constant once the camera has come to rest.
	posA := p.panPos(end + 0.2)
	posB := p.panPos(end + 0.9)
	if math.Abs(posB-posA) > 1e-12 {
		t.Fatalf("rest position drifts: %v vs %v", posA, posB)
	}
}
