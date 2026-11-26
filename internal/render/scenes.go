package render

import "math"

type Scene struct {
	ID       string
	Title    string
	Subtitle string
	Note     string
	Source   func(x, y, w, h int) RGB
	ClipU    [3]float64
	HueWrapU float64
	HueScene bool
}

var Scenes = []Scene{
	{
		ID:       "shadow-steps",
		Title:    "SHADOW STEPS",
		Subtitle: "NEUTRAL 1-56",
		Note:     "TOP TICKS = HARD STEP EDGES",
		Source:   shadowSource,
		ClipU:    [3]float64{-1, -1, -1},
		HueWrapU: -1,
	},
	{
		ID:       "color-slope",
		Title:    "COLOR SLOPE",
		Subtitle: "HUE 300 TO 660",
		Note:     "SAME SOURCE HUE FOR EVERY DEPTH",
		Source:   colorSlopeSource,
		ClipU:    [3]float64{-1, -1, -1},
		HueWrapU: 1.0 / 6.0,
		HueScene: true,
	},
	{
		ID:       "highlight-clip",
		Title:    "HIGHLIGHT CLIP",
		Subtitle: "R G B CLIP ORDER",
		Note:     "MAGENTA HATCH = CLIPPED",
		Source:   highlightSource,
		ClipU:    [3]float64{0.58, 0.70, 0.90},
		HueWrapU: -1,
	},
}

func shadowSource(x, y, w, h int) RGB {
	u := float64(x) / float64(max(1, w-1))
	v := float64(max(0, h-1-y)) / float64(max(1, h-1))
	code := 1 + 48*u + 7*v
	v = code / 255
	return RGB{R: v, G: v, B: v}
}

func colorSlopeSource(x, y, w, h int) RGB {
	u := float64(x) / float64(max(1, w-1))
	v := 1 - float64(y)/float64(max(1, h-1))
	hue := 300 + 360*u
	val := 0.58 + 0.42*v
	return HSV(hue, 1, val)
}

func highlightSource(x, y, w, h int) RGB {
	u := float64(x) / float64(max(1, w-1))
	_ = y
	_ = h
	t := 0.25 + 0.90*u
	return RGB{
		R: t / 0.772,
		G: t / 0.880,
		B: t / 1.060,
	}
}

func sourceAt(sc Scene, x, y, w, h int) RGB {
	return sc.Source(x, y, w, h)
}

func clipColumn(sc Scene, channel int, w int) int {
	if channel < 0 || channel > 2 || sc.ClipU[channel] < 0 {
		return -1
	}
	return int(math.Round(sc.ClipU[channel] * float64(max(1, w-1))))
}

func hueWrapColumn(sc Scene, w int) int {
	if sc.HueWrapU < 0 {
		return -1
	}
	return int(math.Round(sc.HueWrapU * float64(max(1, w-1))))
}

func firstSaturated(sc Scene, p Pipeline, channel, w, h int) int {
	for x := 0; x < w; x++ {
		v := sourceAt(sc, x, h/2, w, h)
		code := 0
		switch channel {
		case 0:
			code = DisplayCode(v.R, p, x, h/2)
		case 1:
			code = DisplayCode(v.G, p, x, h/2)
		case 2:
			code = DisplayCode(v.B, p, x, h/2)
		}
		if code == 255 {
			return x
		}
	}
	return -1
}

func isClipped(v RGB) bool {
	return v.R > 1 || v.G > 1 || v.B > 1
}

func clippedMask(v RGB) [3]bool {
	return [3]bool{v.R > 1, v.G > 1, v.B > 1}
}
