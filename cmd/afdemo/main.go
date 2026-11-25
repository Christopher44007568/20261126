// Command afdemo runs the autofocus engine against a deterministic synthetic
// scene and writes standalone PNG diagnostics. It does not use a camera or
// graphics SDK.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"autofocus/autofocus"
	"autofocus/report"
	"autofocus/sim"
)

type runSummary struct {
	GeneratedAt   time.Time          `json:"generated_at"`
	Scene         simSceneSummary    `json:"scene"`
	Config        autofocus.Config   `json:"config"`
	Snapshot      autofocus.Snapshot `json:"snapshot"`
	FramesRun     int                `json:"frames_run"`
	LockedOnFrame int                `json:"locked_on_frame"`
	HuntingTrips  int                `json:"hunting_trips_after_lock"`
	SharpnessPeak float64            `json:"sharpness_peak"`
	Thumbnails    []thumbnailSummary `json:"thumbnails"`
	Artifacts     map[string]string  `json:"artifacts"`
}

type simSceneSummary struct {
	Width        int `json:"width"`
	Height       int `json:"height"`
	SubjectDepth int `json:"subject_depth"`
	EnterFrame   int `json:"enter_frame"`
	SettledFrame int `json:"settled_frame"`
}

type thumbnailSummary struct {
	Index     int             `json:"index"`
	Frame     int             `json:"frame"`
	Position  int             `json:"position"`
	State     autofocus.State `json:"state"`
	Sharpness float64         `json:"sharpness"`
}

func main() {
	outDir := flag.String("out", "artifacts", "output directory for PNG and JSON diagnostics")
	totalFrames := flag.Int("frames", 160, "number of synthetic frames to run")
	flag.Parse()
	if *totalFrames < 2 {
		fatal(fmt.Errorf("frames must be at least 2"))
	}

	cfg := autofocus.DefaultConfig()
	scene := sim.DefaultScene()
	engine, err := autofocus.New(cfg)
	if err != nil {
		fatal(err)
	}

	frames := make([]report.Frame, 0, *totalFrames)
	lens := cfg.MinPosition
	lockedOnFrame := 0
	previousState := autofocus.StateWaiting
	for frame := 0; frame < *totalFrames; frame++ {
		img := scene.Render(frame, lens)
		decision := engine.Update(img, lens)
		frames = append(frames, report.Frame{
			Image:     img,
			Frame:     frame,
			Position:  lens,
			State:     decision.State,
			Sharpness: decision.Measurement.Sharpness,
			ROI:       decision.Measurement.ROI,
			MoveTo:    decision.MoveTo,
		})
		if decision.State == autofocus.StateLocked && previousState != autofocus.StateLocked {
			lockedOnFrame = frame
		}
		previousState = decision.State
		if decision.MoveTo >= 0 {
			lens = decision.MoveTo
		}
	}

	snap := engine.Snapshot()
	if snap.State != autofocus.StateLocked {
		fatal(fmt.Errorf("final state %s is not locked", snap.State))
	}
	if snap.MovesAfterLock != 0 {
		fatal(fmt.Errorf("locked lens hunted for %d moves", snap.MovesAfterLock))
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	last := frames[len(frames)-1]
	artifacts := map[string]string{
		"ten_frames":      filepath.Join(*outDir, "ten_frames.png"),
		"sharpness_curve": filepath.Join(*outDir, "sharpness_curve.png"),
		"focus_overlay":   filepath.Join(*outDir, "focus_overlay.png"),
		"report_json":     filepath.Join(*outDir, "af_report.json"),
	}
	if err := report.SavePNG(artifacts["ten_frames"], report.ContactSheet(frames, cfg)); err != nil {
		fatal(err)
	}
	if err := report.SavePNG(artifacts["sharpness_curve"], report.SharpnessCurve(snap.Samples, cfg, snap)); err != nil {
		fatal(err)
	}
	if err := report.SavePNG(artifacts["focus_overlay"], report.FocusOverlay(last, cfg, snap)); err != nil {
		fatal(err)
	}

	summary := runSummary{
		GeneratedAt: time.Now().UTC(),
		Scene: simSceneSummary{
			Width: scene.Width, Height: scene.Height,
			SubjectDepth: scene.SubjectDepth,
			EnterFrame:   scene.EnterFrame,
			SettledFrame: scene.SettledFrame,
		},
		Config:        cfg,
		Snapshot:      snap,
		FramesRun:     *totalFrames,
		LockedOnFrame: lockedOnFrame,
		HuntingTrips:  snap.MovesAfterLock,
		SharpnessPeak: snap.BestSharpness,
		Artifacts:     artifacts,
	}
	for i, frame := range report.TenFrames(frames) {
		summary.Thumbnails = append(summary.Thumbnails, thumbnailSummary{
			Index: i, Frame: frame.Frame, Position: frame.Position,
			State: frame.State, Sharpness: frame.Sharpness,
		})
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(artifacts["report_json"], data, 0o644); err != nil {
		fatal(err)
	}

	fmt.Printf("locked=%d true_depth=%d frame=%d moves=%d reversals=%d after_lock=%d\n",
		snap.LockedPosition, scene.SubjectDepth, lockedOnFrame, snap.Moves, snap.Reversals, snap.MovesAfterLock)
	fmt.Printf("wrote %s, %s, %s, %s\n",
		artifacts["ten_frames"], artifacts["sharpness_curve"],
		artifacts["focus_overlay"], artifacts["report_json"])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "afdemo:", err)
	os.Exit(1)
}
