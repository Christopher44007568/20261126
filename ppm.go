package main

import (
	"bufio"
	"fmt"
	"os"
)

type RGB struct {
	R, G, B uint8
}

type Image struct {
	Width  int
	Height int
	Pixels []RGB
}

func NewImage(width, height int) *Image {
	return &Image{Width: width, Height: height, Pixels: make([]RGB, width*height)}
}

func (img *Image) Set(x, y int, c RGB) {
	if x < 0 || x >= img.Width || y < 0 || y >= img.Height {
		return
	}
	img.Pixels[y*img.Width+x] = c
}

func (img *Image) At(x, y int) RGB {
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
	for _, p := range img.Pixels {
		if err := w.WriteByte(p.R); err != nil {
			return err
		}
		if err := w.WriteByte(p.G); err != nil {
			return err
		}
		if err := w.WriteByte(p.B); err != nil {
			return err
		}
	}
	return w.Flush()
}
