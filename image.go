package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Color struct {
	R, G, B uint8
}

func rgb(r, g, b int) Color {
	return Color{R: clampByte(r), G: clampByte(g), B: clampByte(b)}
}

func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (c Color) withAlpha(a uint8) RGBA {
	return RGBA{R: c.R, G: c.G, B: c.B, A: a}
}

type RGBA struct {
	R, G, B, A uint8
}

type Image struct {
	W, H int
	Pix  []uint8
}

func NewImage(w, h int, background Color) *Image {
	im := &Image{W: w, H: h, Pix: make([]uint8, w*h*4)}
	im.Clear(background)
	return im
}

func (im *Image) Clear(c Color) {
	for i := 0; i < len(im.Pix); i += 4 {
		im.Pix[i] = c.R
		im.Pix[i+1] = c.G
		im.Pix[i+2] = c.B
		im.Pix[i+3] = 255
	}
}

func (im *Image) pixelIndex(x, y int) int {
	return (y*im.W + x) * 4
}

func (im *Image) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < im.W && y < im.H
}

func (im *Image) Set(x, y int, c Color) {
	if !im.InBounds(x, y) {
		return
	}
	i := im.pixelIndex(x, y)
	im.Pix[i] = c.R
	im.Pix[i+1] = c.G
	im.Pix[i+2] = c.B
	im.Pix[i+3] = 255
}

func (im *Image) Blend(x, y int, c RGBA) {
	if !im.InBounds(x, y) || c.A == 0 {
		return
	}
	i := im.pixelIndex(x, y)
	a := uint32(c.A)
	inv := 255 - a
	im.Pix[i] = uint8((uint32(c.R)*a + uint32(im.Pix[i])*inv) / 255)
	im.Pix[i+1] = uint8((uint32(c.G)*a + uint32(im.Pix[i+1])*inv) / 255)
	im.Pix[i+2] = uint8((uint32(c.B)*a + uint32(im.Pix[i+2])*inv) / 255)
	im.Pix[i+3] = 255
}

func (im *Image) FillRect(x, y, w, h int, c Color) {
	im.FillRectAlpha(x, y, w, h, c.withAlpha(255))
}

func (im *Image) FillRectAlpha(x, y, w, h int, c RGBA) {
	for yy := maxInt(y, 0); yy < y+h && yy < im.H; yy++ {
		for xx := maxInt(x, 0); xx < x+w && xx < im.W; xx++ {
			im.Blend(xx, yy, c)
		}
	}
}

func (im *Image) FrameRect(x, y, w, h int, c Color) {
	for xx := x; xx < x+w; xx++ {
		im.Set(xx, y, c)
		im.Set(xx, y+h-1, c)
	}
	for yy := y; yy < y+h; yy++ {
		im.Set(x, yy, c)
		im.Set(x+w-1, yy, c)
	}
}

func (im *Image) HorizontalLine(x, y, w int, c Color) {
	for xx := x; xx < x+w; xx++ {
		im.Set(xx, y, c)
	}
}

func (im *Image) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", im.W, im.H); err != nil {
		return err
	}
	row := make([]byte, im.W*3)
	for y := 0; y < im.H; y++ {
		base := y * im.W * 4
		for x := 0; x < im.W; x++ {
			src := base + x*4
			dst := x * 3
			row[dst] = im.Pix[src]
			row[dst+1] = im.Pix[src+1]
			row[dst+2] = im.Pix[src+2]
		}
		if _, err := bw.Write(row); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func WritePPMFile(path string, im *Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := im.WritePPM(f); err != nil {
		return err
	}
	return f.Close()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
