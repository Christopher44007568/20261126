package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FrameReport struct {
	File           string `json:"file"`
	Pose           int    `json:"pose"`
	TLASRefits     int    `json:"tlas_refits"`
	UpdateDuration string `json:"update_duration"`
	RenderDuration string `json:"render_duration"`
}

type Report struct {
	Width         int           `json:"width"`
	Height        int           `json:"height"`
	BLASCount     int           `json:"blas_count"`
	BLASNames     []string      `json:"blas_names"`
	InstanceCount int           `json:"instance_count"`
	StaleBounds   string        `json:"stale_bounds_instance"`
	MissedUpdate  string        `json:"missed_update_instance"`
	Frames        []FrameReport `json:"frames"`
}

func main() {
	outDir := flag.String("out", "artifacts", "directory for pose_0.ppm ... pose_3.ppm")
	width := flag.Int("width", 800, "PPM width in pixels")
	height := flag.Int("height", 500, "PPM height in pixels")
	flag.Parse()

	if err := Generate(*outDir, *width, *height); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func Generate(outDir string, width, height int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	scene := NewScene(width, height)
	report := Report{
		Width:         width,
		Height:        height,
		BLASCount:     len(scene.BLAS),
		InstanceCount: len(scene.TLAS.Instances),
		StaleBounds:   scene.TLAS.Instances[StaleID].Name,
		MissedUpdate:  scene.TLAS.Instances[MissedID].Name,
	}
	for _, blas := range scene.BLAS {
		report.BLASNames = append(report.BLASNames, blas.Name)
	}

	for frame := 0; frame < 4; frame++ {
		updateStart := time.Now()
		if frame > 0 {
			scene.Advance(frame)
		}
		updateDuration := time.Since(updateStart)

		renderStart := time.Now()
		canvas := scene.Render(frame)
		renderDuration := time.Since(renderStart)

		fileName := fmt.Sprintf("pose_%d.ppm", frame)
		path := filepath.Join(outDir, fileName)
		if err := canvas.WritePPM(path); err != nil {
			return err
		}

		report.Frames = append(report.Frames, FrameReport{
			File:           filepath.ToSlash(path),
			Pose:           frame + 1,
			TLASRefits:     scene.TLAS.RefitCount,
			UpdateDuration: updateDuration.String(),
			RenderDuration: renderDuration.String(),
		})
		fmt.Printf("wrote %s (update=%s render=%s)\n", path, updateDuration, renderDuration)
	}

	reportPath := filepath.Join(outDir, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", reportPath)
	return nil
}
