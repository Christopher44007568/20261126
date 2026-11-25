package mipmap

import "math"

type ColorSummary struct {
	ColorSpace         string     `json:"color_space"`
	MeanWorkingRGB     [3]float64 `json:"mean_working_rgb"`
	LuminanceWorking   float64    `json:"luminance_working"`
	MeanOutput8        [3]float64 `json:"mean_output_8"`
	MeanLinearLightRGB [3]float64 `json:"mean_linear_light_rgb"`
	LuminanceLinear    float64    `json:"luminance_linear_light"`
}

func SummarizeColor(img *RGBImage) ColorSummary {
	mean := img.Mean()
	var outputSum [3]float64
	var linearSum [3]float64
	pixels := float64(img.Width * img.Height)
	for i := 0; i < len(img.Pix); i += 3 {
		for channel := 0; channel < 3; channel++ {
			value := clamp01(img.Pix[i+channel])
			if img.Linear {
				outputSum[channel] += 255 * linearToSRGB(value)
				linearSum[channel] += value
			} else {
				outputSum[channel] += 255 * value
				linearSum[channel] += sRGBToLinear(value)
			}
		}
	}

	outputMean := [3]float64{outputSum[0] / pixels, outputSum[1] / pixels, outputSum[2] / pixels}
	linearMean := [3]float64{linearSum[0] / pixels, linearSum[1] / pixels, linearSum[2] / pixels}
	space := "srgb_encoded"
	if img.Linear {
		space = "linear_light"
	}
	return ColorSummary{
		ColorSpace:         space,
		MeanWorkingRGB:     mean,
		LuminanceWorking:   Luminance(mean),
		MeanOutput8:        outputMean,
		MeanLinearLightRGB: linearMean,
		LuminanceLinear:    Luminance(linearMean),
	}
}

// AverageColorError returns RMS distance between two average linear colors.
func AverageColorError(img *RGBImage, target [3]float64) float64 {
	mean := img.Mean()
	var sum float64
	for channel := 0; channel < 3; channel++ {
		delta := mean[channel] - target[channel]
		sum += delta * delta
	}
	return math.Sqrt(sum / 3.0)
}

func AverageOutputColorError(img *RGBImage, target [3]float64) float64 {
	summary := SummarizeColor(img)
	var sum float64
	for channel := 0; channel < 3; channel++ {
		delta := summary.MeanOutput8[channel] - target[channel]
		sum += delta * delta
	}
	return math.Sqrt(sum / 3.0)
}

// AverageSRGBColorError keeps the older descriptive name for callers.
func AverageSRGBColorError(img *RGBImage, target [3]float64) float64 {
	return AverageOutputColorError(img, target)
}

// MeanAbsoluteError is the average per-channel linear-light pixel difference.
func MeanAbsoluteError(a, b *RGBImage) float64 {
	if a.Width != b.Width || a.Height != b.Height {
		return math.NaN()
	}
	var sum float64
	for i := range a.Pix {
		sum += math.Abs(a.Pix[i] - b.Pix[i])
	}
	return sum / float64(len(a.Pix))
}

// BorderBandMAE compares only the outer band containing tile-seam artifacts.
func BorderBandMAE(a, b *RGBImage, band int) float64 {
	if a.Width != b.Width || a.Height != b.Height {
		return math.NaN()
	}
	if band < 1 {
		band = 1
	}
	var sum float64
	var count int
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			border := x < band || y < band || x >= a.Width-band || y >= a.Height-band
			if !border {
				continue
			}
			ai := a.index(x, y)
			bi := b.index(x, y)
			for channel := 0; channel < 3; channel++ {
				sum += math.Abs(a.Pix[ai+channel] - b.Pix[bi+channel])
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func BorderMeanLuminance(img *RGBImage, band int) float64 {
	if band < 1 {
		band = 1
	}
	var sum float64
	var count int
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			border := x < band || y < band || x >= img.Width-band || y >= img.Height-band
			if !border {
				continue
			}
			r, g, b := img.RGB(x, y)
			sum += Luminance([3]float64{r, g, b})
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func MaxMeanColorDrift(chain []*RGBImage, baseMean [3]float64) float64 {
	maxDrift := 0.0
	for _, level := range chain {
		err := AverageColorError(level, baseMean)
		if err > maxDrift {
			maxDrift = err
		}
	}
	return maxDrift
}
