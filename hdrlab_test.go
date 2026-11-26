package hdrlab

import "testing"

// TestPipelineAcceptance runs the full bracket -> align -> merge pipeline
// and asserts the acceptance criteria from the run report.
func TestPipelineAcceptance(t *testing.T) {
	specs := BracketSpecs()
	scene := RenderScene()
	frames := make([]*Img, len(specs))
	ts := make([]float64, len(specs))
	for i, s := range specs {
		frames[i] = RenderFrame(scene, s, uint64(1000+i))
		ts[i] = s.Exposure()
	}

	const ref = 1
	shifts := make([]Shift, len(specs))
	shifts[ref] = Shift{Confidence: 1}
	for i := range specs {
		if i == ref {
			continue
		}
		shifts[i] = EstimateShift(frames[ref], frames[i], ts[ref], ts[i], Margin)
		if shifts[i].DX != specs[i].DX || shifts[i].DY != specs[i].DY {
			t.Errorf("frame %s: estimated shift (%d,%d), want (%d,%d)",
				specs[i].Name, shifts[i].DX, shifts[i].DY, specs[i].DX, specs[i].DY)
		}
	}

	merged := MergeHDR(frames, ts, shifts, ref)
	preview := Tonemap(merged.HDR)
	direct := AverageFrames(frames, shifts, false)
	aligned := AverageFrames(frames, shifts, true)

	bill := Rect{
		MinX: Billboard.MinX - Margin + 12,
		MinY: Billboard.MinY - Margin + 12,
		MaxX: Billboard.MaxX - Margin - 12,
		MaxY: Billboard.MaxY - Margin - 12,
	}
	eHDR := EdgeEnergy(preview, bill)
	eBase := (EdgeEnergy(direct, bill) + EdgeEnergy(aligned, bill)) / 2
	if ratio := eHDR / eBase; ratio <= 1.15 {
		t.Errorf("highlight edge ratio %.3f, want > 1.15 (hdr %.4f baselines %.4f)", ratio, eHDR, eBase)
	}

	carRow := CarY - Margin
	refCrop := AverageFrames(frames[ref:ref+1], shifts[ref:ref+1], false)
	nHDR := CountEdges(preview, carRow, 0.10)
	nRef := CountEdges(refCrop, carRow, 0.10)
	if nHDR > nRef+2 {
		t.Errorf("moving object left multiple edges: hdr row has %d edges, reference %d", nHDR, nRef)
	}

	if merged.MotionFraction < 0.01 || merged.MotionFraction > 0.2 {
		t.Errorf("motion fraction %.4f outside plausible range", merged.MotionFraction)
	}

	clusters := FindClusters(merged.Ghost, OutW, OutH, 100)
	if len(clusters) < 3 {
		t.Errorf("expected >=3 ghost clusters (three car positions), got %d", len(clusters))
	}
}
