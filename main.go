package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func main() {
	p := defaultParams()
	out := "artifacts"
	if err := os.RemoveAll(out); err != nil {
		fatal(err)
	}

	shutters := []Shutter{Rolling, Segmented, Global}
	allEdges := make(map[Shutter][][]float64, len(shutters))
	allStats := make([]Stats, 0, len(shutters))
	allMats := make(map[Shutter][][]float64, len(shutters))
	sharedMax := 0.0

	for _, s := range shutters {
		dir := filepath.Join(out, s.String())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal(err)
		}
		edges := make([][]float64, p.Frames)
		for f := 0; f < p.Frames; f++ {
			start := p.frameStart(f)
			im := renderFrame(p, s, start)
			framePath := filepath.Join(dir, fmt.Sprintf("frame_%02d.ppm", f))
			if err := im.writePPM(framePath); err != nil {
				fatal(err)
			}
			edges[f] = rodLeftEdge(p, s, start)
		}
		allEdges[s] = edges
		allStats = append(allStats, computeStats(p, s, edges))
		var mat [][]float64
		mat, sharedMax = deviationMatrix(p, edges, sharedMax)
		allMats[s] = mat
		if err := writeEdgesCSV(p, edges, filepath.Join(dir, "edges.csv")); err != nil {
			fatal(err)
		}
	}
	for _, s := range shutters {
		dir := filepath.Join(out, s.String())
		if err := shearHeatmap(p, allMats[s], 2, sharedMax, filepath.Join(dir, "shear_heatmap.ppm")); err != nil {
			fatal(err)
		}
	}

	if err := edgeTrajectory(p, allEdges, filepath.Join(out, "edge_trajectory.ppm")); err != nil {
		fatal(err)
	}
	if err := writeReport(p, allStats, filepath.Join(out, "report.txt")); err != nil {
		fatal(err)
	}
	printReport(allStats)
}

// computeStats derives the quantitative invariants checked by the tests.
func computeStats(p Params, s Shutter, edges [][]float64) Stats {
	st := Stats{Shutter: s}
	rest := p.RodX - p.RodHalf
	for _, edge := range edges {
		shear := frameShear(edge)
		if shear > st.MaxShear {
			st.MaxShear = shear
		}
		if shear > 1.0 {
			st.BentFrames++
		}
		maxStep, borderStep, _ := adjacentStep(p, edge)
		if maxStep > st.MaxRowStep {
			st.MaxRowStep = maxStep
		}
		if borderStep > st.MaxBorderStep {
			st.MaxBorderStep = borderStep
		}
		for _, v := range edge {
			if d := math.Abs(v - rest); d > st.RestDeviation {
				st.RestDeviation = d
			}
		}
	}
	return st
}

func writeReport(p Params, stats []Stats, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "frames=%d %dx%d fps=%.0f readout=%.3fs exposure=%.3fs block=%d rows\n",
		p.Frames, p.Width, p.Height, p.FPS, p.ReadoutT, p.Exposure, p.BlockRows)
	for _, st := range stats {
		fmt.Fprintf(f,
			"%-9s maxShear=%7.2fpx maxRowStep=%6.3fpx maxBorderStep=%6.3fpx restDeviation=%7.3fpx bentFrames=%d\n",
			st.Shutter, st.MaxShear, st.MaxRowStep, st.MaxBorderStep, st.RestDeviation, st.BentFrames)
	}
	return nil
}

func printReport(stats []Stats) {
	for _, st := range stats {
		fmt.Printf("%-9s maxShear=%7.2fpx rowStep=%5.3fpx borderStep=%5.3fpx restDev=%6.3fpx bent=%d\n",
			st.Shutter, st.MaxShear, st.MaxRowStep, st.MaxBorderStep, st.RestDeviation, st.BentFrames)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
