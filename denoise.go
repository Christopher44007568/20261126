package main

import "math"

func subtractLineNoise(input *floatImage, fraction float64) *floatImage {
	output := newFloatImage(input.width, input.height)
	width := input.width
	height := input.height

	columnMean := make([]float64, width*3)
	rowMean := make([]float64, height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for c := 0; c < 3; c++ {
				value := input.at(x, y, c)
				columnMean[x*3+c] += value / float64(height)
				rowMean[y*3+c] += value / float64(width)
			}
		}
	}

	globalMean := make([]float64, 3)
	for x := 0; x < width; x++ {
		for c := 0; c < 3; c++ {
			globalMean[c] += columnMean[x*3+c] / float64(width)
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for c := 0; c < 3; c++ {
				columnOffset := columnMean[x*3+c] - globalMean[c]
				rowOffset := rowMean[y*3+c] - globalMean[c]
				value := input.at(x, y, c) - fraction*(columnOffset+rowOffset)
				output.set(x, y, c, math.Max(0, value))
			}
		}
	}
	return output
}

func bilateralDenoise(input *floatImage, radius int, spatialSigma, noiseFloor, detailFraction float64) *floatImage {
	output := newFloatImage(input.width, input.height)
	spatialWeight := make([]float64, radius*2+1)
	for dx := -radius; dx <= radius; dx++ {
		spatialWeight[dx+radius] = math.Exp(-float64(dx*dx) / (2 * spatialSigma * spatialSigma))
	}

	for y := 0; y < input.height; y++ {
		for x := 0; x < input.width; x++ {
			for c := 0; c < 3; c++ {
				center := input.at(x, y, c)
				noiseSigma := noiseFloor + math.Sqrt(math.Max(center, 0))*0.72
				rangeSigma := noiseSigma * 1.45
				weightSum := 0.0
				valueSum := 0.0

				for dy := -radius; dy <= radius; dy++ {
					ny := clampInt(y+dy, 0, input.height-1)
					for dx := -radius; dx <= radius; dx++ {
						nx := clampInt(x+dx, 0, input.width-1)
						neighbor := input.at(nx, ny, c)
						difference := neighbor - center
						weight := spatialWeight[dx+radius] * spatialWeight[dy+radius]
						weight *= math.Exp(-(difference * difference) / (2 * rangeSigma * rangeSigma))
						valueSum += neighbor * weight
						weightSum += weight
					}
				}

				filtered := valueSum / weightSum
				value := center*(1-detailFraction) + filtered*detailFraction
				output.set(x, y, c, math.Max(0, value))
			}
		}
	}
	return output
}

func denoiseCapture(input *floatImage, setup exposure) *floatImage {
	lineCorrected := subtractLineNoise(input, 0.72)
	radius := 2
	if setup.iso >= 1600 {
		radius = 3
	}
	first := bilateralDenoise(lineCorrected, radius, float64(radius)*0.82, setup.read*0.78, 0.76)
	second := bilateralDenoise(first, 2, 1.55, setup.read*0.55, 0.48)
	directional := separableEdgeSmooth(second, 1, 9, setup.read)
	directional = separableEdgeSmooth(directional, 1, 5, setup.read*0.82)
	return blendImages(second, directional, 0.76)
}

func separableEdgeSmooth(input *floatImage, horizontalRadius, verticalRadius int, noiseFloor float64) *floatImage {
	horizontal := newFloatImage(input.width, input.height)
	for y := 0; y < input.height; y++ {
		for x := 0; x < input.width; x++ {
			for c := 0; c < 3; c++ {
				center := input.at(x, y, c)
				rangeSigma := (noiseFloor + math.Sqrt(math.Max(center, 0))*0.75) * 0.85
				weightSum := 0.0
				valueSum := 0.0
				for dx := -horizontalRadius; dx <= horizontalRadius; dx++ {
					nx := clampInt(x+dx, 0, input.width-1)
					neighbor := input.at(nx, y, c)
					weight := math.Exp(-float64(dx*dx)/(2*float64(horizontalRadius*horizontalRadius)) -
						math.Pow(neighbor-center, 2)/(2*rangeSigma*rangeSigma))
					valueSum += neighbor * weight
					weightSum += weight
				}
				horizontal.set(x, y, c, valueSum/weightSum)
			}
		}
	}

	output := newFloatImage(input.width, input.height)
	for y := 0; y < input.height; y++ {
		for x := 0; x < input.width; x++ {
			for c := 0; c < 3; c++ {
				center := horizontal.at(x, y, c)
				rangeSigma := noiseFloor + math.Sqrt(math.Max(center, 0))*0.68
				weightSum := 0.0
				valueSum := 0.0
				for dy := -verticalRadius; dy <= verticalRadius; dy++ {
					ny := clampInt(y+dy, 0, input.height-1)
					neighbor := horizontal.at(x, ny, c)
					weight := math.Exp(-float64(dy*dy)/(2*float64(verticalRadius*verticalRadius)) -
						math.Pow(neighbor-center, 2)/(2*rangeSigma*rangeSigma))
					valueSum += neighbor * weight
					weightSum += weight
				}
				output.set(x, y, c, valueSum/weightSum)
			}
		}
	}
	return output
}

func blendImages(a, b *floatImage, fractionB float64) *floatImage {
	output := newFloatImage(a.width, a.height)
	for i := range a.pix {
		output.pix[i] = a.pix[i]*(1-fractionB) + b.pix[i]*fractionB
	}
	return output
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
