package mipmap

import "math"

// NewDiagnosticTexture creates a power-of-two tileable albedo and height map.
// Several features are centered on the periodic (0,0) corner, so tiling forms
// complete rings only when edge samples are handled continuously.
func NewDiagnosticTexture(size int, linearLight bool) (*RGBImage, *HeightField) {
	color := NewRGBImage(size, size)
	color.Linear = linearLight
	height := NewHeightField(size, size)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			u := float64(x) / float64(size)
			v := float64(y) / float64(size)

			h := diagnosticHeight(x, y, size, u, v)
			height.Set(x, y, h)

			sr, sg, sb := diagnosticSRGB(x, y, size, u, v, h)
			if linearLight {
				sr = sRGBToLinear(sr)
				sg = sRGBToLinear(sg)
				sb = sRGBToLinear(sb)
			}
			color.Set(x, y, sr, sg, sb)
		}
	}
	return color, height
}

func diagnosticHeight(x, y, size int, u, v float64) float64 {
	h := 0.52
	h += 0.16 * math.Sin(2*math.Pi*3*u+0.35) * math.Cos(2*math.Pi*2*v-0.5)
	h += 0.07 * math.Sin(2*math.Pi*7*u+2*math.Pi*5*v)
	h += 0.20 * periodicBump(x, y, 0, 0, size, float64(size)*0.10)
	h += 0.13 * periodicBump(x, y, size/2, size/2, size, float64(size)*0.075)
	return clamp01(h)
}

func diagnosticSRGB(x, y, size int, u, v, h float64) (float64, float64, float64) {
	wave := 0.5 + 0.5*math.Sin(2*math.Pi*4*u+0.8)*math.Cos(2*math.Pi*3*v-0.4)
	r := mix3(24, 38, wave)
	g := mix3(92, 174, wave)
	b := mix3(128, 190, wave)

	grid := gridMask(x, y, 32, 1.4)
	r = mix(r, 218, grid*0.72)
	g = mix(g, 232, grid*0.72)
	b = mix(b, 224, grid*0.72)

	seamStripe := math.Max(
		periodicLineMask(float64(x), 0, size, 5.0),
		periodicLineMask(float64(y), 0, size, 5.0),
	)
	r = mix(r, 255, seamStripe*0.82)
	g = mix(g, 84, seamStripe*0.82)
	b = mix(b, 168, seamStripe*0.82)

	seamRing := periodicRingMask(x, y, 0, 0, size, float64(size)*0.19, 2.2)
	r = mix(r, 255, seamRing)
	g = mix(g, 214, seamRing)
	b = mix(b, 64, seamRing)

	centerDot := periodicBump(x, y, size/2, size/2, size, float64(size)*0.055)
	r = mix(r, 255, centerDot)
	g = mix(g, 246, centerDot)
	b = mix(b, 210, centerDot)

	shade := 0.80 + 0.34*h
	return clampByte(r * shade), clampByte(g * shade), clampByte(b * shade)
}

func mix(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func mix3(a, b, t float64) float64 {
	return mix(float64(a), float64(b), t)
}

func clampByte(value float64) float64 {
	return math.Round(math.Max(0, math.Min(255, value))) / 255.0
}

func gridMask(x, y, spacing int, width float64) float64 {
	dx := x % spacing
	dy := y % spacing
	if dx > spacing/2 {
		dx = spacing - dx
	}
	if dy > spacing/2 {
		dy = spacing - dy
	}
	d := math.Min(float64(dx), float64(dy))
	return smoothMask(d, width)
}

func smoothMask(distance, width float64) float64 {
	if distance > width+1 {
		return 0
	}
	return clamp01(1 - distance/(width+0.75))
}

func periodicDelta(value, center float64, size int) float64 {
	delta := math.Abs(value - center)
	return math.Min(delta, float64(size)-delta)
}

func periodicLineMask(value, center float64, size int, width float64) float64 {
	return smoothMask(periodicDelta(value, center, size), width)
}

func periodicDistance(x, y, cx, cy, size int) float64 {
	dx := periodicDelta(float64(x), float64(cx), size)
	dy := periodicDelta(float64(y), float64(cy), size)
	return math.Hypot(dx, dy)
}

func periodicBump(x, y, cx, cy, size int, radius float64) float64 {
	d := periodicDistance(x, y, cx, cy, size)
	if d >= radius {
		return 0
	}
	t := d / radius
	return 0.5 + 0.5*math.Cos(math.Pi*t)
}

func periodicRingMask(x, y, cx, cy, size int, radius, width float64) float64 {
	d := periodicDistance(x, y, cx, cy, size)
	return smoothMask(math.Abs(d-radius), width)
}
