package main

import (
	"math"
	"sort"
)

type Gains struct {
	R float64
	G float64
	B float64
}

func grayWorldGains(frame *Image) Gains {
	mean := meanRGB(frame)
	return gainsFromLight(mean)
}

func whitePointGains(frame *Image) Gains {
	luminances := make([]float64, len(frame.Pix))
	type sample struct {
		Lum float64
		RGB RGB
	}
	samples := make([]sample, len(frame.Pix))
	for i, pixel := range frame.Pix {
		lum := 0.2126*pixel.R + 0.7152*pixel.G + 0.0722*pixel.B
		luminances[i] = lum
		samples[i] = sample{Lum: lum, RGB: pixel}
	}
	sort.Float64s(luminances)
	threshold := luminances[int(float64(len(luminances)-1)*0.93)]

	sum := RGB{}
	count := 0.0
	for _, candidate := range samples {
		if candidate.Lum >= threshold {
			sum.R += candidate.RGB.R
			sum.G += candidate.RGB.G
			sum.B += candidate.RGB.B
			count++
		}
	}
	if count == 0 {
		return grayWorldGains(frame)
	}
	return gainsFromLight(RGB{R: sum.R / count, G: sum.G / count, B: sum.B / count})
}

func gainsFromLight(light RGB) Gains {
	if light.R == 0 || light.G == 0 || light.B == 0 {
		return Gains{R: 1, G: 1, B: 1}
	}
	return clampGains(Gains{R: light.G / light.R, G: 1, B: light.G / light.B})
}

func clampGains(g Gains) Gains {
	return Gains{
		R: clampGain(g.R),
		G: 1,
		B: clampGain(g.B),
	}
}

func clampGain(v float64) float64 {
	return math.Max(0.125, math.Min(8.0, v))
}

func meanRGB(frame *Image) RGB {
	sum := RGB{}
	for _, pixel := range frame.Pix {
		sum.R += pixel.R
		sum.G += pixel.G
		sum.B += pixel.B
	}
	count := float64(len(frame.Pix))
	return RGB{R: sum.R / count, G: sum.G / count, B: sum.B / count}
}

type TemporalSmoother struct {
	previousRaw Gains
	smoothed    Gains
	initialized bool
	stableCount int
}

func (smoother *TemporalSmoother) Update(raw Gains, alpha float64) Gains {
	if !smoother.initialized {
		smoother.initialized = true
		smoother.previousRaw = raw
		smoother.smoothed = raw
		return raw
	}

	stable := math.Abs(raw.R-smoother.previousRaw.R) < 0.002 &&
		math.Abs(raw.B-smoother.previousRaw.B) < 0.002
	if stable {
		smoother.stableCount++
	} else {
		smoother.stableCount = 0
	}

	if smoother.stableCount >= 1 {
		smoother.smoothed = raw
	} else {
		smoother.smoothed = blendGains(smoother.smoothed, raw, alpha)
	}
	smoother.previousRaw = raw
	return smoother.smoothed
}

func blendGains(previous, current Gains, alpha float64) Gains {
	return clampGains(Gains{
		R: math.Exp((1-alpha)*math.Log(previous.R) + alpha*math.Log(current.R)),
		G: 1,
		B: math.Exp((1-alpha)*math.Log(previous.B) + alpha*math.Log(current.B)),
	})
}

func applyGains(frame *Image, gains Gains) *Image {
	result := NewImage(frame.Width, frame.Height)
	for i, pixel := range frame.Pix {
		result.Pix[i] = RGB{
			R: clamp01(pixel.R * gains.R),
			G: clamp01(pixel.G * gains.G),
			B: clamp01(pixel.B * gains.B),
		}
	}
	return result
}
