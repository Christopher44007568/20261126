package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteArtifacts(report Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for i, frame := range report.Frames {
		name := filepath.Join(outDir, fmt.Sprintf("frame_%03d.ppm", i+1))
		if err := writePixelsPPM(name, frame.Width, frame.Height, frame.Pixels); err != nil {
			return err
		}
	}
	if err := writePPM(filepath.Join(outDir, "reuse_intervals.ppm"), RenderReuseTimeline(report)); err != nil {
		return err
	}
	if err := writePPM(filepath.Join(outDir, "dangling_references.ppm"), RenderReferenceGraph(report)); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "report.json"), report); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "dangling_references.dot"), []byte(renderDOT(report)), 0o644); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func renderDOT(report Report) string {
	var b strings.Builder
	b.WriteString("digraph acceleration_structure_references {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString(fmt.Sprintf(
		"  graph [label=\"generation refs and retirement queue; dangling refs=%d\", labelloc=t, fontname=\"monospace\"];\n",
		report.Metrics.DanglingRefs,
	))
	b.WriteString("  node [fontname=\"monospace\", fontsize=10];\n")
	b.WriteString("  edge [fontname=\"monospace\", fontsize=8];\n")
	b.WriteString("  subgraph cluster_queries { label=\"light-path query refs\"; color=\"#2a3b5c\";\n")
	for _, query := range report.Queries {
		b.WriteString(fmt.Sprintf(
			"    %q [shape=box, label=%q];\n",
			query.ID,
			fmt.Sprintf("%s\\nG%d ref t%d..t%d", query.ID, query.Generation, query.StartTick, query.DoneTick),
		))
	}
	b.WriteString("  }\n")
	b.WriteString("  subgraph cluster_generations { label=\"resource generations\"; color=\"#2a3b5c\";\n")
	for _, gen := range report.Generations {
		b.WriteString(fmt.Sprintf(
			"    %q [shape=box, style=\"rounded,filled\", fillcolor=\"#101a30\", color=%q, label=%q];\n",
			fmt.Sprintf("G%d", gen.ID),
			hexColor(generationColor(gen.ID)),
			fmt.Sprintf("G%d frame %d\\nbuild t%d retire t%d\\nrelease t%d last-ref t%d", gen.ID, gen.Frame, gen.BuiltAt, gen.RetiredAt, gen.ReleasedAt, gen.LastRefDoneAt),
		))
	}
	b.WriteString("  }\n")
	b.WriteString("  subgraph cluster_slots { label=\"shared slots\"; color=\"#2a3b5c\";\n")
	for slot := 0; slot < arenaCapacity; slot++ {
		b.WriteString(fmt.Sprintf("    %q [shape=box, label=%q];\n", fmt.Sprintf("S%d", slot), fmt.Sprintf("slot %d", slot)))
	}
	b.WriteString("  }\n")
	for _, query := range report.Queries {
		b.WriteString(fmt.Sprintf(
			"  %q -> %q [color=%q, label=%q];\n",
			query.ID, fmt.Sprintf("G%d", query.Generation), hexColor(generationColor(query.Generation)),
			fmt.Sprintf("done t%d", query.DoneTick),
		))
	}
	for _, gen := range report.Generations {
		for _, slot := range gen.Slots {
			b.WriteString(fmt.Sprintf(
				"  %q -> %q [style=dashed, color=%q, label=%q];\n",
				fmt.Sprintf("G%d", gen.ID), fmt.Sprintf("S%d", slot), hexColor(generationColor(gen.ID)),
				fmt.Sprintf("released t%d", gen.ReleasedAt),
			))
		}
	}
	for _, reuse := range report.Reuses {
		b.WriteString(fmt.Sprintf(
			"  %q -> %q [style=dotted, color=\"#ffd748\", label=%q];\n",
			fmt.Sprintf("G%d", reuse.From), fmt.Sprintf("G%d", reuse.To),
			fmt.Sprintf("reuse S%d t%d>=t%d", reuse.Slot, reuse.ReusedAt, reuse.LastFromRefDone),
		))
	}
	for _, dangling := range report.Dangling {
		b.WriteString(fmt.Sprintf(
			"  %q -> %q [color=\"#f84c5c\", penwidth=2, label=%q];\n",
			dangling.Query, fmt.Sprintf("S%d", dangling.Slot),
			fmt.Sprintf("DANGLING expected G%d got G%d at t%d", dangling.ExpectedGeneration, dangling.ActualGeneration, dangling.Tick),
		))
	}
	b.WriteString("}\n")
	return b.String()
}

func hexColor(c RGB) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func VerifyReport(report Report) error {
	if len(report.Frames) != 3 {
		return fmt.Errorf("expected 3 frames, got %d", len(report.Frames))
	}
	for _, frame := range report.Frames {
		if frame.GhostPixels != 0 {
			return fmt.Errorf("frame %d contains %d ghost pixels", frame.Frame, frame.GhostPixels)
		}
	}
	if report.Metrics.DanglingRefs != 0 {
		return fmt.Errorf("dangling references detected: %d", report.Metrics.DanglingRefs)
	}
	if report.Metrics.PrematureReleases != 0 {
		return fmt.Errorf("premature releases detected: %d", report.Metrics.PrematureReleases)
	}
	if report.Metrics.SafeReuseEvents != report.Metrics.ReuseEvents {
		return fmt.Errorf("unsafe slot reuse detected: %d/%d safe", report.Metrics.SafeReuseEvents, report.Metrics.ReuseEvents)
	}
	for _, gen := range report.Generations {
		if gen.ReleasedAt != noTick && gen.LastRefDoneAt != noTick && gen.ReleasedAt < gen.LastRefDoneAt {
			return fmt.Errorf("generation %d released at t%d before last query t%d", gen.ID, gen.ReleasedAt, gen.LastRefDoneAt)
		}
	}
	return nil
}
