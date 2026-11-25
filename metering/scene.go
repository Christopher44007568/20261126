package main

import (
	"math"
	"math/rand"
)

// Frame is a linear-light RGB image in [0,1].
type Frame struct {
	W, H int
	R    []float64
	G    []float64
	B    []float64
}

func NewFrame(w, h int) *Frame {
	n := w * h
	return &Frame{W: w, H: h, R: make([]float64, n), G: make([]float64, n), B: make([]float64, n)}
}

// Luminance returns Rec.709 linear luminance at pixel i.
func (f *Frame) Luminance(i int) float64 {
	return 0.2126*f.R[i] + 0.7152*f.G[i] + 0.0722*f.B[i]
}

func (f *Frame) set(i int, r, g, b float64) {
	f.R[i], f.G[i], f.B[i] = r, g, b
}

// SceneParams describes the synthetic studio scene:
// a dark room, a blown-out window, and a mid-tone subject
// that walks across the frame over time.
type SceneParams struct {
	W, H      int
	WindowX0  float64 // window rect, fraction of width
	WindowY0  float64
	WindowX1  float64
	WindowY1  float64
	SubjectR  float64 // subject radius, fraction of height
	SubjectY  float64 // subject center height, fraction of height
	WindowLux float64 // window linear luminance
	RoomLux   float64 // darkest room luminance
	Noise     float64
	Seed      int64
}

func DefaultScene() SceneParams {
	return SceneParams{
		W: 480, H: 360,
		WindowX0: 0.68, WindowY0: 0.08, WindowX1: 0.95, WindowY1: 0.45,
		SubjectR: 0.16, SubjectY: 0.55,
		WindowLux: 3.00, RoomLux: 0.045, // HDR: window is ~6 stops over the room
		Noise: 0.006, Seed: 7,
	}
}

// SubjectCenterX returns the subject's horizontal center (fraction of
// width) at time t in [0,1]: it walks left to right, crossing the window.
func (p SceneParams) SubjectCenterX(t float64) float64 {
	return 0.12 + 0.72*t
}

// SpotPoint returns the moving spot-meter point (fractions of W/H).
// It sweeps a Lissajous path so it crosses window, subject and dark wall.
func (p SceneParams) SpotPoint(t float64) (float64, float64) {
	x := 0.5 + 0.38*math.Sin(2*math.Pi*1.5*t)
	y := 0.45 + 0.30*math.Sin(2*math.Pi*1.0*t+1.3)
	return x, y
}

// Render builds the linear-light frame at time t.
func (p SceneParams) Render(t float64) *Frame {
	f := NewFrame(p.W, p.H)
	rng := rand.New(rand.NewSource(p.Seed))
	noise := make([]float64, p.W*p.H)
	for i := range noise {
		noise[i] = rng.NormFloat64() * p.Noise
	}

	sx := p.SubjectCenterX(t) * float64(p.W)
	sy := p.SubjectY * float64(p.H)
	sr := p.SubjectR * float64(p.H)

	for y := 0; y < p.H; y++ {
		v := float64(y) / float64(p.H-1)
		for x := 0; x < p.W; x++ {
			u := float64(x) / float64(p.W-1)
			i := y*p.W + x

			// Dark room: gentle vertical gradient, slightly warm.
			base := p.RoomLux * (1.25 - 0.5*v)
			r, g, b := base*1.05, base*0.98, base*0.90

			// Bright window with soft edges and a brighter sky core.
			wx0, wy0 := p.WindowX0, p.WindowY0
			wx1, wy1 := p.WindowX1, p.WindowY1
			if u > wx0 && u < wx1 && v > wy0 && v < wy1 {
				edge := math.Min(math.Min(u-wx0, wx1-u)/(wx1-wx0),
					math.Min(v-wy0, wy1-v)/(wy1-wy0))
				glow := 0.75 + 0.25*smoothstep(0.0, 0.25, edge)
				w := p.WindowLux * glow
				r, g, b = w*0.96, w*0.99, w*1.04 // cool daylight
			}

			// Subject: mid-tone ellipse with Lambert-ish shading.
			dx := (float64(x) - sx) / sr
			dy := (float64(y) - sy) / (sr * 1.25)
			d := dx*dx + dy*dy
			if d < 1.0 {
				shade := 0.32 + 0.10*math.Sqrt(math.Max(0, 1.0-d))
				// Rim light from the window side when nearby.
				edge := smoothstep(0.6, 1.0, d)
				r = shade * (1.0 + 0.25*edge)
				g = shade * 0.92
				b = shade * 0.80
			}

			n := noise[i]
			// Keep HDR values (>1): the scene is linear light, clipping
			// happens later in the exposure/tone-mapping stage.
			f.set(i, max0(r+n), max0(g+n), max0(b+n))
		}
	}
	return f
}

func smoothstep(a, b, x float64) float64 {
	t := clamp01((x - a) / (b - a))
	return t * t * (3 - 2*t)
}

func max0(x float64) float64 {
	if x < 0 {
		return 0
	}
	return x
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
