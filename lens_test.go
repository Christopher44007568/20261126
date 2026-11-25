package main

import (
	"math"
	"testing"
)

func TestCircleOfConfusionZeroAtFocalPlane(t *testing.T) {
	got := CircleOfConfusion(3.5, 3.5, 0.08, 0.057)
	if got > 1e-12 {
		t.Fatalf("CoC at focus = %v, want zero", got)
	}
}

func TestSignedCoCDistinguishesNearAndFar(t *testing.T) {
	const focus, focal, diameter = 4.0, 0.1, 0.07
	near := SignedCircleOfConfusion(2.0, focus, focal, diameter)
	far := SignedCircleOfConfusion(8.0, focus, focal, diameter)
	if near >= 0 {
		t.Fatalf("near signed CoC = %v, want negative", near)
	}
	if far <= 0 {
		t.Fatalf("far signed CoC = %v, want positive", far)
	}
	if math.Abs(near) <= CircleOfConfusion(3.9, focus, focal, diameter) {
		t.Fatal("CoC should grow with distance from the focal plane")
	}
}

func TestSmallApertureReducesCoC(t *testing.T) {
	wide := CoCRadiusPixels(7, 3, 0.1, 1.4, 0.024, 400)
	small := CoCRadiusPixels(7, 3, 0.1, 16, 0.024, 400)
	if wide <= small*4 {
		t.Fatalf("wide radius %.2f should greatly exceed small radius %.2f", wide, small)
	}
	if small > 1.5 {
		t.Fatalf("f/16 CoC radius = %.2fpx, want near-sharp", small)
	}
}

func TestHexagonalApertureShapeAndSampling(t *testing.T) {
	a := RegularAperture(6, 2)
	if !a.Contains(0, 0) || !a.Contains(1.7, 0.95) {
		t.Fatal("center and a vertex-near point should be inside hexagon")
	}
	if a.Contains(2.05, 0) {
		t.Fatal("point beyond hexagon apothem should be outside")
	}
	for i := 0; i < 2000; i++ {
		u := math.Mod(float64(i)*0.61803398875, 1)
		v := math.Mod(float64(i)*0.754877666, 1)
		x, y := a.Sample(u, v)
		if !a.Contains(x, y) {
			t.Fatalf("sample (%g,%g) -> (%g,%g) outside aperture", u, v, x, y)
		}
	}
}
