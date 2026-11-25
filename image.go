package main

import (
	"bufio"
	"hash/fnv"
	"io"
)

// Color 是 8-bit/通道的 RGBA 颜色。
type Color struct {
	R, G, B, A uint8
}

// RGB 构造不透明颜色。
func RGB(r, g, b uint8) Color { return Color{r, g, b, 255} }

// Image 是逐帧新建的像素缓冲；不保留任何跨帧像素，
// 从结构上保证“切回已卸载区域时不能复用旧像素”。
type Image struct {
	W, H int
	Pix  []uint8 // 长度 W*H*4，行优先，RGBA
}

// NewImage 返回已清零（全透明黑）的新画布。
func NewImage(w, h int) *Image {
	return &Image{W: w, H: h, Pix: make([]uint8, w*h*4)}
}

func (im *Image) idx(x, y int) int { return (y*im.W + x) * 4 }

// Set 直接写入不透明像素。
func (im *Image) Set(x, y int, c Color) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	i := im.idx(x, y)
	im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = c.R, c.G, c.B, c.A
}

// BlendOver 以 source-alpha 混合方式覆盖当前像素。
func (im *Image) BlendOver(x, y int, c Color) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	i := im.idx(x, y)
	sa := uint32(c.A)
	da := uint32(im.Pix[i+3])
	outA := sa + da*(255-sa)/255
	if outA == 0 {
		return
	}
	blend := func(s, d uint8) uint8 {
		return uint8((uint32(s)*sa + uint32(d)*da*(255-sa)/255) / outA)
	}
	im.Pix[i] = blend(c.R, im.Pix[i])
	im.Pix[i+1] = blend(c.G, im.Pix[i+1])
	im.Pix[i+2] = blend(c.B, im.Pix[i+2])
	im.Pix[i+3] = uint8(outA)
}

// FillRect 用纯色 alpha 混合填充矩形（用于状态色叠加）。
func (im *Image) FillRect(x, y, w, h int, c Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			im.BlendOver(xx, yy, c)
		}
	}
}

// SolidRect 写入不透明矩形。
func (im *Image) SolidRect(x, y, w, h int, c Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			im.Set(xx, yy, c)
		}
	}
}

// Frame 画 1 像素描边。
func (im *Image) Frame(x, y, w, h int, c Color) {
	for xx := x; xx < x+w; xx++ {
		im.Set(xx, y, c)
		im.Set(xx, y+h-1, c)
	}
	for yy := y; yy < y+h; yy++ {
		im.Set(x, yy, c)
		im.Set(x+w-1, yy, c)
	}
}

// RegionHash 计算矩形区域像素的 FNV-1a 哈希，用于断言像素不会被复用。
func (im *Image) RegionHash(x, y, w, h int) uint64 {
	hh := fnv.New64a()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx < 0 || yy < 0 || xx >= im.W || yy >= im.H {
				continue
			}
			i := im.idx(xx, yy)
			_, _ = hh.Write(im.Pix[i : i+4])
		}
	}
	return hh.Sum64()
}

// WritePPM 以二进制 P6 格式写出 PPM（零外部依赖）。
func (im *Image) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := bw.WriteString("P6\n"); err != nil {
		return err
	}
	if _, err := bw.WriteString("# lazyscene on-demand payload frame\n"); err != nil {
		return err
	}
	if _, err := bw.WriteString(itoa(im.W) + " " + itoa(im.H) + "\n255\n"); err != nil {
		return err
	}
	row := make([]byte, 0, im.W*3)
	for y := 0; y < im.H; y++ {
		row = row[:0]
		for x := 0; x < im.W; x++ {
			i := im.idx(x, y)
			row = append(row, im.Pix[i], im.Pix[i+1], im.Pix[i+2])
		}
		if _, err := bw.Write(row); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
