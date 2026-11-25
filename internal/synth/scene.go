package synth

import (
	"math"
)

// Scene is a procedurally textured, bounded world plane (z fixed by Camera).
// Strong vertical/horizontal lines make rolling-shutter shear easy to see.
type Scene struct {
	MinX, MaxX float64
	MinY, MaxY float64
}

func New() *Scene {
	return &Scene{MinX: -1400, MaxX: 1400, MinY: -1000, MaxY: 1000}
}

func (s *Scene) Inside(x, y float64) bool {
	return x >= s.MinX && x <= s.MaxX && y >= s.MinY && y <= s.MaxY
}

// Sample returns linear RGB in [0,1] at a world position.
func (s *Scene) Sample(x, y float64) (r, g, b float64) {
	// Soft base gradient so large-scale flow is visible.
	base := 0.78 + 0.10*math.Sin(x*0.0018) + 0.06*math.Cos(y*0.0022)
	r, g, b = base, base, base*0.97

	// Fine checker patches give high-frequency texture for the sampler.
	cx := math.Floor(x/22) + math.Floor(y/22)
	if int(cx)&1 == 0 && math.Mod(math.Abs(x), 88) < 14 && math.Mod(math.Abs(y), 88) < 14 {
		r, g, b = 0.32, 0.34, 0.40
	}

	// Grid every 60 world units: regular vertical/horizontal reference lines.
	if dist := math.Abs(mod1(x, 60)); dist < 2.2 {
		r, g, b = 0.12, 0.14, 0.18
	}
	if dist := math.Abs(mod1(y, 60)); dist < 2.2 {
		r, g, b = 0.16, 0.18, 0.22
	}
	// Heavy accent lines every 300 units emphasize jelly deformation.
	if dist := math.Abs(mod1(x, 300)); dist < 4.5 {
		r, g, b = 0.55, 0.12, 0.10
	}
	if dist := math.Abs(mod1(y, 300)); dist < 4.5 {
		r, g, b = 0.10, 0.25, 0.55
	}

	// A few colored blocks add non-repetitive landmarks.
	for _, blk := range blocks {
		if x >= blk.x0 && x <= blk.x1 && y >= blk.y0 && y <= blk.y1 {
			r, g, b = blk.r, blk.g, blk.b
		}
	}
	return r, g, b
}

type block struct {
	x0, y0, x1, y1 float64
	r, g, b        float64
}

var blocks = []block{
	{-520, -360, -300, -180, 0.85, 0.55, 0.20},
	{180, 120, 420, 320, 0.20, 0.60, 0.45},
	{-120, 260, 60, 420, 0.45, 0.30, 0.70},
	{620, -420, 820, -260, 0.90, 0.75, 0.25},
	{-820, 420, -640, 560, 0.25, 0.45, 0.80},
}

// mod1 is a symmetric modulo centered at zero (distance to nearest grid line).
func mod1(v, step float64) float64 {
	m := math.Mod(v+step/2, step)
	if m < 0 {
		m += step
	}
	return m - step/2
}
