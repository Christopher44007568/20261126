package render

import "math"

// RGB is a display-referred, linear-in-code source sample. Values may exceed
// one for the HDR highlight scene; clipping happens in the bit-depth pipeline.
type RGB struct {
	R float64
	G float64
	B float64
}

type Pipeline struct {
	Bits   int
	Name   string
	Short  string
	Dither bool
}

var Pipelines = []Pipeline{
	{Bits: 8, Name: "8-BIT DIRECT", Short: "8-BIT", Dither: false},
	{Bits: 10, Name: "10-BIT TO 8-BIT", Short: "10 TO 8", Dither: true},
	{Bits: 12, Name: "12-BIT TO 8-BIT", Short: "12 TO 8", Dither: true},
}

var bayer4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
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

func ClampCode(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// Quantize first snaps the source to the selected integer pipeline. The result
// remains normalized so it can be inspected as a 16-bit master as well.
func Quantize(v float64, bits int) float64 {
	levels := float64((int(1) << bits) - 1)
	return math.Round(Clamp01(v)*levels) / levels
}

// DisplayCode converts a source sample to the final 8-bit display. The 10- and
// 12-bit paths retain their fractional 8-bit coordinate and deposit it with a
// 4x4 Bayer pattern. This is what turns extra master codes into visible levels;
// hard 8-bit output still has only 256 integer codes.
func DisplayCode(v float64, p Pipeline, x, y int) int {
	q := Quantize(v, p.Bits) * 255
	if !p.Dither || p.Bits == 8 {
		return ClampCode(int(math.Floor(q + 0.5)))
	}
	base := math.Floor(q)
	frac := q - base
	threshold := (float64(bayer4[y&3][x&3]) + 0.5) / 16
	if frac > threshold {
		base++
	}
	return ClampCode(int(base))
}

func DisplayRGB(src RGB, p Pipeline, x, y int) RGB {
	return RGB{
		R: float64(DisplayCode(src.R, p, x, y)),
		G: float64(DisplayCode(src.G, p, x, y)),
		B: float64(DisplayCode(src.B, p, x, y)),
	}
}

func HardCode(v float64, bits int) int {
	return ClampCode(int(math.Floor(Quantize(v, bits)*255 + 0.5)))
}

func HSV(hue, sat, val float64) RGB {
	h := math.Mod(hue, 360)
	if h < 0 {
		h += 360
	}
	s := Clamp01(sat)
	v := Clamp01(val)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return RGB{R: r + m, G: g + m, B: b + m}
}

func RGBHue(c RGB) (float64, float64) {
	maxv := math.Max(c.R, math.Max(c.G, c.B))
	minv := math.Min(c.R, math.Min(c.G, c.B))
	delta := maxv - minv
	if delta <= 1e-12 || maxv <= 1e-12 {
		return 0, 0
	}
	var h float64
	switch maxv {
	case c.R:
		h = 60 * math.Mod((c.G-c.B)/delta, 6)
	case c.G:
		h = 60 * ((c.B-c.R)/delta + 2)
	default:
		h = 60 * ((c.R-c.G)/delta + 4)
	}
	if h < 0 {
		h += 360
	}
	return math.Mod(h, 360), delta / maxv
}

func HueDistance(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func Mix(a, b RGB, t float64) RGB {
	t = Clamp01(t)
	return RGB{
		R: Lerp(a.R, b.R, t),
		G: Lerp(a.G, b.G, t),
		B: Lerp(a.B, b.B, t),
	}
}
