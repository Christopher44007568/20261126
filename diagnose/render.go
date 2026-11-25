package diagnose

import (
	"fmt"
	"os"
	"strings"
)

// WriteMarkdown saves a human-readable per-level color mismatch report.
func WriteMarkdown(path string, report *Report) error {
	var builder strings.Builder
	builder.WriteString("# Normal Map sRGB Mislabel Diagnostic\n\n")
	if report.Mismatch {
		builder.WriteString("- Result: **stable color mismatch detected**\n")
		builder.WriteString(fmt.Sprintf("- First failing level: **%d**\n", report.FirstFailingLevel))
		builder.WriteString(fmt.Sprintf("- Strongest level: **%d**\n", report.WorstLevel))
		if err := report.LocateFirstMismatch(); err != nil {
			builder.WriteString(fmt.Sprintf("- Location: %s\n", err.Error()))
		}
	} else {
		builder.WriteString("- Result: no mismatch above threshold\n")
	}
	builder.WriteString("\n| Level | Size | Max Δ | Mean Δ | Changed texels | Bounds | Delta fingerprint |\n")
	builder.WriteString("|---:|---|---:|---:|---:|---|---|\n")
	for _, level := range report.Levels {
		bounds := "-"
		if level.ChangedBounds != nil {
			b := level.ChangedBounds
			bounds = fmt.Sprintf("(%d,%d)-(%d,%d)", b.MinX, b.MinY, b.MaxX, b.MaxY)
		}
		builder.WriteString(fmt.Sprintf(
			"| %d | %dx%d | %.9f | %.9f | %d/%d (%.2f%%) | %s | `%s` |\n",
			level.Level, level.Width, level.Height, level.MaxAbsDelta, level.MeanAbsRGB,
			level.ChangedTexels, level.TotalTexels, level.ChangedRatio*100,
			bounds, level.DeltaFingerprint[:12]))
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}
