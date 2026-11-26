package hdrlab

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// BuildReport assembles the JSON run report.
func BuildReport(outDir string, specs []FrameSpec, shifts []Shift, m *Metering, met metricsReport, acc acceptanceReport, files map[string]string) ([]byte, error) {
	rep := report{
		Tool:                "hdrlab",
		StandardLibraryOnly: true,
		OutputDirectory:     outDir,
		Metrics:             met,
		Acceptance:          acc,
		Files:               files,
	}
	for i, s := range specs {
		fr := frameReport{
			Name:        s.Name,
			EV:          s.EV,
			Exposure:    s.Exposure(),
			Width:       SceneW,
			Height:      SceneH,
			EstimatedDX: shifts[i].DX,
			EstimatedDY: shifts[i].DY,
			Confidence:  shifts[i].Confidence,
		}
		if i != 1 {
			dx, dy := s.DX, s.DY
			fr.TrueDX = &dx
			fr.TrueDY = &dy
		}
		rep.Frames = append(rep.Frames, fr)
		rep.Alignment.Shifts = append(rep.Alignment.Shifts, shiftReport{
			DX:         shifts[i].DX,
			DY:         shifts[i].DY,
			Score:      shifts[i].Score,
			Confidence: shifts[i].Confidence,
		})
	}
	rep.Alignment.Reference = 1
	rep.Metering = meteringReport{
		Columns:               m.Cols,
		Rows:                  m.Rows,
		CenterEV:              m.CenterEV,
		HighlightClipFraction: m.ClipFraction,
		DeepShadowFraction:    m.ShadowFraction,
	}
	for _, c := range m.Cells {
		rep.Metering.Cells = append(rep.Metering.Cells, cellReport{
			Bounds: boundsReport{
				Min: point{X: c.Bounds.MinX, Y: c.Bounds.MinY},
				Max: point{X: c.Bounds.MaxX, Y: c.Bounds.MaxY},
			},
			MeanLuma:       c.MeanLuma,
			EVRelative:     c.EVRelative,
			ClipFraction:   c.ClipFrac,
			ShadowFraction: c.ShadowFrac,
		})
	}
	return json.MarshalIndent(rep, "", "  ")
}

// WriteReport saves the JSON report into outDir.
func WriteReport(outDir string, data []byte) error {
	return os.WriteFile(filepath.Join(outDir, "hdr_report.json"), data, 0o644)
}
