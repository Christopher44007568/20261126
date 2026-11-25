package main

import "math"

const fullWell = 9000.0

type exposure struct {
	name   string
	scale  float64
	iso    int
	read   float64
	fpn    float64
	output string
}

var exposures = []exposure{
	{name: "low", scale: 0.11, iso: 3200, read: 34, fpn: 12, output: "exposure_low.png"},
	{name: "middle", scale: 0.43, iso: 800, read: 15, fpn: 8, output: "exposure_middle.png"},
	{name: "high", scale: 1.55, iso: 200, read: 7, fpn: 5, output: "exposure_high.png"},
}

var channelGain = [3]float64{1.0, 1.18, 1.42}

type fixedPattern struct {
	pixel  *floatImage
	column []float64
	row    []float64
}

func buildFixedPattern(r *rng, strength float64) *fixedPattern {
	pixel := newFloatImage(sceneWidth, sceneHeight)
	column := make([]float64, sceneWidth)
	row := make([]float64, sceneHeight)
	for x := 0; x < sceneWidth; x++ {
		column[x] = r.gaussian(0, strength*2.15)
	}
	for y := 0; y < sceneHeight; y++ {
		row[y] = r.gaussian(0, strength*0.85)
	}
	for y := 0; y < sceneHeight; y++ {
		for x := 0; x < sceneWidth; x++ {
			for c := 0; c < 3; c++ {
				channelScale := 0.9 + 0.18*float64(c)
				pixel.set(x, y, c, r.gaussian(0, strength*channelScale))
			}
		}
	}
	return &fixedPattern{pixel: pixel, column: column, row: row}
}

type capturedImage struct {
	clean        *floatImage
	raw          *floatImage
	shotNoise    *floatImage
	readNoise    *floatImage
	patternNoise *floatImage
	clipped      []bool
}

func capture(scene *floatImage, r *rng, setup exposure) *capturedImage {
	clean := newFloatImage(sceneWidth, sceneHeight)
	raw := newFloatImage(sceneWidth, sceneHeight)
	shot := newFloatImage(sceneWidth, sceneHeight)
	read := newFloatImage(sceneWidth, sceneHeight)
	pattern := newFloatImage(sceneWidth, sceneHeight)
	clipped := make([]bool, sceneWidth*sceneHeight)
	fpn := buildFixedPattern(r, setup.fpn)

	for y := 0; y < sceneHeight; y++ {
		for x := 0; x < sceneWidth; x++ {
			for c := 0; c < 3; c++ {
				expected := scene.at(x, y, c) * setup.scale * channelGain[c]
				expected = math.Min(expected, fullWell)
				clean.set(x, y, c, expected)

				electrons := r.poisson(expected)
				shotValue := electrons - expected
				shot.set(x, y, c, shotValue)

				readSigma := setup.read * (1 + 0.12*float64(c))
				readValue := r.gaussian(0, readSigma)
				read.set(x, y, c, readValue)

				patternValue := fpn.pixel.at(x, y, c) + fpn.column[x] + fpn.row[y]
				pattern.set(x, y, c, patternValue)

				value := electrons + readValue + patternValue
				if value < 0 {
					value = 0
				}
				if value >= fullWell {
					value = fullWell
					clipped[y*sceneWidth+x] = true
				}
				raw.set(x, y, c, value)
			}
		}
	}

	return &capturedImage{
		clean:        clean,
		raw:          raw,
		shotNoise:    shot,
		readNoise:    read,
		patternNoise: pattern,
		clipped:      clipped,
	}
}

func signedComponent(component *floatImage, sigmaScale float64) *floatImage {
	visual := newFloatImage(sceneWidth, sceneHeight)
	for y := 0; y < sceneHeight; y++ {
		for x := 0; x < sceneWidth; x++ {
			i := visual.index(x, y)
			for c := 0; c < 3; c++ {
				signed := math.Tanh(component.at(x, y, c) / sigmaScale)
				if signed >= 0 {
					visual.pix[i] = 18 * signed
					visual.pix[i+1] = 82 * signed
					visual.pix[i+2] = 255 * signed
				} else {
					magnitude := -signed
					visual.pix[i] = 255 * magnitude
					visual.pix[i+1] = 96 * magnitude
					visual.pix[i+2] = 28 * magnitude
				}
			}
		}
	}
	return visual
}

func fixedComponent(component *floatImage, sigma float64) *floatImage {
	visual := newFloatImage(sceneWidth, sceneHeight)
	for y := 0; y < sceneHeight; y++ {
		for x := 0; x < sceneWidth; x++ {
			i := visual.index(x, y)
			for c := 0; c < 3; c++ {
				magnitude := math.Abs(math.Tanh(component.at(x, y, c) / sigma))
				visual.pix[i] = 34 + 221*magnitude
				visual.pix[i+1] = 190 * magnitude
				visual.pix[i+2] = 72 * magnitude
			}
		}
	}
	return visual
}
