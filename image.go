package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// RGB 是 8-bit 不透明颜色。
type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

func hexRGB(value string) RGB {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		panic("invalid rgb color: " + value)
	}
	var c RGB
	if _, err := fmt.Sscanf(value, "%02x%02x%02x", &c.R, &c.G, &c.B); err != nil {
		panic("invalid rgb color " + value + ": " + err.Error())
	}
	return c
}

// Image 是 RGBA 画布。渲染每次都新建画布，避免跨 variant 残留像素。
type Image struct {
	W   int
	H   int
	Pix []uint8
}

func NewImage(w, h int) *Image {
	return &Image{W: w, H: h, Pix: make([]uint8, w*h*4)}
}

func (im *Image) offset(x, y int) int { return (y*im.W + x) * 4 }

func (im *Image) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < im.W && y < im.H
}

func (im *Image) Set(x, y int, c RGB) {
	if !im.InBounds(x, y) {
		return
	}
	i := im.offset(x, y)
	im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = c.R, c.G, c.B, 255
}

// Blend 设置带 source-alpha 的像素；alpha 为 255 时等价于覆盖。
func (im *Image) Blend(x, y int, c RGB, alpha uint8) {
	if !im.InBounds(x, y) {
		return
	}
	i := im.offset(x, y)
	if alpha == 255 {
		im.Set(x, y, c)
		return
	}
	a := uint32(alpha)
	inv := 255 - a
	im.Pix[i] = uint8((uint32(c.R)*a + uint32(im.Pix[i])*inv) / 255)
	im.Pix[i+1] = uint8((uint32(c.G)*a + uint32(im.Pix[i+1])*inv) / 255)
	im.Pix[i+2] = uint8((uint32(c.B)*a + uint32(im.Pix[i+2])*inv) / 255)
	im.Pix[i+3] = 255
}

func (im *Image) Fill(c RGB) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			im.Set(x, y, c)
		}
	}
}

func (im *Image) FillRect(r Rect, c RGB) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			im.Set(x, y, c)
		}
	}
}

func (im *Image) BlendRect(r Rect, c RGB, alpha uint8) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			im.Blend(x, y, c, alpha)
		}
	}
}

func (im *Image) FrameRect(r Rect, thickness int, c RGB) {
	if thickness < 1 {
		thickness = 1
	}
	im.FillRect(Rect{X: r.X, Y: r.Y, W: r.W, H: thickness}, c)
	im.FillRect(Rect{X: r.X, Y: r.Y + r.H - thickness, W: r.W, H: thickness}, c)
	im.FillRect(Rect{X: r.X, Y: r.Y, W: thickness, H: r.H}, c)
	im.FillRect(Rect{X: r.X + r.W - thickness, Y: r.Y, W: thickness, H: r.H}, c)
}

// RGBBytes 返回紧凑 RGB 像素流，用于哈希和 PPM 编码。
func (im *Image) RGBBytes() []byte {
	out := make([]byte, im.W*im.H*3)
	j := 0
	for i := 0; i < len(im.Pix); i += 4 {
		out[j] = im.Pix[i]
		out[j+1] = im.Pix[i+1]
		out[j+2] = im.Pix[i+2]
		j += 3
	}
	return out
}

func (im *Image) PixelSHA256() string {
	sum := sha256.Sum256(im.RGBBytes())
	return hex.EncodeToString(sum[:])
}

// WritePPM 写出二进制 P6 PPM，不依赖 image/png 或任何第三方库。
func (im *Image) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", im.W, im.H); err != nil {
		return err
	}
	rgb := im.RGBBytes()
	if _, err := bw.Write(rgb); err != nil {
		return err
	}
	return bw.Flush()
}

func WritePPMFile(path string, im *Image) error {
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
