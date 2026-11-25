package internal

import (
	"image"
	"image/png"
	"math"
	"os"
)

// toSRGB applies a linear-to-sRGB gamma for display, since the renderer works
// in linear light.
func toSRGB(c float64) uint8 {
	c = Clamp(c, 0, 1)
	var v float64
	if c <= 0.0031308 {
		v = 12.92 * c
	} else {
		v = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(v*255 + 0.5)
}

// WritePNG encodes an RGBA buffer to path.
func WritePNG(path string, buf []RGBA, w, h int) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, c := range buf {
		off := i * 4
		img.Pix[off+0] = toSRGB(c.R)
		img.Pix[off+1] = toSRGB(c.G)
		img.Pix[off+2] = toSRGB(c.B)
		img.Pix[off+3] = 0xff
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
