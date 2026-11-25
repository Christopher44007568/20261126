package main

import "math"

const (
	sceneWidth  = 640
	sceneHeight = 480
)

type floatImage struct {
	width, height int
	pix           []float64
}

func newFloatImage(width, height int) *floatImage {
	return &floatImage{width: width, height: height, pix: make([]float64, width*height*3)}
}

func (f *floatImage) index(x, y int) int {
	return (y*f.width + x) * 3
}

func (f *floatImage) at(x, y, channel int) float64 {
	return f.pix[f.index(x, y)+channel]
}

func (f *floatImage) set(x, y, channel int, value float64) {
	f.pix[f.index(x, y)+channel] = value
}

func referenceScene() *floatImage {
	scene := newFloatImage(sceneWidth, sceneHeight)
	for y := 0; y < sceneHeight; y++ {
		for x := 0; x < sceneWidth; x++ {
			fx := float64(x) / float64(sceneWidth)
			fy := float64(y) / float64(sceneHeight)
			base := 55 + 105*fy

			gray := 35 + 690*fx
			if y < 130 {
				base = gray
			}

			if y > 155 && x < 190 {
				checker := (x/24+y/24)%2 == 0
				if checker {
					base = 190
				} else {
					base = 65
				}
			}

			colors := [][3]float64{
				{760, 115, 115},
				{115, 690, 150},
				{90, 180, 790},
				{720, 650, 120},
				{610, 145, 650},
			}
			for patch, color := range colors {
				if x >= 230+patch*74 && x < 290+patch*74 && y > 165 && y < 270 {
					base = -1
					for c := 0; c < 3; c++ {
						scene.set(x, y, c, color[c])
					}
				}
			}

			if base >= 0 {
				for c := 0; c < 3; c++ {
					scene.set(x, y, c, base)
				}
			}

			if x > 405 && y > 60 && y < 420 {
				stripe := (x-405)/8%2 == 0
				amplitude := 38 + 22*math.Sin(float64(y)*0.035)
				for c := 0; c < 3; c++ {
					old := scene.at(x, y, c)
					if stripe {
						scene.set(x, y, c, old+amplitude)
					} else {
						scene.set(x, y, c, math.Max(8, old-amplitude*0.55))
					}
				}
			}

			dx := fx - 0.31
			dy := fy - 0.78
			if dx*dx+dy*dy < 0.032*0.032 {
				for c := 0; c < 3; c++ {
					scene.set(x, y, c, scene.at(x, y, c)*0.45+18)
				}
			}

			if x < 380 && y > 300 {
				dot := math.Hypot(float64(x%18-9), float64(y%18-9)) < 2.7
				if dot {
					for c := 0; c < 3; c++ {
						scene.set(x, y, c, scene.at(x, y, c)+32)
					}
				}
			}

			edgeDistance := math.Abs(float64(x - (105 + y*3/5)))
			if edgeDistance < 3.2 && y > 285 {
				for c := 0; c < 3; c++ {
					scene.set(x, y, c, 520+30*float64(c))
				}
			}
		}
	}
	return scene
}
