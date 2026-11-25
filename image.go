package main

import (
	"bufio"
	"math"
	"os"
)

type Image struct {
	W, H   int
	Pixels []Vec3 // linear RGB, row-major from top
}

func NewImage(w, h int) *Image {
	return &Image{w, h, make([]Vec3, w*h)}
}

func (im *Image) Set(x, y int, c Vec3) { im.Pixels[y*im.W+x] = c }

// gammaEncode applies a filmic-ish shoulder plus sRGB gamma so HDR radiance
// maps to displayable bytes without clipping every highlight.
func gammaEncode(x float64) int {
	x = x / (1 + x) // Reinhard tone map
	v := math.Pow(clamp(x), 1/2.2)
	return int(v*255 + 0.5)
}

// WritePPM writes binary P6 PPM, which needs only the Go standard library.
func (im *Image) WritePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.WriteString("P6\n"); err != nil {
		return err
	}
	if _, err := w.WriteString(itoa(im.W) + " " + itoa(im.H) + "\n255\n"); err != nil {
		return err
	}
	buf := make([]byte, 0, im.W*im.H*3)
	for _, c := range im.Pixels {
		buf = append(buf, byte(gammaEncode(c.X)), byte(gammaEncode(c.Y)), byte(gammaEncode(c.Z)))
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return w.Flush()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
