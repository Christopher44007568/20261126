package cli

import (
	"math"

	"mipwrap/internal/mipmap"
)

const conservationTolerance = 1e-9

func buildReport(base *mipmap.RGBImage, normalStrength float64, wrap, naive, inside []*mipmap.RGBImage, energy []mipmap.EnergyReport) mipmap.Report {
	metrics := make([]mipmap.LevelMetric, 0, len(wrap)+len(naive)+len(inside))
	for index := range wrap {
		var energyReport *mipmap.EnergyReport
		if index < len(energy) {
			energyReport = &energy[index]
		}
		metrics = append(metrics, mipmap.BuildLevelMetric("wrap_normalized", index, wrap[index], base, wrap[index], energyReport))
		metrics = append(metrics, mipmap.BuildLevelMetric("naive_zero_pad", index, naive[index], base, wrap[index], nil))
		metrics = append(metrics, mipmap.BuildLevelMetric("inside_renormalized", index, inside[index], base, wrap[index], nil))
	}

	probeMetrics, maxProbeDeviation := constantProbe()
	maxWrapError := mipmap.MaxMeanColorDrift(wrap, base.Mean())

	return mipmap.Report{
		Width:                    base.Width,
		Height:                   base.Height,
		NormalStrength:           normalStrength,
		MaxAllowedMeanError:      conservationTolerance,
		Pass:                     maxWrapError <= conservationTolerance && maxProbeDeviation <= conservationTolerance,
		ColorSpace:               mipmap.SummarizeColor(base).ColorSpace,
		MaxWrapColorErrorWorking: maxWrapError,
		MaxProbeEdgeDeviation:    maxProbeDeviation,
		ColorMetrics:             metrics,
		ConstantProbe:            probeMetrics,
		EnergyReports:            energy,
	}
}

func constantProbe() ([]mipmap.ProbeMetric, float64) {
	const gray = 0.5
	base := mipmap.NewRGBImage(64, 64)
	base.FillGray(gray)

	wrapChain, _ := mipmap.GenerateWrapMipChain(base)
	naiveChain := mipmap.GenerateNaiveMipChain(base)
	insideChain := mipmap.GenerateInsideMipChain(base)

	result := make([]mipmap.ProbeMetric, 0, len(wrapChain)+len(naiveChain)+len(insideChain))
	maxWrapDeviation := 0.0
	appendMethod := func(method string, chain []*mipmap.RGBImage, trackWrap bool) {
		for index, level := range chain {
			border := mipmap.BorderMeanLuminance(level, 1)
			ratio := border / gray
			result = append(result, mipmap.ProbeMetric{
				Method:          method,
				Level:           index,
				Width:           level.Width,
				Height:          level.Height,
				Luminance:       level.MeanLuminance(),
				BorderLuminance: border,
				BrightnessRatio: ratio,
			})
			if trackWrap {
				deviation := math.Abs(ratio - 1)
				if deviation > maxWrapDeviation {
					maxWrapDeviation = deviation
				}
			}
		}
	}

	appendMethod("wrap_normalized", wrapChain, true)
	appendMethod("naive_zero_pad", naiveChain, false)
	appendMethod("inside_renormalized", insideChain, false)
	return result, maxWrapDeviation
}
