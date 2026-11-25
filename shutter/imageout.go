package shutter

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// ToNRGBA converts a linear RGB frame plus coverage into an opaque NRGBA
// image; uncovered pixels use the supplied background color.
func ToNRGBA(width, height int, frame []RGB, cov []float64, bg color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			a := clamp01(cov[i])
			fg := frame[i]
			c := color.NRGBA{
				R: blendChan(fg.R, float64(bg.R)/255, a),
				G: blendChan(fg.G, float64(bg.G)/255, a),
				B: blendChan(fg.B, float64(bg.B)/255, a),
				A: 255,
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func blendChan(fg, bg, a float64) uint8 {
	v := fg*a + bg*(1-a)
	return uint8(clamp01(v)*255 + 0.5)
}

// EncodePNG writes img to w as a PNG using only the standard library.
func EncodePNG(w io.Writer, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// VectorColor maps a motion vector to an RGBA color using a wheel where hue
// encodes direction and saturation encodes magnitude relative to maxLen.
// Zero vectors render near-black so a static background reads as no motion.
func VectorColor(vx, vy, maxLen float64) color.NRGBA {
	mag := math.Hypot(vx, vy)
	if mag < 1e-9 || maxLen <= 0 {
		return color.NRGBA{A: 255}
	}
	angle := math.Atan2(vy, vx)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	hue := angle / (2 * math.Pi)
	sat := clamp01(mag / maxLen)
	r, g, b := hsvToRGB(hue, sat, 1)
	return color.NRGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}

// VectorFieldNRGBA renders the vector field as a color-coded image.
func VectorFieldNRGBA(field *VectorField) *image.NRGBA {
	maxLen := 0.0
	for i := range field.X {
		if l := math.Hypot(field.X[i], field.Y[i]); l > maxLen {
			maxLen = l
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, field.Width, field.Height))
	for y := 0; y < field.Height; y++ {
		for x := 0; x < field.Width; x++ {
			i := y*field.Width + x
			img.SetNRGBA(x, y, VectorColor(field.X[i], field.Y[i], maxLen))
		}
	}
	return img
}

// MaxVectorLength returns the largest vector magnitude in the field.
func (f *VectorField) MaxVectorLength() float64 {
	m := 0.0
	for i := range f.X {
		if l := math.Hypot(f.X[i], f.Y[i]); l > m {
			m = l
		}
	}
	return m
}

// hsvToRGB converts h,s,v in [0,1] to r,g,b in [0,1].
func hsvToRGB(h, s, v float64) (r, g, b float64) {
	i := math.Floor(h * 6)
	f := h*6 - i
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

// WriteVectorBinary serializes the field in a small documented binary format:
// magic "MVF1", uint16 width/height, then float64 x,y pairs row-major.
func WriteVectorBinary(w io.Writer, field *VectorField) error {
	buf := make([]byte, 0, 8+4+16*field.Width*field.Height)
	buf = append(buf, 'M', 'V', 'F', '1')
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], uint16(field.Width))
	binary.LittleEndian.PutUint16(hdr[2:4], uint16(field.Height))
	buf = append(buf, hdr[:]...)
	var pair [16]byte
	for i := range field.X {
		binary.LittleEndian.PutUint64(pair[0:8], math.Float64bits(field.X[i]))
		binary.LittleEndian.PutUint64(pair[8:16], math.Float64bits(field.Y[i]))
		buf = append(buf, pair[:]...)
	}
	_, err := w.Write(buf)
	return err
}
