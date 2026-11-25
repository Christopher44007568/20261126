package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Canvas struct {
	W, H int
	Pix  []byte
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pix: make([]byte, w*h*3)}
}

func (c *Canvas) setRGB(x, y int, r, g, b uint8) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i+0] = r
	c.Pix[i+1] = g
	c.Pix[i+2] = b
}

func (c *Canvas) blend(x, y int, r, g, b uint8, alpha float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	i := (y*c.W + x) * 3
	a := alpha
	c.Pix[i+0] = uint8(float64(c.Pix[i+0])*(1-a) + float64(r)*a + 0.5)
	c.Pix[i+1] = uint8(float64(c.Pix[i+1])*(1-a) + float64(g)*a + 0.5)
	c.Pix[i+2] = uint8(float64(c.Pix[i+2])*(1-a) + float64(b)*a + 0.5)
}

func (c *Canvas) Fill(r, g, b uint8) {
	for i := 0; i < len(c.Pix); i += 3 {
		c.Pix[i+0] = r
		c.Pix[i+1] = g
		c.Pix[i+2] = b
	}
}

func (c *Canvas) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	if _, err := bw.Write(c.Pix); err != nil {
		return err
	}
	return bw.Flush()
}

func (c *Canvas) SavePPM(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := c.WritePPM(f); err != nil {
		return err
	}
	return f.Close()
}
