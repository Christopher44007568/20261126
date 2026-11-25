package main

import "math"

func cropFloatImage(src *floatImage, rect imageRect) *floatImage {
	dst := newFloatImage(rect.width, rect.height)
	for y := 0; y < rect.height; y++ {
		for x := 0; x < rect.width; x++ {
			for c := 0; c < 3; c++ {
				dst.set(x, y, c, src.at(rect.x+x, rect.y+y, c))
			}
		}
	}
	return dst
}

type imageRect struct {
	x, y, width, height int
}

func highFrequencyEnergy(src *floatImage) float64 {
	total := 0.0
	count := 0.0
	for y := 1; y < src.height-1; y++ {
		for x := 1; x < src.width-1; x++ {
			for c := 0; c < 3; c++ {
				laplacian := src.at(x-1, y, c) + src.at(x+1, y, c) +
					src.at(x, y-1, c) + src.at(x, y+1, c) - 4*src.at(x, y, c)
				total += laplacian * laplacian
				count++
			}
		}
	}
	return total / count
}

func laplacianAt(src *floatImage, x, y, c int) float64 {
	return src.at(x-1, y, c) + src.at(x+1, y, c) +
		src.at(x, y-1, c) + src.at(x, y+1, c) - 4*src.at(x, y, c)
}

func textureAlignment(reference, candidate *floatImage) float64 {
	numerator := 0.0
	referenceEnergy := 0.0
	candidateEnergy := 0.0
	referenceResponses := make([]float64, 0, reference.width*reference.height*3)
	for y := 1; y < reference.height-1; y++ {
		for x := 1; x < reference.width-1; x++ {
			for c := 0; c < 3; c++ {
				referenceResponses = append(referenceResponses, sobelMagnitude(reference, x, y, c))
			}
		}
	}
	threshold := percentile(referenceResponses, 0.55)
	index := 0
	for y := 1; y < reference.height-1; y++ {
		for x := 1; x < reference.width-1; x++ {
			for c := 0; c < 3; c++ {
				referenceResponse := referenceResponses[index]
				index++
				if referenceResponse < threshold {
					continue
				}
				candidateResponse := sobelMagnitude(candidate, x, y, c)
				numerator += referenceResponse * candidateResponse
				referenceEnergy += referenceResponse * referenceResponse
				candidateEnergy += candidateResponse * candidateResponse
			}
		}
	}
	denominator := math.Sqrt(referenceEnergy * candidateEnergy)
	if denominator == 0 {
		return 0
	}
	return clampFloat(numerator/denominator, 0, 1)
}

func percentile(values []float64, q float64) float64 {
	sorted := append([]float64(nil), values...)
	for i := 1; i < len(sorted); i++ {
		value := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > value {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = value
	}
	return sorted[int(float64(len(sorted)-1)*q)]
}

func sobelMagnitude(src *floatImage, x, y, c int) float64 {
	gx := -src.at(x-1, y-1, c) + src.at(x+1, y-1, c) -
		2*src.at(x-1, y, c) + 2*src.at(x+1, y, c) -
		src.at(x-1, y+1, c) + src.at(x+1, y+1, c)
	gy := -src.at(x-1, y-1, c) - 2*src.at(x, y-1, c) - src.at(x+1, y-1, c) +
		src.at(x-1, y+1, c) + 2*src.at(x, y+1, c) + src.at(x+1, y+1, c)
	return math.Hypot(gx, gy)
}

func flatRegionRMS(src *floatImage) float64 {
	total := 0.0
	count := 0.0
	for y := 2; y < src.height-2; y++ {
		for x := 2; x < src.width-2; x++ {
			for c := 0; c < 3; c++ {
				average := 0.0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						average += src.at(x+dx, y+dy, c) / 9
					}
				}
				difference := src.at(x, y, c) - average
				total += difference * difference
				count++
			}
		}
	}
	return math.Sqrt(total / count)
}

func textureRetention(noisy, denoised, clean *floatImage, textureRect, flatRect imageRect) map[string]float64 {
	noisyTexture := cropFloatImage(noisy, textureRect)
	denoisedTexture := cropFloatImage(denoised, textureRect)
	cleanTexture := cropFloatImage(clean, textureRect)
	noisyFlat := cropFloatImage(noisy, flatRect)
	denoisedFlat := cropFloatImage(denoised, flatRect)

	cleanEnergy := highFrequencyEnergy(cleanTexture)
	noisyEnergy := highFrequencyEnergy(noisyTexture)
	denoisedEnergy := highFrequencyEnergy(denoisedTexture)
	retention := textureAlignment(cleanTexture, denoisedTexture)
	noiseReduction := 1 - flatRegionRMS(denoisedFlat)/math.Max(flatRegionRMS(noisyFlat), 1e-9)

	return map[string]float64{
		"cleanTextureEnergy":    cleanEnergy,
		"noisyTextureEnergy":    noisyEnergy,
		"denoisedTextureEnergy": denoisedEnergy,
		"textureRetention":      retention,
		"flatNoiseReduction":    noiseReduction,
	}
}
