package stab

import (
	"math"
	"testing"

	"deshake/internal/geom"
	"deshake/internal/synth"
)

func testCamera() geom.Camera {
	return geom.Camera{W: 240, H: 180, Fx: 300, Fy: 300, Cx: 120, Cy: 90, Depth: 700}
}

func Test3DRemovesGeometryJelly(t *testing.T) {
	cam := testCamera()
	motion := synth.NewMotion(cam, 24)
	worst := 0.0
	for fi := 0; fi < motion.N; fi++ {
		f := motion.Capture(synth.New(), fi)
		none, _ := JellyWaviness(cam, f, "none")
		d3, _ := JellyWaviness(cam, f, "3d")
		if d3 > 1e-6 {
			t.Fatalf("frame %d: 3D geometric jelly = %v, want 0", fi, d3)
		}
		if none > worst {
			worst = none
		}
	}
	if worst < 0.5 {
		t.Fatalf("source jelly too small to be meaningful: %.3f px", worst)
	}
}

func TestHolesAreMarkedAndNotStretched(t *testing.T) {
	cam := testCamera()
	motion := synth.NewMotion(cam, 24)
	sawHole := false
	for fi := 0; fi < motion.N; fi++ {
		f := motion.Capture(synth.New(), fi)
		for _, fr := range []*FrameResult{Stabilize2D(cam, f), Stabilize3D(cam, f)} {
			for i, hole := range fr.Hole {
				if !hole {
					continue
				}
				sawHole = true
				if fr.Valid[i] {
					t.Fatal("hole pixel must also be invalid")
				}
				// Hole pixels keep zeroed color; no source/previous-frame value.
				j := i * 3
				if fr.Pix[j] != 0 || fr.Pix[j+1] != 0 || fr.Pix[j+2] != 0 {
					t.Fatal("hole pixel must not carry stretched pixel data")
				}
			}
		}
	}
	if !sawHole {
		t.Fatal("expected at least one hole across the sequence")
	}
}

func TestCropBoxContainsNoHoles(t *testing.T) {
	cam := testCamera()
	motion := synth.NewMotion(cam, 24)
	for fi := 0; fi < motion.N; fi++ {
		f := motion.Capture(synth.New(), fi)
		fr := Stabilize3D(cam, f)
		b := fr.Box
		for y := b.Y0; y < b.Y1; y++ {
			for x := b.X0; x < b.X1; x++ {
				if !fr.Valid[y*fr.W+x] {
					t.Fatalf("frame %d: crop box contains a hole at (%d,%d)", fi, x, y)
				}
			}
		}
	}
}

func TestGlobalCropIsContainedInEveryFrame(t *testing.T) {
	cam := testCamera()
	motion := synth.NewMotion(cam, 24)
	var boxes []Rect
	var results []*FrameResult
	for fi := 0; fi < motion.N; fi++ {
		f := motion.Capture(synth.New(), fi)
		fr := Stabilize3D(cam, f)
		boxes = append(boxes, fr.Box)
		results = append(results, fr)
	}
	g := intersectBoxesLocal(boxes)
	for fi, fr := range results {
		for y := g.Y0; y < g.Y1; y++ {
			for x := g.X0; x < g.X1; x++ {
				if !fr.Valid[y*fr.W+x] {
					t.Fatalf("global crop invalid in frame %d at (%d,%d)", fi, x, y)
				}
			}
		}
	}
}

func TestBoundaryWavinessStraightEdge(t *testing.T) {
	fr := newResult(40, 40)
	for y := 0; y < 40; y++ {
		for x := 5; x < 35; x++ {
			fr.Valid[y*40+x] = true
		}
	}
	rmse, _ := BoundaryWaviness(fr)
	if math.Abs(rmse) > 1e-9 {
		t.Fatalf("straight edge waviness = %v, want 0", rmse)
	}
}

// intersectBoxesLocal mirrors the main-package global crop intersection.
func intersectBoxesLocal(boxes []Rect) Rect {
	g := boxes[0]
	for _, b := range boxes[1:] {
		if b.X0 > g.X0 {
			g.X0 = b.X0
		}
		if b.Y0 > g.Y0 {
			g.Y0 = b.Y0
		}
		if b.X1 < g.X1 {
			g.X1 = b.X1
		}
		if b.Y1 < g.Y1 {
			g.Y1 = b.Y1
		}
	}
	return g
}
