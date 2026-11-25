package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type Canvas struct {
	Width  int
	Height int
	Pixels []RGB
}

func NewCanvas(width, height int) *Canvas {
	return &Canvas{Width: width, Height: height, Pixels: make([]RGB, width*height)}
}

func (c *Canvas) Clone() *Canvas {
	pixels := append(make([]RGB, 0, len(c.Pixels)), c.Pixels...)
	return &Canvas{Width: c.Width, Height: c.Height, Pixels: pixels}
}

func (c *Canvas) At(x, y int) int { return y*c.Width + x }

func (c *Canvas) InBounds(x, y int) bool {
	return x >= 0 && x < c.Width && y >= 0 && y < c.Height
}

func (c *Canvas) Set(x, y int, color RGB) {
	if !c.InBounds(x, y) {
		return
	}
	c.Pixels[c.At(x, y)] = color
}

func (c *Canvas) Get(x, y int) RGB {
	if !c.InBounds(x, y) {
		return RGB{}
	}
	return c.Pixels[c.At(x, y)]
}

func (c *Canvas) Blend(x, y int, color RGB, alpha float64) {
	if !c.InBounds(x, y) {
		return
	}
	i := c.At(x, y)
	c.Pixels[i] = c.Pixels[i].Mul(1 - alpha).Add(color.Mul(alpha))
}

func (c *Canvas) Fill(color RGB) {
	for i := range c.Pixels {
		c.Pixels[i] = color
	}
}

func (c *Canvas) FillRect(x0, y0, w, h int, color RGB, alpha float64) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.Blend(x, y, color, alpha)
		}
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 int, color RGB) {
	c.LineAlpha(x0, y0, x1, y1, color, 1)
}

func (c *Canvas) LineAlpha(x0, y0, x1, y1 int, color RGB, alpha float64) {
	dx := math.Abs(float64(x1 - x0))
	dy := -math.Abs(float64(y1 - y0))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	x, y := x0, y0
	for {
		c.Blend(x, y, color, alpha)
		if x == x1 && y == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
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
	writer := bufio.NewWriter(file)
	row := make([]byte, c.Width*3)
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			color := c.Pixels[c.At(x, y)]
			color = V(linearToSRGB(color.X), linearToSRGB(color.Y), linearToSRGB(color.Z))
			row[x*3+0] = byte(clamp01(color.X)*255 + 0.5)
			row[x*3+1] = byte(clamp01(color.Y)*255 + 0.5)
			row[x*3+2] = byte(clamp01(color.Z)*255 + 0.5)
		}
		if _, err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func linearToSRGB(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func srgbToLinear(v float64) float64 {
	v = clamp01(v)
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func ReadPPM(path string) (*Canvas, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(strings.NewReader(string(data)))
	var magic string
	var width, height, maxValue int
	if _, err := fmt.Fscan(reader, &magic); err != nil {
		return nil, err
	}
	if magic != "P6" {
		return nil, fmt.Errorf("unsupported PPM magic %q", magic)
	}
	if _, err := fmt.Fscan(reader, &width, &height, &maxValue); err != nil {
		return nil, err
	}
	if maxValue != 255 {
		return nil, fmt.Errorf("unsupported max value %d", maxValue)
	}
	if _, err := reader.ReadByte(); err != nil {
		return nil, err
	}
	canvas := NewCanvas(width, height)
	raw := make([]byte, width*height*3)
	if _, err := io.ReadFull(reader, raw); err != nil {
		return nil, err
	}
	for i := 0; i < width*height; i++ {
		canvas.Pixels[i] = V(
			srgbToLinear(float64(raw[i*3])/255),
			srgbToLinear(float64(raw[i*3+1])/255),
			srgbToLinear(float64(raw[i*3+2])/255),
		)
	}
	return canvas, nil
}
