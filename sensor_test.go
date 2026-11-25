package main

import (
	"math"
	"testing"
)

func TestPoissonDistribution(t *testing.T) {
	r := newRNG(99)
	const lambda = 24.0
	const samples = 20000
	mean := 0.0
	secondMoment := 0.0
	for range samples {
		value := r.poisson(lambda)
		mean += value / samples
		secondMoment += value * value / samples
	}
	variance := secondMoment - mean*mean
	if math.Abs(mean-lambda) > 0.35 {
		t.Fatalf("poisson mean = %.3f, want %.3f", mean, lambda)
	}
	if math.Abs(variance-lambda) > 1.4 {
		t.Fatalf("poisson variance = %.3f, want %.3f", variance, lambda)
	}
}

func TestCaptureHasDistinctNoiseSources(t *testing.T) {
	scene := referenceScene()
	r := newRNG(77)
	captured := capture(scene, r, exposures[0])

	shotMean := channelMean(captured.shotNoise)
	readMean := channelMean(captured.readNoise)
	patternMean := channelMean(captured.patternNoise)
	for c := 0; c < 3; c++ {
		if math.Abs(shotMean[c]) > 4 {
			t.Fatalf("shot residual mean too large: %.3f", shotMean[c])
		}
		if math.Abs(readMean[c]) > 1.2 {
			t.Fatalf("read residual mean too large: %.3f", readMean[c])
		}
		if math.Abs(patternMean[c]) > 2.5 {
			t.Fatalf("fixed pattern mean too large: %.3f", patternMean[c])
		}
	}

	columnVariance := varianceAcrossColumns(captured.patternNoise)
	pixelVariance := pixelVarianceAroundColumnMean(captured.patternNoise)
	if columnVariance <= pixelVariance*1.2 {
		t.Fatalf("column FPN not prominent enough: column %.3f pixel %.3f", columnVariance, pixelVariance)
	}
}

func TestDenoiseReducesNoiseWhileRetainingTexture(t *testing.T) {
	scene := referenceScene()
	setup := exposures[0]
	r := newRNG(1024)
	captured := capture(scene, r, setup)
	denoised := denoiseCapture(captured.raw, setup)

	textureRect := imageRect{x: 414, y: 86, width: 150, height: 138}
	flatRect := imageRect{x: 36, y: 330, width: 145, height: 105}
	metrics := textureRetention(captured.raw, denoised, captured.clean, textureRect, flatRect)
	t.Logf("metrics: %+v", metrics)

	if metrics["flatNoiseReduction"] < 0.22 {
		t.Fatalf("flat noise reduction %.3f too small", metrics["flatNoiseReduction"])
	}
	if metrics["textureRetention"] < 0.68 {
		t.Fatalf("texture retention %.3f too small", metrics["textureRetention"])
	}
	if metrics["textureRetention"] > 1.25 {
		t.Fatalf("texture retention %.3f suggests metric failure", metrics["textureRetention"])
	}
}

func channelMean(img *floatImage) [3]float64 {
	var means [3]float64
	count := float64(img.width * img.height)
	for y := 0; y < img.height; y++ {
		for x := 0; x < img.width; x++ {
			for c := 0; c < 3; c++ {
				means[c] += img.at(x, y, c) / count
			}
		}
	}
	return means
}

func varianceAcrossColumns(img *floatImage) float64 {
	means := make([]float64, img.width)
	for x := 0; x < img.width; x++ {
		sum := 0.0
		for y := 0; y < img.height; y++ {
			for c := 0; c < 3; c++ {
				sum += img.at(x, y, c)
			}
		}
		means[x] = sum / float64(img.height*3)
	}
	return populationVariance(means)
}

func pixelVarianceAroundColumnMean(img *floatImage) float64 {
	values := make([]float64, 0, img.width*img.height)
	for x := 0; x < img.width; x++ {
		columnSum := 0.0
		for y := 0; y < img.height; y++ {
			for c := 0; c < 3; c++ {
				columnSum += img.at(x, y, c)
			}
		}
		columnMean := columnSum / float64(img.height*3)
		for y := 0; y < img.height; y++ {
			for c := 0; c < 3; c++ {
				values = append(values, img.at(x, y, c)-columnMean)
			}
		}
	}
	return populationVariance(values)
}

func populationVariance(values []float64) float64 {
	mean := 0.0
	for _, value := range values {
		mean += value / float64(len(values))
	}
	variance := 0.0
	for _, value := range values {
		difference := value - mean
		variance += difference * difference / float64(len(values))
	}
	return variance
}
