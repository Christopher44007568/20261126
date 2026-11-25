package scene

import "math"

const (
	WorldWidth  = 1920
	WorldHeight = 540
	CropWidth   = 960
	FrameWidth  = 480
	FrameHeight = 270
	FPS         = 30
	Duration    = 10.0
	FrameCount  = FPS * int(Duration)
	PanStart    = 0.80
	PanDuration = 0.85
)

type RGB struct {
	R, G, B float32
}

type HDR struct {
	Width  int
	Height int
	Pix    []RGB
}

func NewHDR(width, height int) *HDR {
	return &HDR{Width: width, Height: height, Pix: make([]RGB, width*height)}
}

func (h *HDR) index(x, y int) int {
	return y*h.Width + x
}

func (h *HDR) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= h.Width || y >= h.Height {
		return
	}
	h.Pix[h.index(x, y)] = c
}

func (h *HDR) Add(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= h.Width || y >= h.Height {
		return
	}
	p := &h.Pix[h.index(x, y)]
	p.R += c.R
	p.G += c.G
	p.B += c.B
}

func (h *HDR) At(x, y int) RGB {
	return h.Pix[h.index(x, y)]
}

func Luminance(c RGB) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func LerpRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: float32(lerp(float64(a.R), float64(b.R), t)),
		G: float32(lerp(float64(a.G), float64(b.G), t)),
		B: float32(lerp(float64(a.B), float64(b.B), t)),
	}
}

func SmoothStep(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func PanX(timeSeconds float64) int {
	p := SmoothStep((timeSeconds - PanStart) / PanDuration)
	const endX = 760
	return int(math.Round(p * endX))
}

func CameraFrame(world *HDR, timeSeconds float64) *HDR {
	x0 := PanX(timeSeconds)
	out := NewHDR(FrameWidth, FrameHeight)
	for y := 0; y < FrameHeight; y++ {
		wy0 := y * 2
		for x := 0; x < FrameWidth; x++ {
			wx0 := x0 + x*2
			var sum RGB
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					c := world.At(wx0+dx, wy0+dy)
					sum.R += c.R
					sum.G += c.G
					sum.B += c.B
				}
			}
			out.Set(x, y, RGB{sum.R / 4, sum.G / 4, sum.B / 4})
		}
	}
	return out
}
