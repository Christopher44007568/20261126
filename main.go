// Command animationtiming generates timing audit frame sheets with Go's
// standard library only.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"animationtiming/raster"
	"animationtiming/render"
	"animationtiming/report"
	"animationtiming/timeline"
)

func main() {
	outDir := "artifacts/golang"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := run(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	scene, err := timeline.DemoScene()
	if err != nil {
		return err
	}
	sheets, staticPolicy := scene.BuildAllSheets()

	expectedSkipped := map[string]bool{"01_normal_speed": true}
	canvases := make([]*raster.Canvas, 0, len(sheets))
	for _, check := range staticPolicy.Nodes {
		if !check.IdenticalToBindPose {
			return fmt.Errorf("static invariant failed for node %s", check.ID)
		}
	}
	for i := range sheets {
		_, shouldSkip := expectedSkipped[sheets[i].ID]
		skips := len(sheets[i].SkippedSignatureKeys) > 0
		if shouldSkip != skips {
			return fmt.Errorf("sheet %s skip state=%v, expected skip=%v", sheets[i].ID, skips, shouldSkip)
		}

		imageName := sheets[i].ID + ".png"
		sheets[i].Image = imageName
		canvas := render.RenderSheet(scene, sheets[i])
		if err := canvas.WritePNGFile(filepath.Join(outDir, imageName)); err != nil {
			return err
		}
		canvases = append(canvases, canvas)
		fmt.Printf("wrote %s (%dx%d)\n", filepath.Join(outDir, imageName), canvas.W, canvas.H)
	}

	audit := report.New(scene, sheets, staticPolicy)
	if err := report.WriteJSON(filepath.Join(outDir, "timeline_report.json"), audit); err != nil {
		return err
	}
	if err := report.WriteFile(filepath.Join(outDir, "index.html"), []byte(report.BuildHTML(audit))); err != nil {
		return err
	}
	if err := render.WriteCombinedPNG(filepath.Join(outDir, audit.CombinedImage), canvases); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", filepath.Join(outDir, audit.CombinedImage))

	fmt.Printf("wrote %s and index.html\n", filepath.Join(outDir, "timeline_report.json"))
	fmt.Println("timing invariants: normal=skips WAVE_PEAK/JUMP_PEAK, slow/interpolation=recover keys, static nodes=PASS")
	return nil
}
