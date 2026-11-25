package main

import "math"

// BuildExposureStrip concatenates all exposure images horizontally with white
// separators and per-exposure labels into one reviewable PPM.
func BuildExposureStrip(images []*Gray8, times []float64) *RGB8 {
	const gap = 6
	const labelH = 16
	w := len(images)*imgW + (len(images)-1)*gap
	h := imgH + labelH
	canvas := NewRGB8(w, h)
	fill(canvas, 255, 255, 255)

	for j, img := range images {
		ox := j * (imgW + gap)
		for y := 0; y < imgH; y++ {
			for x := 0; x < imgW; x++ {
				v := img.At(x, y)
				canvas.Set(ox+x, y+labelH, v, v, v)
			}
		}
		label := "E" + itoa(j+1) + " T=" + ftoa(times[j], 3) + "S"
		DrawText(canvas, ox+2, 2, label, 0, 0, 0)
	}
	return canvas
}

// ErrorReport quantifies reconstruction quality against the known synthetic
// irradiance. Because g() has an arbitrary gauge offset, both an offset-fixed
// log RMSE and a scale-invariant (aligned) log RMSE are reported.
type ErrorReport struct {
	N               int
	LogRMSEAligned  float64
	LogRMSEFixed    float64
	LinearRelRMSE   float64
	MeanLogBias     float64
	InvalidPixels   int
	NonMonotonicCnt int
	ExtrapolatedCnt int
}

func Evaluate(logE []float64, ok []bool, trueE []float64, nonMonotonic, extrapolated []bool) ErrorReport {
	var rep ErrorReport
	var diffs []float64
	for i, valid := range ok {
		if !valid {
			rep.InvalidPixels++
			continue
		}
		diffs = append(diffs, logE[i]-math.Log(trueE[i]))
	}
	rep.N = len(diffs)
	if rep.N == 0 {
		return rep
	}
	bias := 0.0
	for _, d := range diffs {
		bias += d
	}
	bias /= float64(rep.N)
	rep.MeanLogBias = bias

	var sseAligned, sseFixed, relSSE float64
	for i, d := range diffs {
		sseAligned += (d - bias) * (d - bias)
		sseFixed += d * d
		// linear-space relative error, aligned via the common scale factor
		recon := math.Exp(logE[i] - bias)
		rel := (recon - trueE[i]) / trueE[i]
		relSSE += rel * rel
	}
	rep.LogRMSEAligned = math.Sqrt(sseAligned / float64(rep.N))
	rep.LogRMSEFixed = math.Sqrt(sseFixed / float64(rep.N))
	rep.LinearRelRMSE = math.Sqrt(relSSE / float64(rep.N))

	for _, v := range nonMonotonic {
		if v {
			rep.NonMonotonicCnt++
		}
	}
	for _, v := range extrapolated {
		if v {
			rep.ExtrapolatedCnt++
		}
	}
	return rep
}
