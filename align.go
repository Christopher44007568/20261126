package hdrlab

import (
	"math"
	"sort"
)

// Shift is an integer translation that maps a frame onto the reference.
type Shift struct {
	DX, DY     int
	Score      float64
	Confidence float64
}

// lin converts a gamma-encoded sample back to linear.
func lin(z float64) float64 { return math.Pow(Clamp01(z), 2.2) }

// EstimateShift searches integer translations in [-radius,radius] for the
// one minimizing exposure-compensated luma disagreement with ref. The
// per-pixel difference is capped so the moving object cannot dominate.
// The returned shift maps reference coordinates onto candidate samples:
// aligned[x,y] = cand[x-dx, y-dy].
func EstimateShift(ref, cand *Img, tRef, tCand float64, radius int) Shift {
	best := Shift{}
	bestScore := math.Inf(1)
	scores := make([]float64, 0, (2*radius+1)*(2*radius+1))
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			s := shiftScore(ref, cand, tRef, tCand, dx, dy, radius)
			scores = append(scores, s)
			if s < bestScore {
				bestScore = s
				best = Shift{DX: dx, DY: dy, Score: s}
			}
		}
	}
	sorted := append([]float64(nil), scores...)
	sort.Float64s(sorted)
	med := sorted[len(sorted)/2]
	if med > 0 {
		best.Confidence = Clamp01(1 - bestScore/med)
	}
	return best
}

func shiftScore(ref, cand *Img, tRef, tCand float64, dx, dy, margin int) float64 {
	var sum float64
	var n int
	for y := margin; y < ref.H-margin; y++ {
		for x := margin; x < ref.W-margin; x++ {
			zr := ref.Luma(x, y)
			zc := cand.Luma(x-dx, y-dy)
			if zr > SatClip || zc > SatClip {
				continue // saturated pixels carry no alignment signal
			}
			d := lin(zr)/tRef - lin(zc)/tCand
			if d < 0 {
				d = -d
			}
			if d > 0.25 {
				d = 0.25 // robust cap against moving-object outliers
			}
			sum += d
			n++
		}
	}
	if n == 0 {
		return math.Inf(1)
	}
	return sum / float64(n)
}
