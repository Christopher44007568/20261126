package ae

import (
	"math"
	"sort"

	"aelab/internal/scene"
)

type Metering struct {
	GeometricLum  float64
	ArithmeticLum float64
	Percentile95  float64
	ShadowLum     float64
	HighlightLum  float64
}

func Measure(frame *scene.HDR) Metering {
	weightedLogSum := 0.0
	weightSum := 0.0
	arithSum := 0.0
	lums := make([]float64, 0, frame.Width*frame.Height)

	for y := 0; y < frame.Height; y++ {
		ny := (float64(y)/float64(frame.Height-1) - 0.5) / 0.58
		for x := 0; x < frame.Width; x++ {
			nx := (float64(x)/float64(frame.Width-1) - 0.5) / 0.62
			lum := math.Max(scene.Luminance(frame.At(x, y)), 1e-5)
			weight := 0.32 + 0.68*math.Exp(-1.55*(nx*nx+ny*ny))
			weightedLogSum += weight * math.Log(lum)
			weightSum += weight
			arithSum += lum
			lums = append(lums, lum)
		}
	}

	sort.Float64s(lums)
	p95 := lums[int(float64(len(lums)-1)*0.95)]
	p10 := lums[int(float64(len(lums)-1)*0.10)]
	p99 := lums[int(float64(len(lums)-1)*0.99)]

	return Metering{
		GeometricLum:  math.Exp(weightedLogSum / weightSum),
		ArithmeticLum: arithSum / float64(len(lums)),
		Percentile95:  p95,
		ShadowLum:     p10,
		HighlightLum:  p99,
	}
}

func TargetEV(m Metering) float64 {
	const middleGray = 0.175
	gain := middleGray / math.Max(m.GeometricLum, 0.008)
	if highlightGain := 2.25 / math.Max(m.Percentile95, 0.01); gain > highlightGain {
		gain = highlightGain
	}
	gain = math.Max(0.07, math.Min(9.0, gain))
	return math.Log2(gain)
}
