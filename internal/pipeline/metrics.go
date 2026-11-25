package pipeline

import (
	"math"

	"aelab/internal/scene"
)

func calculateMetrics(result ProfileResult, initialEV, finalTarget float64) Metrics {
	records := result.Records
	m := Metrics{InitialEV: initialEV, FinalTargetEV: finalTarget}
	if len(records) == 0 {
		return m
	}

	m.FinalActualEV = records[len(records)-1].ActualEV
	settleThreshold := 0.055
	lastViolation := -1
	crossingSide := 0
	lumaDirection := 0

	for i, r := range records {
		m.MaxEVStep = math.Max(m.MaxEVStep, math.Abs(r.EVStep))
		m.MaxLumaJump = math.Max(m.MaxLumaJump, math.Abs(r.LumaDelta))
		m.MeanAbsLumaJump += math.Abs(r.LumaDelta)

		if r.TimeS >= TransitionEnd {
			deviation := r.ActualEV - finalTarget
			m.MaxAbsDeviationEV = math.Max(m.MaxAbsDeviationEV, math.Abs(deviation))
			if r.OvershootEV > 0 {
				m.MaxOvershootEV = math.Max(m.MaxOvershootEV, r.OvershootEV)
			}
			if math.Abs(deviation) > settleThreshold {
				lastViolation = i
			}
			const crossingThreshold = 0.018
			if deviation > crossingThreshold {
				if crossingSide < 0 {
					m.ZeroCrossings++
				}
				crossingSide = 1
			} else if deviation < -crossingThreshold {
				if crossingSide > 0 {
					m.ZeroCrossings++
				}
				crossingSide = -1
			}
		}

		if r.TimeS >= TransitionStart {
			const reversalThreshold = 0.0035
			if r.LumaDelta > reversalThreshold {
				if lumaDirection < 0 {
					m.LumaReversals++
				}
				lumaDirection = 1
			} else if r.LumaDelta < -reversalThreshold {
				if lumaDirection > 0 {
					m.LumaReversals++
				}
				lumaDirection = -1
			}
		}
	}
	m.MeanAbsLumaJump /= float64(len(records))

	if lastViolation < 0 {
		m.SettleTimeS = TransitionEnd
	} else if lastViolation == len(records)-1 {
		m.SettleTimeS = -1
	} else {
		m.SettleTimeS = records[lastViolation+1].TimeS
	}

	lastStart := len(records) - scene.FPS
	if lastStart < 0 {
		lastStart = 0
	}
	count := 0
	for i := lastStart; i < len(records); i++ {
		r := records[i]
		m.WindowMean += r.WindowMean
		m.WindowEdge += r.WindowEdge
		m.WindowClipHigh += r.WindowClipHigh
		m.ShadowMean += r.ShadowMean
		m.ShadowStd += r.ShadowStd
		m.ShadowEdge += r.ShadowEdge
		count++
	}
	if count > 0 {
		m.WindowMean /= float64(count)
		m.WindowEdge /= float64(count)
		m.WindowClipHigh /= float64(count)
		m.ShadowMean /= float64(count)
		m.ShadowStd /= float64(count)
		m.ShadowEdge /= float64(count)
	}

	m.PassesNoFlash = m.MaxLumaJump <= 0.032 && m.MaxEVStep <= 0.26
	m.PassesSmoothConverge = m.SettleTimeS > 0 && m.SettleTimeS <= 3.2 && m.MaxOvershootEV <= 0.06 && m.ZeroCrossings <= 1
	return m
}
