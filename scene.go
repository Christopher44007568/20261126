package main

import "math"

const (
	frameWidth  = 256
	frameHeight = 192
	frameCount  = 40
	exposure    = 0.65
)

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

func buildScene() (*Image, []bool) {
	scene := NewImage(frameWidth, frameHeight)
	neutralMask := make([]bool, frameWidth*frameHeight)
	background := RGB{R: 0.34, G: 0.35, B: 0.33}

	for y := 0; y < frameHeight; y++ {
		for x := 0; x < frameWidth; x++ {
			scene.Set(x, y, addSceneNoise(background, x, y, 0.003))
		}
	}

	patches := []struct {
		Rect    Rect
		Color   RGB
		Neutral bool
	}{
		{Rect{20, 22, 42, 38}, RGB{0.25, 0.25, 0.25}, true},
		{Rect{78, 22, 42, 38}, RGB{0.62, 0.18, 0.14}, false},
		{Rect{136, 22, 42, 38}, RGB{0.16, 0.55, 0.18}, false},
		{Rect{194, 22, 42, 38}, RGB{0.15, 0.22, 0.60}, false},
		{Rect{20, 77, 42, 38}, RGB{0.45, 0.45, 0.45}, true},
		{Rect{78, 77, 42, 38}, RGB{0.15, 0.52, 0.55}, false},
		{Rect{136, 77, 42, 38}, RGB{0.55, 0.20, 0.48}, false},
		{Rect{194, 77, 42, 38}, RGB{0.65, 0.58, 0.14}, false},
		{Rect{20, 132, 42, 38}, RGB{0.65, 0.65, 0.65}, true},
		{Rect{78, 132, 42, 38}, RGB{0.70, 0.34, 0.12}, false},
		{Rect{136, 132, 42, 38}, RGB{0.34, 0.24, 0.55}, false},
		{Rect{194, 132, 42, 38}, RGB{0.80, 0.80, 0.80}, true},
	}

	for _, patch := range patches {
		for y := patch.Rect.Y; y < patch.Rect.Y+patch.Rect.Height; y++ {
			for x := patch.Rect.X; x < patch.Rect.X+patch.Rect.Width; x++ {
				scene.Set(x, y, addSceneNoise(patch.Color, x, y, 0.0025))
				neutralMask[y*frameWidth+x] = patch.Neutral
			}
		}
	}

	balanceSceneReflectance(scene, neutralMask)
	return scene, neutralMask
}

func balanceSceneReflectance(scene *Image, neutralMask []bool) {
	sum := RGB{}
	neutralCount := 0
	for i, pixel := range scene.Pix {
		if neutralMask[i] {
			neutralCount++
			continue
		}
		sum.R += pixel.R
		sum.G += pixel.G
		sum.B += pixel.B
	}
	adjustable := float64(len(scene.Pix) - neutralCount)
	deltaR := (sum.G - sum.R) / adjustable
	deltaB := (sum.G - sum.B) / adjustable
	for i := range scene.Pix {
		if neutralMask[i] {
			continue
		}
		scene.Pix[i].R = clamp01(scene.Pix[i].R + deltaR)
		scene.Pix[i].B = clamp01(scene.Pix[i].B + deltaB)
	}
}

func addSceneNoise(c RGB, x, y int, amplitude float64) RGB {
	noise := deterministicNoise(x, y)*amplitude - amplitude/2
	return RGB{
		R: clamp01(c.R + noise),
		G: clamp01(c.G + noise*0.7),
		B: clamp01(c.B + noise*1.3),
	}
}

func deterministicNoise(x, y int) float64 {
	value := math.Sin(float64(x)*12.9898+float64(y)*78.233) * 43758.5453
	return value - math.Floor(value)
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func renderFrame(scene *Image, light RGB) *Image {
	frame := NewImage(scene.Width, scene.Height)
	for i, reflectance := range scene.Pix {
		frame.Pix[i] = RGB{
			R: clamp01(reflectance.R * light.R * exposure),
			G: clamp01(reflectance.G * light.G * exposure),
			B: clamp01(reflectance.B * light.B * exposure),
		}
	}
	return frame
}
