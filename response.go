package main

import "fmt"

// Weight is the Debevec-Malik triangular hat function: trusted midtones,
// zero weight at the clipped/black extremes.
func Weight(z int) float64 {
	if z <= 127 {
		return float64(z)
	}
	return float64(255 - z)
}

// RecoverResponse solves for g[z] = ln(E*t) at each digital level z plus the
// per-sampled-pixel log irradiance ln(E_p), following Debevec & Malik (1997).
// A stratified subset of pixels keeps the sparse system small; it is solved
// directly via normal equations + partial-pivot elimination.
func RecoverResponse(images []*Gray8, logTimes []float64, lambda float64) ([]float64, error) {
	const n = 256
	sampled := samplePixels(images, 240)
	nP := len(sampled)
	unknowns := n + nP

	ata := make([][]float64, unknowns)
	for i := range ata {
		ata[i] = make([]float64, unknowns)
	}
	atb := make([]float64, unknowns)

	// Data equations: w*(g(z) - ln(E_p)) = w*ln(t_j).
	for si, p := range sampled {
		for j, img := range images {
			z := int(img.Pix[p])
			w := Weight(z)
			if w <= 0 {
				continue
			}
			addDataEquation(ata, atb, z, n+si, w, w*logTimes[j])
		}
	}

	// Smoothness: lambda * second differences on g.
	for z := 1; z < n-1; z++ {
		NormalEquations(ata, atb, []Triplet{
			{Col: z - 1, Val: lambda},
			{Col: z, Val: -2 * lambda},
			{Col: z + 1, Val: lambda},
		}, 0)
	}

	// Gauge anchor g(128) = 0.
	NormalEquations(ata, atb, []Triplet{{Col: 128, Val: lambda * 10}}, 0)

	x, err := SolveGaussian(ata, atb)
	if err != nil {
		return nil, fmt.Errorf("response solve: %w", err)
	}
	return x[:n], nil
}

func addDataEquation(ata [][]float64, atb []float64, colG, colE int, w, rhs float64) {
	NormalEquations(ata, atb, []Triplet{
		{Col: colG, Val: w},
		{Col: colE, Val: -w},
	}, rhs)
}

// samplePixels picks a deterministic, spatially stratified set spread over the
// image that exhibits usable midtone observations across exposures.
func samplePixels(images []*Gray8, count int) []int {
	pixels := len(images[0].Pix)
	step := pixels / count
	if step < 1 {
		step = 1
	}
	out := make([]int, 0, count)
	for p := 0; p < pixels && len(out) < count; p += step {
		score := 0.0
		for _, img := range images {
			score += Weight(int(img.Pix[p]))
		}
		if score > 0 {
			out = append(out, p)
		}
	}
	return out
}

// ReconstructIrradiance merges exposures into log irradiance per pixel.
// Levels flagged as extrapolated or non-monotonic are excluded so unreliable
// curve regions never contaminate the merge. ok is false when no trusted
// observation exists for a pixel.
func ReconstructIrradiance(images []*Gray8, g, logTimes []float64,
	nonMonotonic, extrapolated []bool) (logE []float64, ok []bool) {
	pixels := len(images[0].Pix)
	logE = make([]float64, pixels)
	ok = make([]bool, pixels)
	for p := 0; p < pixels; p++ {
		var wsum, acc float64
		for j, img := range images {
			z := int(img.Pix[p])
			if nonMonotonic[z] || extrapolated[z] {
				continue
			}
			w := Weight(z)
			if w <= 0 {
				continue
			}
			acc += w * (g[z] - logTimes[j])
			wsum += w
		}
		if wsum > 0 {
			logE[p] = acc / wsum
			ok[p] = true
		}
	}
	return logE, ok
}

// AnalyzeResponse flags non-monotonic levels (red) and levels only reachable by
// extrapolation beyond reliably observed data (orange).
func AnalyzeResponse(g []float64, images []*Gray8) (nonMonotonic, extrapolated []bool) {
	n := len(g)
	// Effective coverage weights observations by the hat function: clipped and
	// near-clipped levels contribute almost nothing to the solve.
	coverage := make([]float64, n)
	for _, img := range images {
		for _, z := range img.Pix {
			coverage[z] += Weight(int(z))
		}
	}

	nonMonotonic = make([]bool, n)
	for z := 1; z < n; z++ {
		if g[z] <= g[z-1] {
			nonMonotonic[z] = true
		}
	}

	// Levels below the reliability threshold are held only by the smoothness
	// prior. Flood fill from both ends so whole saturated tails are marked.
	const minCoverage = 50
	extrapolated = make([]bool, n)
	for z := 0; z < n && coverage[z] < minCoverage; z++ {
		extrapolated[z] = true
	}
	for z := n - 1; z >= 0 && coverage[z] < minCoverage; z-- {
		extrapolated[z] = true
	}
	return nonMonotonic, extrapolated
}
