package main

import (
	"fmt"
	"math"
	"os"
)

type Canvas struct {
	Width  int
	Height int
	Pixels []Vec3
}

func NewCanvas(width, height int) *Canvas {
	return &Canvas{Width: width, Height: height, Pixels: make([]Vec3, width*height)}
}

func (c *Canvas) index(x, y int) int { return y*c.Width + x }

func (c *Canvas) Set(x, y int, color Vec3) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	c.Pixels[c.index(x, y)] = color
}

func (c *Canvas) Blend(x, y int, color Vec3, alpha float64) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	i := c.index(x, y)
	c.Pixels[i] = c.Pixels[i].Mul(1 - alpha).Add(color.Mul(alpha))
}

func (c *Canvas) FillRect(x0, y0, width, height int, color Vec3, alpha float64) {
	for y := y0; y < y0+height; y++ {
		for x := x0; x < x0+width; x++ {
			c.Blend(x, y, color, alpha)
		}
	}
}

func (c *Canvas) WritePPM(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "P6\n%d %d\n255\n", c.Width, c.Height); err != nil {
		return err
	}
	row := make([]byte, c.Width*3)
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			color := c.Pixels[c.index(x, y)]
			color = V(linearToSRGB(color.X), linearToSRGB(color.Y), linearToSRGB(color.Z))
			row[x*3+0] = byte(clamp01(color.X)*255 + 0.5)
			row[x*3+1] = byte(clamp01(color.Y)*255 + 0.5)
			row[x*3+2] = byte(clamp01(color.Z)*255 + 0.5)
		}
		if _, err := file.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func linearToSRGB(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
