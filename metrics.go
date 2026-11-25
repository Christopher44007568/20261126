package main

import "math"

type FrameReport struct {
	Index           int
	LightName       string
	TrueCCT         float64
	InputError      float64
	GrayGains       Gains
	WhiteGains      Gains
	SmoothGains     Gains
	GrayError       float64
	WhiteError      float64
	SmoothError     float64
	GrayCCT         float64
	WhiteCCT        float64
	SmoothCCT       float64
	CorrectedGray   *Image
	CorrectedWhite  *Image
	CorrectedSmooth *Image
}

type RunReport struct {
	Frames []FrameReport
}

func analyzeFrames(frames []*Image, lights []LightState, neutralMask []bool) RunReport {
	reports := make([]FrameReport, len(frames))
	smoother := TemporalSmoother{}

	for i, frame := range frames {
		grayGains := grayWorldGains(frame)
		whiteGains := whitePointGains(frame)
		smoothGains := smoother.Update(grayGains, 0.4)

		correctedGray := applyGains(frame, grayGains)
		correctedWhite := applyGains(frame, whiteGains)
		correctedSmooth := applyGains(frame, smoothGains)

		reports[i] = FrameReport{
			Index:           i,
			LightName:       lights[i].Name,
			TrueCCT:         lights[i].CCT,
			InputError:      neutralRMSE(frame, neutralMask),
			GrayGains:       grayGains,
			WhiteGains:      whiteGains,
			SmoothGains:     smoothGains,
			GrayError:       neutralRMSE(correctedGray, neutralMask),
			WhiteError:      neutralRMSE(correctedWhite, neutralMask),
			SmoothError:     neutralRMSE(correctedSmooth, neutralMask),
			GrayCCT:         cctFromGains(grayGains),
			WhiteCCT:        cctFromGains(whiteGains),
			SmoothCCT:       cctFromGains(smoothGains),
			CorrectedGray:   correctedGray,
			CorrectedWhite:  correctedWhite,
			CorrectedSmooth: correctedSmooth,
		}
	}
	return RunReport{Frames: reports}
}

func cctFromGains(g Gains) float64 {
	return cctFromRGB(RGB{R: 1 / g.R, G: 1, B: 1 / g.B})
}

func neutralRMSE(frame *Image, mask []bool) float64 {
	sum := 0.0
	count := 0.0
	for i, pixel := range frame.Pix {
		if !mask[i] {
			continue
		}
		average := (pixel.R + pixel.G + pixel.B) / 3
		deltaR := pixel.R - average
		deltaG := pixel.G - average
		deltaB := pixel.B - average
		sum += deltaR*deltaR + deltaG*deltaG + deltaB*deltaB
		count++
	}
	return math.Sqrt(sum / (3 * count))
}
