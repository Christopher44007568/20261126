package aliaslab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunExperiment(outDir string, width, height int) (*ExperimentReport, error) {
	g, err := NewShowcase(width, height)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	correctIntervals := AnalyzeIntervals(g, ReadWriteLifetime)
	faultyIntervals := AnalyzeIntervals(g, WritesOnlyLifetime)
	plans := make([]Plan, 3)
	modes := []AliasMode{ModeNoReuse, ModeValid, ModeInvalid}
	for i, mode := range modes {
		plan, err := BuildPlan(g, mode)
		if err != nil {
			return nil, err
		}
		plans[i] = plan
	}
	runs := make([]*RunResult, 3)
	for i, plan := range plans {
		run, err := Run(g, plan)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", plan.Mode, err)
		}
		if run.Final == nil {
			return nil, fmt.Errorf("%s produced no final image", plan.Mode)
		}
		runs[i] = run
	}

	barriers := make([][]Barrier, 3)
	for i, plan := range plans {
		barriers[i] = BuildBarriers(g, plan)
	}

	contamination, polluted := FindFirstContamination(runs[0].Final, runs[2].Final)
	if !polluted {
		return nil, fmt.Errorf("error-reuse image unexpectedly matches reference")
	}
	highlighted := runs[2].Final.Clone()
	HighlightFirstRegion(highlighted, contamination)

	report := &ExperimentReport{
		Width:            width,
		Height:           height,
		Resources:        len(g.Resources),
		Passes:           passReports(g),
		CorrectIntervals: correctIntervals,
		FaultyIntervals:  faultyIntervals,
	}
	fileNames := []string{"no_reuse.ppm", "valid_reuse.ppm", "error_reuse.ppm"}
	images := []*Image{runs[0].Final, runs[1].Final, highlighted}
	for i, mode := range modes {
		path := filepath.Join(outDir, fileNames[i])
		if err := WritePPM(path, images[i]); err != nil {
			return nil, err
		}
		modeReport := newModeReport(plans[i], barriers[i], runs[i], fileNames[i])
		modeReport.ReferenceMatch = i != 2 && runs[0].Final.Equal(runs[i].Final)
		if i == 2 {
			modeReport.Contamination = &contamination
		}
		if i != 0 {
			first := FirstDivergingPass(runs[0], runs[i])
			if first >= 0 {
				modeReport.FirstDivergingPass = first + 1
				modeReport.FirstBadPassID = g.Passes[first].ID
				modeReport.FirstBadResource = g.Passes[first].Writes[0]
			}
		}
		if err := fillFileMetadata(&modeReport, path); err != nil {
			return nil, err
		}
		switch mode {
		case ModeNoReuse:
			report.NoReuse = modeReport
		case ModeValid:
			report.ValidReuse = modeReport
		case ModeInvalid:
			report.ErrorReuse = modeReport
		}
	}

	report.Assertions = buildAssertions(g, plans, runs, barriers, contamination)
	texts := map[string]string{
		"intervals.log": combinedIntervalLog(g, correctIntervals, faultyIntervals),
		"aliases.log":   combinedAliasLog(plans...),
		"barriers.log":  combinedBarrierLog(plans, barriers),
		"summary.txt":   summaryText(report),
	}
	for name, text := range texts {
		if err := writeTextAtomic(filepath.Join(outDir, name), text); err != nil {
			return nil, err
		}
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	if err := writeBytesAtomic(filepath.Join(outDir, "result.json"), payload); err != nil {
		return nil, err
	}
	if !report.allAssertionsPassed() {
		return report, fmt.Errorf("one or more experiment assertions failed; see result.json")
	}
	return report, nil
}

func newModeReport(plan Plan, barriers []Barrier, run *RunResult, ppm string) ModeReport {
	return ModeReport{
		Mode:        string(plan.Mode),
		Policy:      plan.Policy.String(),
		SlotCount:   plan.SlotCount,
		ReuseCount:  plan.ReuseCount,
		Allocations: plan.Allocations,
		Slots:       plan.Slots,
		Barriers:    barriers,
		Violations:  plan.Violations,
		PPM:         ppm,
	}
}

func fillFileMetadata(report *ModeReport, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	report.PPMBytes = int64(len(data))
	report.SHA256 = hex.EncodeToString(sum[:])
	return nil
}

func buildAssertions(g *Graph, plans []Plan, runs []*RunResult,
	barriers [][]Barrier, contamination Contamination) []Assertion {
	validAliasBarriers := 0
	for _, barrier := range barriers[1] {
		if strings.HasPrefix(barrier.Kind, "ALIAS_") {
			validAliasBarriers++
		}
	}
	firstBad := FirstDivergingPass(runs[0], runs[2])
	return []Assertion{
		{
			Name:   "no_reuse_has_one_slot_per_resource",
			Passed: plans[0].SlotCount == len(g.Resources) && plans[0].ReuseCount == 0,
			Detail: fmt.Sprintf("%d slots for %d resources",
				plans[0].SlotCount, len(g.Resources)),
		},
		{
			Name:   "valid_plan_has_no_lifetime_violation",
			Passed: len(plans[1].Violations) == 0,
			Detail: fmt.Sprintf("%d violations", len(plans[1].Violations)),
		},
		{
			Name:   "valid_plan_reuses_transients",
			Passed: plans[1].ReuseCount > 0 && plans[1].SlotCount < plans[0].SlotCount,
			Detail: fmt.Sprintf("%d resources -> %d slots",
				len(plans[1].Intervals), plans[1].SlotCount),
		},
		{
			Name:   "valid_pixels_equal_reference",
			Passed: runs[1].Final.Equal(runs[0].Final),
			Detail: "byte-for-byte equal RGB planes",
		},
		{
			Name:   "valid_alias_has_barriers",
			Passed: validAliasBarriers > 0,
			Detail: fmt.Sprintf("%d alias transition barriers", validAliasBarriers),
		},
		{
			Name:   "faulty_lifetime_is_detected",
			Passed: len(plans[2].Violations) > 0,
			Detail: fmt.Sprintf("%d read/alias violations", len(plans[2].Violations)),
		},
		{
			Name:   "faulty_image_diverges_after_reuse",
			Passed: firstBad > 0,
			Detail: fmt.Sprintf("first bad pass index %d", firstBad+1),
		},
		{
			Name:   "faulty_image_has_contamination_region",
			Passed: contamination.TotalPixels > 0 && contamination.RegionPixels > 0,
			Detail: fmt.Sprintf("%d differing px; first region %dx%d at (%d,%d)",
				contamination.TotalPixels, contamination.Bounds.Width(),
				contamination.Bounds.Height(), contamination.Bounds.MinX,
				contamination.Bounds.MinY),
		},
	}
}

func summaryText(report *ExperimentReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "transient alias experiment %dx%d\n\n", report.Width, report.Height)
	b.WriteString(modeTextReport(report.NoReuse))
	b.WriteByte('\n')
	b.WriteString(modeTextReport(report.ValidReuse))
	b.WriteByte('\n')
	b.WriteString(modeTextReport(report.ErrorReuse))
	b.WriteString("\nassertions:\n")
	for _, assertion := range report.Assertions {
		status := "PASS"
		if !assertion.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "  %s %-42s %s\n", status, assertion.Name, assertion.Detail)
	}
	return b.String()
}

func writeTextAtomic(path, text string) error {
	return writeBytesAtomic(path, []byte(text))
}

func writeBytesAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".text-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
