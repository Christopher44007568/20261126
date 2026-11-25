package main

import "math"

// ReinhardTonemap maps linear irradiance into display values. It uses a
// log-average key (Reinhard global operator) followed by gamma encoding.
func ReinhardTonemap(logE []float64, ok []bool) *RGB8 {
	h, w := imgH, imgW
	img := NewRGB8(w, h)

	var sum, count float64
	for i, valid := range ok {
		if valid {
			sum += logE[i]
			count++
		}
	}
	avgLog := sum / count
	key := math.Exp(avgLog) * 0.18

	for p := 0; p < w*h; p++ {
		if !ok[p] {
			img.Set(p%w, p/w, 255, 0, 255) // magenta: unrecoverable pixel
			continue
		}
		scaled := math.Exp(logE[p]) / key
		mapped := scaled / (1 + scaled)
		display := math.Pow(mapped, 1/2.2)
		v := uint8(math.Round(clamp01(display) * 255))
		img.Set(p%w, p/w, v, v, v)
	}
	return img
}
