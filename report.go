package main

import (
	"encoding/json"
	"os"
)

type VisualReport struct {
	Comparison       string `json:"comparison"`
	Amplification    int    `json:"amplification"`
	SourceRetained   bool   `json:"sourceRetained"`
	TargetRetained   bool   `json:"targetRetained"`
	ErrorMapRendered bool   `json:"errorMapRendered"`
}

type Report struct {
	Schema      string        `json:"schema"`
	Title       string        `json:"title"`
	Status      string        `json:"status"`
	AveragePass bool          `json:"averagePass"`
	LocalPass   bool          `json:"localPass"`
	Visual      VisualReport  `json:"visual"`
	Thresholds  Thresholds    `json:"thresholds"`
	Summary     Summary       `json:"summary"`
	Groups      []GroupResult `json:"groups"`
	Patches     []PatchResult `json:"patches"`
}

func buildReport(m *Manifest, a *Analysis, comparison string, amplification int) Report {
	return Report{
		Schema:      "patchcheck/v1",
		Title:       m.Title,
		Status:      a.Summary.Status,
		AveragePass: a.Summary.AveragePass,
		LocalPass:   a.Summary.LocalPass,
		Visual: VisualReport{
			Comparison:       comparison,
			Amplification:    amplification,
			SourceRetained:   true,
			TargetRetained:   true,
			ErrorMapRendered: true,
		},
		Thresholds: m.Thresholds,
		Summary:    a.Summary,
		Groups:     a.Groups,
		Patches:    a.Patches,
	}
}

func writeReport(path string, report Report) error {
	if err := ensureParent(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
