package main

import "math"

const errorThreshold = 3.0

type Viewport struct {
	X    int
	Y    int
	Size int
}

type Camera struct {
	Center UV
	Scale  float64
}

func (v Viewport) CenterX() float64 { return float64(v.X) + float64(v.Size)/2 }
func (v Viewport) CenterY() float64 { return float64(v.Y) + float64(v.Size)/2 }

func (c Camera) UVAt(px, py int, v Viewport) UV {
	return UV{
		U: c.Center.U + (float64(px)+0.5-v.CenterX())/c.Scale,
		V: c.Center.V + (float64(py)+0.5-v.CenterY())/c.Scale,
	}
}

func nodeTexelPixels(level int, scale float64) float64 {
	return scale / float64(tilePixels*levelCount(level))
}

func levelCount(level int) int {
	return int(1) << level
}

func idealLOD(c Camera) int {
	for level := 0; level < maxLOD; level++ {
		if nodeTexelPixels(level, c.Scale) <= errorThreshold {
			return level
		}
	}
	return maxLOD
}

func screenErrorPixels(level int, c Camera) float64 {
	texel := nodeTexelPixels(level, c.Scale)
	return math.Max(0, texel-errorThreshold)
}

func CameraAt(frame float64) Camera {
	keyframes := []Camera{
		{Center: UV{U: 0.50, V: 0.50}, Scale: 380},
		{Center: UV{U: 0.59, V: 0.53}, Scale: 650},
		{Center: UV{U: 0.68, V: 0.56}, Scale: 1250},
	}
	if frame <= 4 {
		t := smoothStep(clampFloat(frame/4, 0, 1))
		return lerpCamera(keyframes[0], keyframes[1], t)
	}
	t := smoothStep(clampFloat((frame-4)/4, 0, 1))
	return lerpCamera(keyframes[1], keyframes[2], t)
}

func smoothStep(t float64) float64 {
	return t * t * (3 - 2*t)
}

func lerpCamera(a, b Camera, t float64) Camera {
	scale := math.Exp(lerpFloat(math.Log(a.Scale), math.Log(b.Scale), t))
	return Camera{
		Center: UV{
			U: lerpFloat(a.Center.U, b.Center.U, t),
			V: lerpFloat(a.Center.V, b.Center.V, t),
		},
		Scale: scale,
	}
}

func lerpFloat(a, b, t float64) float64 {
	return a + (b-a)*t
}
