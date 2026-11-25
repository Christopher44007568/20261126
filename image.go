package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
)

// Image is a float64 RGB image, channels interleaved, values in [0,1].
type Image struct {
	W, H int
	Pix  []float64
}

func NewImage(w, h int) *Image {
	return &Image{W: w, H: h, Pix: make([]float64, w*h*3)}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (im *Image) Set(x, y int, r, g, b float64) {
	i := (y*im.W + x) * 3
	im.Pix[i] = clamp01(r)
	im.Pix[i+1] = clamp01(g)
	im.Pix[i+2] = clamp01(b)
}

func (im *Image) Get(x, y int) (float64, float64, float64) {
	i := (y*im.W + x) * 3
	return im.Pix[i], im.Pix[i+1], im.Pix[i+2]
}

// SavePPM writes a binary P6 PPM.
func (im *Image) SavePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", im.W, im.H); err != nil {
		return err
	}
	buf := make([]byte, len(im.Pix))
	for i, v := range im.Pix {
		buf[i] = uint8(math.Round(clamp01(v) * 255))
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return w.Flush()
}

// Clone returns a deep copy.
func (im *Image) Clone() *Image {
	out := NewImage(im.W, im.H)
	copy(out.Pix, im.Pix)
	return out
}
