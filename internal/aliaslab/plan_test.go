package aliaslab

import "testing"

func TestValidReuseIsLegalAndPixelExact(t *testing.T) {
	g, err := NewShowcase(96, 54)
	if err != nil {
		t.Fatal(err)
	}
	noPlan, err := BuildPlan(g, ModeNoReuse)
	if err != nil {
		t.Fatal(err)
	}
	validPlan, err := BuildPlan(g, ModeValid)
	if err != nil {
		t.Fatal(err)
	}
	if noPlan.SlotCount != 6 || noPlan.ReuseCount != 0 {
		t.Fatalf("no-reuse slots=%d reuse=%d", noPlan.SlotCount, noPlan.ReuseCount)
	}
	if validPlan.SlotCount != 3 || validPlan.ReuseCount != 3 {
		t.Fatalf("valid slots=%d reuse=%d", validPlan.SlotCount, validPlan.ReuseCount)
	}
	if len(validPlan.Violations) != 0 {
		t.Fatalf("valid plan has violations: %+v", validPlan.Violations)
	}
	for _, slot := range validPlan.Slots {
		for i := 0; i < len(slot.Users); i++ {
			for j := i + 1; j < len(slot.Users); j++ {
				if intervalsOverlap(slot.Users[i], slot.Users[j]) {
					t.Fatalf("slot %d has overlapping actual lifetimes: %+v %+v",
						slot.Slot, slot.Users[i], slot.Users[j])
				}
			}
		}
	}

	noRun, err := Run(g, noPlan)
	if err != nil {
		t.Fatal(err)
	}
	validRun, err := Run(g, validPlan)
	if err != nil {
		t.Fatal(err)
	}
	if !noRun.Final.Equal(validRun.Final) {
		t.Fatal("valid alias render is not pixel-identical to no-reuse reference")
	}
	if pass := FirstDivergingPass(noRun, validRun); pass != -1 {
		t.Fatalf("first diverging pass=%d, want none", pass+1)
	}

	barriers := BuildBarriers(g, validPlan)
	aliasBarriers := 0
	for _, barrier := range barriers {
		if barrier.Kind == "ALIAS_WAR" {
			aliasBarriers++
		}
	}
	if aliasBarriers != 3 {
		t.Fatalf("alias barriers=%d, want 3", aliasBarriers)
	}
}

func TestFaultyLifetimeProducesStaleColor(t *testing.T) {
	g, err := NewShowcase(96, 54)
	if err != nil {
		t.Fatal(err)
	}
	noPlan, err := BuildPlan(g, ModeNoReuse)
	if err != nil {
		t.Fatal(err)
	}
	badPlan, err := BuildPlan(g, ModeInvalid)
	if err != nil {
		t.Fatal(err)
	}
	if badPlan.SlotCount != 1 || badPlan.ReuseCount != 5 {
		t.Fatalf("bad slots=%d reuse=%d", badPlan.SlotCount, badPlan.ReuseCount)
	}
	if len(badPlan.Violations) == 0 {
		t.Fatal("writes-only lifetime did not trigger violations")
	}

	noRun, err := Run(g, noPlan)
	if err != nil {
		t.Fatal(err)
	}
	badRun, err := Run(g, badPlan)
	if err != nil {
		t.Fatal(err)
	}
	firstBad := FirstDivergingPass(noRun, badRun)
	if firstBad != 2 {
		t.Fatalf("first diverging pass=%d, want pass 3 (index 2)", firstBad+1)
	}
	if g.Passes[firstBad].ID != "clouds" {
		t.Fatalf("first bad pass=%q, want clouds", g.Passes[firstBad].ID)
	}
	contamination, ok := FindFirstContamination(noRun.Final, badRun.Final)
	if !ok || contamination.TotalPixels == 0 {
		t.Fatal("faulty render has no contamination")
	}
	if badRun.Final.At(contamination.FirstX, contamination.FirstY) ==
		noRun.Final.At(contamination.FirstX, contamination.FirstY) {
		t.Fatal("reported first pixel is not actually different")
	}

	barriers := BuildBarriers(g, badPlan)
	foundStaleRead := false
	for _, barrier := range barriers {
		if barrier.Kind == "ALIAS_RAW" && barrier.Pass == 2 &&
			barrier.FromResource == ResSun && barrier.ToResource == ResSky {
			foundStaleRead = true
		}
	}
	if !foundStaleRead {
		t.Fatal("barrier log does not expose pass-3 sky read from the sun-alias slot")
	}
}
