package aliaslab

import (
	"fmt"
	"strings"
)

type PassReport struct {
	Index  int          `json:"index"`
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Reads  []ResourceID `json:"reads"`
	Writes []ResourceID `json:"writes"`
}

type Assertion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type ModeReport struct {
	Mode               string               `json:"mode"`
	Policy             string               `json:"policy"`
	SlotCount          int                  `json:"slot_count"`
	ReuseCount         int                  `json:"reuse_count"`
	Allocations        []ResourceAllocation `json:"allocations"`
	Slots              []SlotAllocation     `json:"slots"`
	Barriers           []Barrier            `json:"barriers"`
	Violations         []Violation          `json:"violations"`
	PPM                string               `json:"ppm"`
	PPMBytes           int64                `json:"ppm_bytes"`
	SHA256             string               `json:"sha256"`
	ReferenceMatch     bool                 `json:"reference_match"`
	FirstDivergingPass int                  `json:"first_diverging_pass,omitempty"`
	FirstBadPassID     string               `json:"first_bad_pass_id,omitempty"`
	FirstBadResource   ResourceID           `json:"first_bad_resource,omitempty"`
	Contamination      *Contamination       `json:"contamination,omitempty"`
}

type ExperimentReport struct {
	Width            int          `json:"width"`
	Height           int          `json:"height"`
	Resources        int          `json:"resources"`
	Passes           []PassReport `json:"passes"`
	CorrectIntervals []Interval   `json:"correct_intervals"`
	FaultyIntervals  []Interval   `json:"faulty_intervals"`
	NoReuse          ModeReport   `json:"no_reuse"`
	ValidReuse       ModeReport   `json:"valid_reuse"`
	ErrorReuse       ModeReport   `json:"error_reuse"`
	Assertions       []Assertion  `json:"assertions"`
}

func (r *ExperimentReport) allAssertionsPassed() bool {
	for _, assertion := range r.Assertions {
		if !assertion.Passed {
			return false
		}
	}
	return true
}

func passReports(g *Graph) []PassReport {
	out := make([]PassReport, 0, len(g.Passes))
	for i, p := range g.Passes {
		out = append(out, PassReport{
			Index:  i + 1,
			ID:     p.ID,
			Name:   p.Name,
			Reads:  append([]ResourceID(nil), p.Reads...),
			Writes: append([]ResourceID(nil), p.Writes...),
		})
	}
	return out
}

func combinedAliasLog(plans ...Plan) string {
	var b strings.Builder
	for i, plan := range plans {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "=== %s ===\n", plan.Mode)
		b.WriteString(plan.AliasLogText())
	}
	return b.String()
}

func combinedBarrierLog(plans []Plan, barriers [][]Barrier) string {
	var b strings.Builder
	for i, plan := range plans {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "=== %s ===\n", plan.Mode)
		b.WriteString(BarrierLogText(plan.Mode, barriers[i]))
	}
	return b.String()
}

func combinedIntervalLog(g *Graph, correct, faulty []Interval) string {
	var b strings.Builder
	b.WriteString("=== correct lifetime: max(last read, last write) ===\n")
	b.WriteString(g.AccessLogText(correct))
	b.WriteString("\n=== faulty lifetime: last write only (injected bug) ===\n")
	b.WriteString(g.AccessLogText(faulty))
	return b.String()
}

func modeTextReport(report ModeReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: slots=%d reused=%d barriers=%d violations=%d\n",
		report.Mode, report.SlotCount, report.ReuseCount,
		len(report.Barriers), len(report.Violations))
	if report.ReferenceMatch {
		b.WriteString("  reference_match=true (every pixel identical)\n")
	}
	if report.FirstDivergingPass > 0 {
		fmt.Fprintf(&b, "  first divergence: pass %d %s -> %s\n",
			report.FirstDivergingPass, report.FirstBadPassID, report.FirstBadResource)
	}
	if report.Contamination != nil {
		c := report.Contamination
		fmt.Fprintf(&b, "  first contamination: pixel (%d,%d), bounds (%d,%d)-(%d,%d), %d px\n",
			c.FirstX, c.FirstY, c.Bounds.MinX, c.Bounds.MinY,
			c.Bounds.MaxX, c.Bounds.MaxY, c.RegionPixels)
	}
	return b.String()
}
