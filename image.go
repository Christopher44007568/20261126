package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// Pixel is the uncompressed texel representation used by the demo.
type Pixel struct {
	R, G, B, A uint8
}

// Bitmap is a small RGBA raster. Width and height must be multiples of four
// for block compression.
type Bitmap struct {
	W, H int
	Pix  []Pixel
}

func NewBitmap(w, h int) *Bitmap {
	return &Bitmap{W: w, H: h, Pix: make([]Pixel, w*h)}
}

func (im *Bitmap) At(x, y int) Pixel {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return Pixel{}
	}
	return im.Pix[y*im.W+x]
}

func (im *Bitmap) Set(x, y int, p Pixel) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	im.Pix[y*im.W+x] = p
}

func (im *Bitmap) Block(x, y int) Block {
	var b Block
	for row := 0; row < BlockSide; row++ {
		for col := 0; col < BlockSide; col++ {
			b[row*BlockSide+col] = im.At(x+col, y+row)
		}
	}
	return b
}

func (im *Bitmap) SetBlock(x, y int, b Block) {
	for row := 0; row < BlockSide; row++ {
		for col := 0; col < BlockSide; col++ {
			im.Set(x+col, y+row, b[row*BlockSide+col])
		}
	}
}

func (im *Bitmap) ToRGBA() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, im.W, im.H))
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := im.At(x, y)
			out.SetRGBA(x, y, color.RGBA{R: p.R, G: p.G, B: p.B, A: p.A})
		}
	}
	return out
}

func writePNG(path string, im *Bitmap) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, im.ToRGBA()); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
