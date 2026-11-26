package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type RGB struct {
	R, G, B uint8
}

var (
	colorBackgroundTop    = RGB{18, 27, 48}
	colorBackgroundBottom = RGB{74, 91, 124}
	colorErrorOuter       = RGB{255, 0, 255}
	colorErrorCore        = RGB{255, 244, 92}
	colorFalseHit         = RGB{0, 229, 255}
)

type Image struct {
	Width  int
	Height int
	Pixels []RGB
}

func NewImage(width, height int) *Image {
	return &Image{Width: width, Height: height, Pixels: make([]RGB, width*height)}
}

func (img *Image) inBounds(x, y int) bool {
	return x >= 0 && x < img.Width && y >= 0 && y < img.Height
}

func (img *Image) Set(x, y int, c RGB) {
	if img.inBounds(x, y) {
		img.Pixels[y*img.Width+x] = c
	}
}

func (img *Image) At(x, y int) RGB {
	if !img.inBounds(x, y) {
		return RGB{}
	}
	return img.Pixels[y*img.Width+x]
}

func (img *Image) Fill(c RGB) {
	for i := range img.Pixels {
		img.Pixels[i] = c
	}
}

func WritePPM(path string, img *Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", img.Width, img.Height); err != nil {
		return err
	}
	for i := range img.Pixels {
		if err := w.WriteByte(img.Pixels[i].R); err != nil {
			return err
		}
		if err := w.WriteByte(img.Pixels[i].G); err != nil {
			return err
		}
		if err := w.WriteByte(img.Pixels[i].B); err != nil {
			return err
		}
	}
	return w.Flush()
}

// ReadPPM decodes the small P6 files emitted by this program. It is kept in
// the main package so tests can validate the exact on-disk artifact.
func ReadPPM(path string) (*Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := bufio.NewReader(file)
	var magic string
	if _, err := fmt.Fscanf(r, "%2s\n", &magic); err != nil {
		return nil, err
	}
	if magic != "P6" {
		return nil, fmt.Errorf("unsupported PPM magic %q", magic)
	}

	width, height, maxValue := 0, 0, 0
	if _, err := fmt.Fscanf(r, "%d %d\n%d\n", &width, &height, &maxValue); err != nil {
		return nil, err
	}
	if maxValue != 255 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid PPM header %dx%d max=%d", width, height, maxValue)
	}

	img := NewImage(width, height)
	raw := make([]byte, len(img.Pixels)*3)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, err
	}
	for i := range img.Pixels {
		img.Pixels[i] = RGB{R: raw[i*3], G: raw[i*3+1], B: raw[i*3+2]}
	}
	return img, nil
}
