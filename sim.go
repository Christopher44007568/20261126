package main

import "math"

// Params collects every tunable quantity of the shutter simulation.
// The scene is a single vertical rod rendered through an orthographic
// camera that performs one smooth horizontal pan and comes to rest again.
type Params struct {
	Width  int // sensor width in pixels
	Height int // sensor height in pixels (number of readout rows)
	Frames int // number of consecutive frames produced per shutter path
	FPS    float64

	RodX        float64 // world x of the rigid rod, in pixel units
	RodHalf     float64 // rod half width in pixels
	Exposure    float64 // per-row / per-segment exposure duration in seconds
	ReadoutT    float64 // rolling-shutter time between row 0 and row H-1
	BlockRows   int     // rows per exposure segment in the segmented path
	FeatherRows float64 // half width of the linear cross-fade at block borders
	Samples     int     // temporal samples integrated during one exposure

	PanStart float64 // simulation time at which the pan begins
	PanRamp  float64 // duration of each acceleration / deceleration ramp
	PanFlat  float64 // duration of the constant-velocity plateau
	PanVMax  float64 // peak pan velocity in pixels per second
}

func defaultParams() Params {
	return Params{
		Width: 320, Height: 240, Frames: 20, FPS: 30,
		RodX: 160, RodHalf: 6,
		Exposure:    0.020,
		ReadoutT:    0.030,
		BlockRows:   16,
		FeatherRows: 4,
		Samples:     5,
		PanStart:    0.20, PanRamp: 0.15, PanFlat: 0.10, PanVMax: 1350,
	}
}

func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// panVel is the camera's horizontal velocity at simulation time t.
// The profile is ramp up -> constant plateau -> ramp down to zero, so the
// camera is stationary both before PanStart and after the pan completes.
func (p Params) panVel(t float64) float64 {
	s := t - p.PanStart
	if s < 0 || s > 2*p.PanRamp+p.PanFlat {
		return 0
	}
	switch {
	case s < p.PanRamp:
		return p.PanVMax * smoothstep(s/p.PanRamp)
	case s < p.PanRamp+p.PanFlat:
		return p.PanVMax
	default:
		u := (s - p.PanRamp - p.PanFlat) / p.PanRamp
		return p.PanVMax * (1 - smoothstep(u))
	}
}

// panPos is the analytic integral of panVel, giving the camera's
// horizontal offset (pixels) at time t without numerical drift.
func (p Params) panPos(t float64) float64 {
	s := t - p.PanStart
	if s <= 0 {
		return 0
	}
	ramp := func(u float64) float64 {
		// integral of smoothstep on [0,u]
		return u*u*u - 0.5*u*u*u*u
	}
	t1 := p.PanRamp
	t2 := p.PanRamp + p.PanFlat
	t3 := p.PanRamp + p.PanFlat + p.PanRamp
	switch {
	case s <= t1:
		return p.PanVMax * p.PanRamp * ramp(s/p.PanRamp)
	case s <= t2:
		return p.PanVMax * (p.PanRamp*0.5 + (s - t1))
	case s <= t3:
		u := (s - t2) / p.PanRamp
		plateau := p.PanRamp*0.5 + p.PanFlat
		if u >= 1 {
			return p.PanVMax * (p.PanRamp + p.PanFlat)
		}
		return p.PanVMax * (plateau + p.PanRamp*(u-ramp(u)))
	default:
		// The deceleration ramp integrates to exactly one ramp-length, so
		// the resting camera position is a true constant with no float drift.
		return p.PanVMax * (p.PanRamp + p.PanFlat)
	}
}

// frameStart returns the simulation time at which frame f begins its readout.
func (p Params) frameStart(f int) float64 {
	return float64(f) / p.FPS
}

// rowCenter is the mid-exposure sample time of a sensor row.
func (p Params) rowCenter(frameStart, rowFrac float64) float64 {
	return frameStart + p.Exposure/2 + rowFrac*p.ReadoutT
}

// segmentCenter is the mid-exposure sample time of row y under segmented
// exposure. All rows of one block share one exposure window; block borders
// are linearly cross-faded over FeatherRows so the timing map is continuous
// and no row can jump its sample instant relative to its neighbours.
func (p Params) segmentCenter(frameStart float64, y int) float64 {
	b := float64(p.BlockRows)
	h := float64(p.Height - 1)
	block := float64(y) / b
	// Rows inside a block share one exposure instant; only a feather zone of
	// half width FeatherRows around each block border cross-fades, which keeps
	// the timing map C0-continuous and forbids any border row jump.
	local := math.Mod(float64(y), b)
	half, fz := b/2, p.FeatherRows
	var w float64
	switch {
	case local <= half-fz:
		w = 0
	case local >= half+fz:
		w = 1
	default:
		w = (local - (half - fz)) / (2 * fz)
	}
	blend := b * w
	eff := math.Floor(block)*b + blend
	frac := eff / h
	return p.rowCenter(frameStart, frac)
}

// globalCenter is the single sample instant shared by every row.
func (p Params) globalCenter(frameStart float64) float64 {
	return p.rowCenter(frameStart, 0.5)
}
