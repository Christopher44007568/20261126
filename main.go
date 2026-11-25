package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type scenario struct {
	name      string
	noise     float64
	glitch    bool
	seed      int64
	lambda    float64
	plotTitle string
}

func main() {
	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	scenarios := []scenario{
		{
			name:      "clean",
			noise:     0.5,
			glitch:    false,
			seed:      42,
			lambda:    2000,
			plotTitle: "SCENARIO A: WELL-BEHAVED SYNTHETIC CAMERA",
		},
		{
			name:      "glitched",
			noise:     1.5,
			glitch:    true,
			seed:      99,
			lambda:    2000,
			plotTitle: "SCENARIO B: MIDTONE CALIBRATION DEFECT (NON-MONOTONIC)",
		},
	}

	report := ""
	for _, sc := range scenarios {
		rep, err := runScenario(sc, outDir)
		if err != nil {
			fatal(err)
		}
		fmt.Println(rep.console())
		report += rep.console() + "\n"
	}

	reportPath := filepath.Join(outDir, "error_report.txt")
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s\n", reportPath)
}

type scenarioResult struct {
	sc  scenario
	rep ErrorReport
}

func (r scenarioResult) console() string {
	return fmt.Sprintf(
		"[%s] pixels=%d invalid=%d\n"+
			"  log RMSE (gauge aligned) = %.5f\n"+
			"  log RMSE (g(128)=0 fixed) = %.5f\n"+
			"  linear relative RMSE      = %.5f (%.2f%%)\n"+
			"  mean log bias             = %+.5f\n"+
			"  non-monotonic levels      = %d (red)\n"+
			"  extrapolated levels       = %d (orange)\n",
		r.sc.name, r.rep.N, r.rep.InvalidPixels,
		r.rep.LogRMSEAligned, r.rep.LogRMSEFixed,
		r.rep.LinearRelRMSE, 100*r.rep.LinearRelRMSE,
		r.rep.MeanLogBias,
		r.rep.NonMonotonicCnt, r.rep.ExtrapolatedCnt)
}

func runScenario(sc scenario, outDir string) (scenarioResult, error) {
	irradiance := SyntheticIrradiance(sc.seed)
	times := ExposureSet()
	logTimes := make([]float64, len(times))
	for i, t := range times {
		logTimes[i] = math.Log(t)
	}

	images := ShootExposures(irradiance, times, sc.noise, sc.glitch, sc.seed+1)

	g, err := RecoverResponse(images, logTimes, sc.lambda)
	if err != nil {
		return scenarioResult{}, err
	}
	gTrue := make([]float64, 256)
	for z := range gTrue {
		gTrue[z] = InverseTrueResponse(z)
	}

	nonMonotonic, extrapolated := AnalyzeResponse(g, images)
	logE, ok := ReconstructIrradiance(images, g, logTimes, nonMonotonic, extrapolated)
	rep := Evaluate(logE, ok, irradiance, nonMonotonic, extrapolated)

	strip := BuildExposureStrip(images, times)
	if err := strip.WritePPM(filepath.Join(outDir, sc.name+"_exposure_strip.ppm")); err != nil {
		return scenarioResult{}, err
	}
	plot := PlotResponse(g, gTrue, nonMonotonic, extrapolated, sc.plotTitle)
	if err := plot.WritePPM(filepath.Join(outDir, sc.name+"_response_curve.ppm")); err != nil {
		return scenarioResult{}, err
	}
	mapped := ReinhardTonemap(logE, ok)
	if err := mapped.WritePPM(filepath.Join(outDir, sc.name+"_reconstructed.ppm")); err != nil {
		return scenarioResult{}, err
	}

	return scenarioResult{sc: sc, rep: rep}, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
