// Package report writes machine-readable timing audits and a browsable HTML
// index. It uses only the Go standard library.
package report

import (
	"encoding/json"
	"os"
	"path/filepath"

	"animationtiming/timeline"
)

// Report is the JSON contract for a generated animation audit.
type Report struct {
	Title                  string                 `json:"title"`
	HeroSampleRangeSeconds [2]float64             `json:"hero_sample_range_seconds"`
	HeroTimeSamples        []timeline.Sample      `json:"hero_time_samples"`
	FrameRateNote          string                 `json:"frame_rate_note"`
	Dependencies           string                 `json:"dependencies"`
	CombinedImage          string                 `json:"combined_image"`
	Sheets                 []timeline.SheetResult `json:"sheets"`
	StaticPolicy           timeline.StaticPolicy  `json:"static_policy"`
}

// New builds a report from the embedded scene and evaluated sheets.
func New(scene *timeline.Scene, sheets []timeline.SheetResult, policy timeline.StaticPolicy) Report {
	hero, _ := scene.NodeByID(scene.PrimaryID)
	first := hero.Samples[0].Time
	last := hero.Samples[len(hero.Samples)-1].Time
	return Report{
		Title:                  scene.Title,
		HeroSampleRangeSeconds: [2]float64{first, last},
		HeroTimeSamples:        hero.Samples,
		FrameRateNote:          "playback clock is fixed at 6 fps; sample timestamps are irregular",
		Dependencies:           "Go standard library only",
		CombinedImage:          "animation_frame_sheets_all.png",
		Sheets:                 sheets,
		StaticPolicy:           policy,
	}
}

// WriteJSON creates parent directories and writes indented JSON.
func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// WriteFile writes an arbitrary text artifact.
func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
