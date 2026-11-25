package stab

import (
	"math"

	"deshake/internal/geom"
	"deshake/internal/synth"
)

// Metrics summarizes one stabilized frame for comparison.
type Metrics struct {
	Frame       int
	HoleFrac    float64
	WavyRMSE    float64 // content jelly: RMSE of vertical-line x vs row (px)
	WavyPP      float64 // content jelly: peak-to-peak line deviation (px)
	EdgeWavRMSE float64 // physical valid-boundary waviness after warp (px)
	EdgeWavPP   float64
	Edge        Margin
	Box         Rect
}

// HoleFraction returns the share of output pixels marked as holes.
func HoleFraction(fr *FrameResult) float64 {
	n := 0
	for _, h := range fr.Hole {
		if h {
			n++
		}
	}
	return float64(n) / float64(fr.W*fr.H)
}

// EdgeProfile returns, for every row, the leftmost and rightmost valid pixel,
// and for every column the topmost and bottommost valid pixel. Missing entries
// are -1. Wavy boundaries show up directly as variation of these profiles.
func EdgeProfile(fr *FrameResult) (left, right, top, bottom []int) {
	w, h := fr.W, fr.H
	left = make([]int, h)
	right = make([]int, h)
	top = make([]int, w)
	bottom = make([]int, w)
	for i := range left {
		left[i], right[i] = -1, -1
	}
	for i := range top {
		top[i], bottom[i] = -1, -1
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if fr.Valid[y*w+x] {
				if left[y] < 0 {
					left[y] = x
				}
				right[y] = x
			}
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			if fr.Valid[y*w+x] {
				if top[x] < 0 {
					top[x] = y
				}
				bottom[x] = y
			}
		}
	}
	return left, right, top, bottom
}

// BoundaryWaviness measures ripple on the valid-region edges via the spread of
// first differences. A straight edge (even slanted or clipped to the frame)
// has a constant step and scores ~0, while an undulating rolling-shutter edge
// has varying steps. RMSE of step deviations and peak-to-peak step spread are
// returned in pixels per row/column.
func BoundaryWaviness(fr *FrameResult) (rmse, pp float64) {
	left, right, top, bottom := EdgeProfile(fr)
	var allResid []float64
	globalPP := 0.0
	w, h := fr.W, fr.H
	// First differences along each genuine inward boundary. A straight edge has
	// near-constant steps; rolling-shutter ripple varies them. Border clips and
	// isolated jumps are excluded with a jump guard and percentile statistics.
	collect := func(profile []int, border int) {
		var steps []float64
		prev := -1
		for i, v := range profile {
			_ = i
			interior := v > 2 && v < border-3
			if v < 0 || !interior {
				prev = v
				continue
			}
			if prev > 2 && prev < border-3 {
				d := float64(v - prev)
				if abs64(d) < float64(border)/4 {
					steps = append(steps, d)
				}
			}
			prev = v
		}
		if len(steps) < 8 {
			return
		}
		sorted := append([]float64(nil), steps...)
		sortFloat(sorted)
		lo := sorted[len(sorted)/10]
		hi := sorted[(len(sorted)*9)/10]
		globalPP = math.Max(globalPP, hi-lo)
		med := sorted[len(sorted)/2]
		for _, d := range steps {
			allResid = append(allResid, d-med)
		}
	}
	collect(left, w)
	collect(right, w)
	collect(top, h)
	collect(bottom, h)
	if len(allResid) == 0 {
		return 0, 0
	}
	sse := 0.0
	for _, v := range allResid {
		sse += v * v
	}
	return math.Sqrt(sse / float64(len(allResid))), globalPP
}

func sortFloat(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// JellyWaviness quantifies residual rolling-shutter content deformation by
// projecting a world-space vertical line into the output and measuring its
// non-rigid curvature. A rigid (possibly tilted) straight line scores ~0;
// inter-row rolling-shutter shear produces a curved profile and scores high.
// The linear trend is removed first so a global tilt does not count as jelly.
func JellyWaviness(cam geom.Camera, f *synth.Frame, mode string) (rmse, pp float64) {
	w, h := f.W, f.H
	worldX := nearestGridX(cam, f, mode)
	xs := make([]float64, 0, h)
	for y := 0; y < h; y++ {
		var ux, uy float64
		switch mode {
		case "none":
			rowF := geom.RowReadTime(float64(y), h)
			hs := rowHomography(f, rowF)
			ux, uy = hs.Apply(worldX, 0)
		case "2d":
			hAct := cam.WorldToPixel(f.MidPose)
			hRef := cam.WorldToPixel(f.RefPose)
			ax, ay := hAct.Apply(0, 0)
			rx, ry := hRef.Apply(0, 0)
			rowF := geom.RowReadTime(float64(y), h)
			hs := rowHomography(f, rowF)
			px, py := hs.Apply(worldX, 0)
			ux, uy = px-(ax-rx), py-(ay-ry)
		default: // 3d: reproject every row onto the reference pose
			hRef := cam.WorldToPixel(f.RefPose)
			// Use the actual row pose to place the point, then undo geometry as
			// the 3D pipeline does: output = H_ref * H_row^-1 * source pixel.
			rowF := geom.RowReadTime(float64(y), h)
			hs := rowHomography(f, rowF)
			sx, sy := hs.Apply(worldX, 0)
			ux, uy = hRef.Mul(hs.Invert()).Apply(sx, sy)
			_ = uy
		}
		if ux >= 0 && ux <= float64(w) {
			xs = append(xs, ux)
		}
	}
	return lineDeviation(xs)
}

func nearestGridX(cam geom.Camera, f *synth.Frame, mode string) float64 {
	// Pick the vertical accent grid line nearest the view center.
	hRef := cam.WorldToPixel(f.RefPose)
	inv := hRef.Invert()
	wx, _ := inv.Apply(float64(f.W)/2, float64(f.H)/2)
	return math.Round(wx/300) * 300
}

// ImageJelly tracks the darkest vertical grid line in an output frame and
// measures its horizontal curvature across rows after removing linear tilt.
// It validates the geometry-based metric on the actual rendered pixels.
func ImageJelly(fr *FrameResult) (rmse, pp float64) {
	w, h := fr.W, fr.H
	// Estimate the line column at the central row from luminance minima.
	cy := h / 2
	type cand struct {
		x   int
		val float64
	}
	var cands []cand
	for x := 4; x < w-4; x++ {
		i := (cy*w + x) * 3
		lum := 0.3*fr.Pix[i] + 0.59*fr.Pix[i+1] + 0.11*fr.Pix[i+2]
		if lum < 0.25 && fr.Valid[cy*w+x] {
			cands = append(cands, cand{x, lum})
		}
	}
	if len(cands) < 1 {
		return 0, 0
	}
	best := cands[0]
	for _, c := range cands[1:] {
		if c.val < best.val {
			best = c
		}
	}
	xs := make([]float64, 0, h)
	for y := 0; y < h; y++ {
		// Search +-8 px around the reference column for the darkest valid pixel.
		lo := best.x - 8
		if lo < 1 {
			lo = 1
		}
		hi := best.x + 8
		if hi > w-2 {
			hi = w - 2
		}
		mx, mv := -1, 2.0
		for x := lo; x <= hi; x++ {
			if !fr.Valid[y*w+x] {
				continue
			}
			i := (y*w + x) * 3
			lum := 0.3*fr.Pix[i] + 0.59*fr.Pix[i+1] + 0.11*fr.Pix[i+2]
			if lum < mv {
				mv, mx = lum, x
			}
		}
		if mx >= 0 {
			xs = append(xs, float64(mx)+0.5)
		}
	}
	return lineDeviation(xs)
}

func lineDeviation(xs []float64) (rmse, pp float64) {
	n := float64(len(xs))
	if len(xs) < 8 {
		return 0, 0
	}
	var sx, sy, sxx, sxy float64
	for i, x := range xs {
		y := float64(i)
		sx += x
		sy += y
		sxx += y * y
		sxy += x * y
	}
	den := n*sxx - sy*sy
	if math.Abs(den) < 1e-9 {
		return 0, 0
	}
	slope := (n*sxy - sx*sy) / den
	intercept := (sx - slope*sy) / n
	mn, mx := math.MaxFloat64, -math.MaxFloat64
	sse := 0.0
	for i, x := range xs {
		res := x - (slope*float64(i) + intercept)
		sse += res * res
		if res < mn {
			mn = res
		}
		if res > mx {
			mx = res
		}
	}
	return math.Sqrt(sse / n), mx - mn
}
