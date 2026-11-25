package main

import (
	"image/color"
	"math"
)

type RGB struct {
	R uint8
	G uint8
	B uint8
}

type Lab struct {
	L float64
	A float64
	B float64
}

func rgbFromColor(c color.Color) RGB {
	r, g, b, _ := c.RGBA()
	return RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
}

func (c RGB) color() color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
}

func (c RGB) lab() Lab {
	return rgbToLab(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255)
}

func (c RGB) relativeLuminance() float64 {
	return 0.2126729*srgbLinear(float64(c.R)/255) +
		0.7151522*srgbLinear(float64(c.G)/255) +
		0.0721750*srgbLinear(float64(c.B)/255)
}

func srgbLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

func rgbToLab(r, g, b float64) Lab {
	rl := srgbLinear(r)
	gl := srgbLinear(g)
	bl := srgbLinear(b)

	x := 0.4124564*rl + 0.3575761*gl + 0.1804375*bl
	y := 0.2126729*rl + 0.7151522*gl + 0.0721750*bl
	z := 0.0193339*rl + 0.1191920*gl + 0.9503041*bl

	xn := labPivot(x / 0.95047)
	yn := labPivot(y)
	zn := labPivot(z / 1.08883)

	return Lab{
		L: 116*yn - 16,
		A: 500 * (xn - yn),
		B: 200 * (yn - zn),
	}
}

func labPivot(v float64) float64 {
	const threshold = 0.008856
	if v > threshold {
		return math.Cbrt(v)
	}
	return 7.787*v + 16.0/116.0
}

func deltaE2000(a, b Lab) float64 {
	const pow25To7 = 6103515625.0

	c1 := math.Hypot(a.A, a.B)
	c2 := math.Hypot(b.A, b.B)
	cBar := (c1 + c2) / 2
	cBar7 := math.Pow(cBar, 7)
	g := 0.5 * (1 - math.Sqrt(cBar7/(cBar7+pow25To7)))

	a1p := (1 + g) * a.A
	a2p := (1 + g) * b.A
	c1p := math.Hypot(a1p, a.B)
	c2p := math.Hypot(a2p, b.B)

	h1p := hueDegrees(a.B, a1p)
	h2p := hueDegrees(b.B, a2p)
	dhp := 0.0
	if c1p*c2p != 0 {
		dhp = h2p - h1p
		if dhp > 180 {
			dhp -= 360
		} else if dhp < -180 {
			dhp += 360
		}
	}

	dL := b.L - a.L
	dC := c2p - c1p
	dH := 2 * math.Sqrt(c1p*c2p) * math.Sin(degToRad(dhp)/2)
	lBar := (a.L + b.L) / 2
	cBarP := (c1p + c2p) / 2
	hBarP := h1p + h2p
	if math.Abs(h1p-h2p) > 180 {
		if h1p+h2p < 360 {
			hBarP += 360
		} else {
			hBarP -= 360
		}
	}
	hBarP /= 2

	t := 1 -
		0.17*math.Cos(degToRad(hBarP-30)) +
		0.24*math.Cos(degToRad(2*hBarP)) +
		0.32*math.Cos(degToRad(3*hBarP+6)) -
		0.20*math.Cos(degToRad(4*hBarP-63))
	dTheta := 30 * math.Exp(-math.Pow((hBarP-275)/25, 2))
	rc := 2 * math.Sqrt(math.Pow(cBarP, 7)/(math.Pow(cBarP, 7)+pow25To7))
	rt := -math.Sin(degToRad(2*dTheta)) * rc
	lDelta := lBar - 50
	sl := 1 + 0.015*lDelta*lDelta/math.Sqrt(20+lDelta*lDelta)
	sc := 1 + 0.045*cBarP
	sh := 1 + 0.015*cBarP*t

	termL := dL / sl
	termC := dC / sc
	termH := dH / sh
	return math.Sqrt(termL*termL + termC*termC + termH*termH + rt*termC*termH)
}

func hueDegrees(b, a float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	h := math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}

func degToRad(v float64) float64 {
	return v * math.Pi / 180
}

func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}
