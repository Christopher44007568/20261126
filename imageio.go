package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// scaleNearest 用最近邻把图像放大 factor 倍，保持色块边界锐利。
func scaleNearest(src image.Image, factor int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*factor, b.Dy()*factor))
	for y := 0; y < b.Dy()*factor; y++ {
		for x := 0; x < b.Dx()*factor; x++ {
			dst.Set(x, y, src.At(b.Min.X+x/factor, b.Min.Y+y/factor))
		}
	}
	return dst
}

// savePNG 输出放大后的 PNG 到 outDir。
func savePNG(outDir, name string, src image.Image, scale int) error {
	dst := scaleNearest(src, scale)
	f, err := os.Create(filepath.Join(outDir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, dst)
}
