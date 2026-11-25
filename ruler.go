package main

import "fmt"

const (
	rulerWidth  = 2048
	rulerHeight = 300
	rulerZNear  = 2.2
	rulerZFar   = 8.2
	rulerFocusZ = 4.6
)

func rulerX(z float64) int {
	u := (z - rulerZNear) / (rulerZFar - rulerZNear)
	return int(u * rulerWidth)
}

func MakeRulerTexture() *Texture {
	c := NewCanvas(rulerWidth, rulerHeight)
	c.Fill(239, 232, 214)

	focusX := rulerX(rulerFocusZ)
	c.Rect(0, 0, focusX, 58, 178, 67, 58, 1)
	c.Rect(focusX, 0, rulerWidth-focusX, 58, 42, 103, 190, 1)
	c.Rect(focusX-7, 0, 14, rulerHeight, 46, 220, 130, 0.95)
	c.Rect(focusX-2, 0, 4, rulerHeight, 235, 255, 235, 1)

	DrawTextShadow(c, "NEAR", 34, 9, 5, 255, 235, 220)
	farText := "FAR"
	DrawTextShadow(c, farText, rulerWidth-TextWidth(farText, 5)-34, 9, 5, 220, 238, 255)
	title := "NEAR-FAR FOCUS RULER"
	DrawTextShadow(c, title, rulerWidth/2-TextWidth(title, 4)/2, 14, 4, 245, 248, 255)

	border := [3]uint8{34, 42, 58}
	c.Rect(0, 58, rulerWidth, 5, border[0], border[1], border[2], 1)
	c.Rect(0, rulerHeight-8, rulerWidth, 8, border[0], border[1], border[2], 1)

	for z := rulerZNear; z <= rulerZFar+0.001; z += 0.25 {
		x := rulerX(z)
		isMajor := false
		for integer := 3; integer <= 8; integer++ {
			if absFloat(z-float64(integer)) < 0.01 {
				isMajor = true
			}
		}
		if absFloat(z-rulerFocusZ) < 0.01 {
			isMajor = true
		}
		if isMajor {
			c.Rect(x-3, 72, 6, 88, border[0], border[1], border[2], 1)
		} else {
			c.Rect(x-1, 72, 2, 48, 80, 88, 104, 1)
		}
	}

	for z := 3; z <= 8; z++ {
		x := rulerX(float64(z))
		text := fmt.Sprintf("%dM", z)
		scale := 10
		tw := TextWidth(text, scale)
		DrawTextShadow(c, text, x-tw/2, 174, scale, 28, 34, 48)
	}

	focusText := "FOCAL PLANE 4.6M"
	fs := 6
	fw := TextWidth(focusText, fs)
	fx := focusX + 24
	if fx+fw+30 > rulerWidth {
		fx = focusX - fw - 54
	}
	c.Rect(fx-14, 252-8, fw+28, 52, 20, 110, 70, 0.92)
	DrawText(c, focusText, fx, 252, fs, 235, 255, 235)

	return TextureFromCanvas(c)
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func rulerScene(fStop float64) DemoScene {
	aperture := NewAperture(6, 0.364, 1.4)
	if fStop > 0 {
		aperture = withFStop(aperture, fStop)
	}
	eye := V(0, 0.34, -0.25)
	focusPoint := V(0.05, 0.0, rulerFocusZ)
	focus := focusPoint.Sub(eye).Len()
	cam := NewCamera(eye, focusPoint, V(0, 1, 0), 40, 1.6, focus, aperture)

	tex := MakeRulerTexture()
	rulerMat := &Material{Albedo: V(1, 1, 1), Unlit: true, Texture: tex, TexStrength: 1}
	quad := &Quad{
		Origin: V(-1.25, -0.18, rulerZNear),
		UAxis:  V(3.25, 0, rulerZFar-rulerZNear),
		VAxis:  V(0, 0.72, 0),
		Mat:    rulerMat,
	}
	scene := &Scene{BackgroundColor: darkBackground}
	scene.Add(quad)

	return DemoScene{
		FileBase: "near_far_ruler",
		Title:    "NEAR / FAR DEPTH RULER",
		Subtitle: fmt.Sprintf("F%g  FOCUS 4.6M: NEAR/FAR SIGNED COC", aperture.FNumber),
		Scene:    scene, Camera: cam, Exposure: 1.0,
	}
}
