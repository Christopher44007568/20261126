// Package render builds comparison and slice visualizations for 3D
// LUTs using only the standard library image packages.
package render

import (
	"image"
	"image/color"
	"math"

	"luttool/lut"
)

// MapFunc transforms one color; r,g,b are in [0,1].
type MapFunc func(r, g, b float64) (float64, float64, float64)

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func toNRGBA(v float64) uint8 {
	return uint8(math.Round(clamp01(v) * 255))
}

// TestImage builds a procedural test chart: a hue/value field, a
// neutral ramp, and fully saturated primaries, so channel swaps and
// domain shifts are all visible.
func TestImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rampH := h / 5
	patchH := h / 5
	fieldH := h - rampH - patchH
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b float64
			switch {
			case y < fieldH:
				hue := float64(x) / float64(w-1)
				val := 1 - float64(y)/float64(fieldH-1)
				r, g, b = hsv(hue, 0.85, 0.25+0.75*val)
			case y < fieldH+rampH:
				v := float64(x) / float64(w-1)
				r, g, b = v, v, v
			default:
				seg := x * 6 / w
				r, g, b = primary(seg)
			}
			img.SetNRGBA(x, y, color.NRGBA{toNRGBA(r), toNRGBA(g), toNRGBA(b), 255})
		}
	}
	return img
}

func primary(seg int) (float64, float64, float64) {
	switch seg {
	case 0:
		return 1, 0, 0
	case 1:
		return 1, 1, 0
	case 2:
		return 0, 1, 0
	case 3:
		return 0, 1, 1
	case 4:
		return 0, 0, 1
	default:
		return 1, 0, 1
	}
}

// hsv converts h,s,v in [0,1] to rgb in [0,1].
func hsv(h, s, v float64) (float64, float64, float64) {
	h = h - math.Floor(h)
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch i % 6 {
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

// Apply maps every pixel of src through fn.
func Apply(src *image.NRGBA, fn MapFunc) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.NRGBAAt(x, y)
			r, g, bl := fn(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255)
			out.SetNRGBA(x, y, color.NRGBA{toNRGBA(r), toNRGBA(g), toNRGBA(bl), 255})
		}
	}
	return out
}

// Diff renders |a-b| per channel, amplified by gain, on black.
func Diff(a, b *image.NRGBA, gain float64) *image.NRGBA {
	bd := a.Bounds()
	out := image.NewNRGBA(bd)
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			ca, cb := a.NRGBAAt(x, y), b.NRGBAAt(x, y)
			d := func(u, v uint8) uint8 {
				df := math.Abs(float64(u)-float64(v)) / 255 * gain
				return toNRGBA(df)
			}
			out.SetNRGBA(x, y, color.NRGBA{d(ca.R, cb.R), d(ca.G, cb.G), d(ca.B, cb.B), 255})
		}
	}
	return out
}

// HStack concatenates images horizontally with a dark gap.
func HStack(gap int, imgs ...*image.NRGBA) *image.NRGBA {
	w, h := 0, 0
	for _, im := range imgs {
		w += im.Bounds().Dx()
		if im.Bounds().Dy() > h {
			h = im.Bounds().Dy()
		}
	}
	w += gap * (len(imgs) - 1)
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill := color.NRGBA{24, 24, 24, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(x, y, fill)
		}
	}
	ox := 0
	for _, im := range imgs {
		for y := 0; y < im.Bounds().Dy(); y++ {
			for x := 0; x < im.Bounds().Dx(); x++ {
				out.SetNRGBA(ox+x, y, im.NRGBAAt(im.Bounds().Min.X+x, im.Bounds().Min.Y+y))
			}
		}
		ox += im.Bounds().Dx() + gap
	}
	return out
}

// VStack concatenates images vertically with a dark gap.
func VStack(gap int, imgs ...*image.NRGBA) *image.NRGBA {
	w, h := 0, 0
	for _, im := range imgs {
		h += im.Bounds().Dy()
		if im.Bounds().Dx() > w {
			w = im.Bounds().Dx()
		}
	}
	h += gap * (len(imgs) - 1)
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	oy := 0
	for _, im := range imgs {
		for y := 0; y < im.Bounds().Dy(); y++ {
			for x := 0; x < im.Bounds().Dx(); x++ {
				out.SetNRGBA(x, oy+y, im.NRGBAAt(im.Bounds().Min.X+x, im.Bounds().Min.Y+y))
			}
		}
		oy += im.Bounds().Dy() + gap
	}
	return out
}

// SliceAt renders one interpolated slice of the LUT at a fixed blue
// value: x sweeps red, y sweeps green (top = 1), each pixel trilinearly
// interpolated through the full table and domain.
func SliceAt(l *lut.LUT, blue float64, res int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, res, res))
	for y := 0; y < res; y++ {
		g := 1 - float64(y)/float64(res-1)
		for x := 0; x < res; x++ {
			r := float64(x) / float64(res-1)
			or, og, ob := l.Apply(r, g, blue)
			img.SetNRGBA(x, y, color.NRGBA{toNRGBA(or), toNRGBA(og), toNRGBA(ob), 255})
		}
	}
	return img
}
