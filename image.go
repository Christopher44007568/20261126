package hdrlab

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// Img is a float64 RGB image. Display-ready images store gamma-encoded
// values in [0,1]; radiance (HDR) images store linear values that may
// exceed 1.
type Img struct {
	W, H int
	Pix  []float64 // interleaved RGB
}

func NewImg(w, h int) *Img {
	return &Img{W: w, H: h, Pix: make([]float64, w*h*3)}
}

func (im *Img) Set(x, y int, r, g, b float64) {
	i := (y*im.W + x) * 3
	im.Pix[i], im.Pix[i+1], im.Pix[i+2] = r, g, b
}

func (im *Img) Get(x, y int) (r, g, b float64) {
	i := (y*im.W + x) * 3
	return im.Pix[i], im.Pix[i+1], im.Pix[i+2]
}

// GetClamped samples with edge clamping so out-of-range reads are safe.
func (im *Img) GetClamped(x, y int) (r, g, b float64) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= im.W {
		x = im.W - 1
	}
	if y >= im.H {
		y = im.H - 1
	}
	return im.Get(x, y)
}

// Luma returns Rec.709 luma of a pixel.
func (im *Img) Luma(x, y int) float64 {
	i := (y*im.W + x) * 3
	return 0.2126*im.Pix[i] + 0.7152*im.Pix[i+1] + 0.0722*im.Pix[i+2]
}

func (im *Img) Clone() *Img {
	out := NewImg(im.W, im.H)
	copy(out.Pix, im.Pix)
	return out
}

func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// WritePPM writes a binary P6 PPM, clamping values to [0,1].
func WritePPM(path string, im *Img) error {
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
		buf[i] = uint8(Clamp01(v)*255 + 0.5)
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return w.Flush()
}

// ToRGBA converts to an 8-bit image for PNG encoding.
func (im *Img) ToRGBA() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, im.W, im.H))
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			r, g, b := im.Get(x, y)
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(Clamp01(r)*255 + 0.5),
				G: uint8(Clamp01(g)*255 + 0.5),
				B: uint8(Clamp01(b)*255 + 0.5),
				A: 255,
			})
		}
	}
	return out
}

func WritePNG(path string, im *Img) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, im.ToRGBA())
}

// SaveImage writes both PPM and PNG versions of an image.
func SaveImage(base string, im *Img) error {
	if err := WritePPM(base+".ppm", im); err != nil {
		return err
	}
	return WritePNG(base+".png", im)
}
