package main

import "math"

type RGB struct {
	R float64
	G float64
	B float64
}

type Image struct {
	Width  int
	Height int
	Pix    []RGB
}

func NewImage(width, height int) *Image {
	return &Image{
		Width:  width,
		Height: height,
		Pix:    make([]RGB, width*height),
	}
}

func (img *Image) At(x, y int) RGB {
	return img.Pix[y*img.Width+x]
}

func (img *Image) Set(x, y int, c RGB) {
	img.Pix[y*img.Width+x] = c
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func encodeSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}
