package main

import "math"

// Mode identifies a metering pattern.
type Mode int

const (
	ModeFullFrame Mode = iota
	ModeCenterWeighted
	ModeHighlightProtect
	ModeSpot // moving spot point, used to prove temporal stability
)

var modeNames = map[Mode]string{
	ModeFullFrame:        "fullframe",
	ModeCenterWeighted:   "center",
	ModeHighlightProtect: "highlight",
	ModeSpot:             "spot",
}

func (m Mode) String() string { return modeNames[m] }

// Weights returns a normalized metering weight map for the frame.
// spotX/spotY are the moving spot point in pixel coords (spot mode only).
func Weights(f *Frame, mode Mode, spotX, spotY float64) []float64 {
	w := make([]float64, f.W*f.H)
	cx, cy := float64(f.W-1)/2, float64(f.H-1)/2
	switch mode {
	case ModeFullFrame:
		for i := range w {
			w[i] = 1
		}
	case ModeCenterWeighted:
		sigma := 0.28 * math.Min(float64(f.W), float64(f.H))
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				dx, dy := float64(x)-cx, float64(y)-cy
				w[y*f.W+x] = 0.15 + 0.85*math.Exp(-(dx*dx+dy*dy)/(2*sigma*sigma))
			}
		}
	case ModeHighlightProtect:
		// Weight rises with luminance: bright regions dominate the reading.
		for i := range w {
			l := f.Luminance(i)
			w[i] = 0.05 + 4.0*l*l
		}
	case ModeSpot:
		sigma := 0.045 * math.Min(float64(f.W), float64(f.H))
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				dx, dy := float64(x)-spotX, float64(y)-spotY
				w[y*f.W+x] = math.Exp(-(dx*dx + dy*dy) / (2 * sigma * sigma))
			}
		}
	}
	normalize(w)
	return w
}

func normalize(w []float64) {
	sum := 0.0
	for _, v := range w {
		sum += v
	}
	if sum <= 0 {
		return
	}
	for i := range w {
		w[i] /= sum
	}
}

// Meter returns the weighted mean linear luminance of the frame.
func Meter(f *Frame, w []float64) float64 {
	sum := 0.0
	for i, wi := range w {
		sum += wi * f.Luminance(i)
	}
	return sum
}

const (
	// MiddleGray is the 18% gray target luminance.
	MiddleGray = 0.18
	// HighlightTarget is where the protected highlight percentile lands.
	HighlightTarget = 0.85
	// HighlightPercentile is the weighted luminance percentile guarded
	// against clipping.
	HighlightPercentile = 0.90
	// MaxHighlightUnderEV caps how far highlight protection may push
	// exposure below the center-weighted reading, so the subject is
	// never crushed into a black blob.
	MaxHighlightUnderEV = 2.0
)

// EVForLuminance converts a metered luminance to the exposure
// compensation (in EV stops) needed to place it at middle gray.
func EVForLuminance(lum float64) float64 {
	if lum < 1e-6 {
		lum = 1e-6
	}
	return math.Log2(MiddleGray / lum)
}

// HighlightProtectEV exposes so the frame's weighted 90th-percentile
// luminance lands at HighlightTarget (just below clipping), then clamps
// the result so it never underexposes more than MaxHighlightUnderEV
// relative to the center-weighted EV (which tracks the subject).
func HighlightProtectEV(f *Frame) (ev float64, weights []float64) {
	hw := Weights(f, ModeHighlightProtect, 0, 0)
	cw := Weights(f, ModeCenterWeighted, 0, 0)
	p := WeightedPercentile(f, hw, HighlightPercentile)
	if p < 1e-6 {
		p = 1e-6
	}
	evHL := math.Log2(HighlightTarget / p)
	evCenter := EVForLuminance(Meter(f, cw))
	floor := evCenter - MaxHighlightUnderEV
	if evHL < floor {
		evHL = floor
	}
	return evHL, hw
}

// WeightedPercentile returns the smallest luminance L such that the
// total weight of pixels with luminance <= L reaches fraction q of the
// total weight. Weights must be normalized.
func WeightedPercentile(f *Frame, w []float64, q float64) float64 {
	const bins = 512
	hist := make([]float64, bins)
	maxL := 1e-6
	for i := range w {
		if l := f.Luminance(i); l > maxL {
			maxL = l
		}
	}
	for i, wi := range w {
		b := int(f.Luminance(i) / maxL * (bins - 1))
		hist[b] += wi
	}
	acc := 0.0
	for b := 0; b < bins; b++ {
		acc += hist[b]
		if acc >= q {
			return float64(b) / (bins - 1) * maxL
		}
	}
	return maxL
}

// Smoother applies an exponential moving average plus a per-frame slew
// limit so a moving metering point cannot make exposure jump per frame.
type Smoother struct {
	Alpha   float64 // EMA blend factor in (0,1]
	MaxStep float64 // max EV change per frame
	value   float64
	set     bool
}

func NewSmoother(alpha, maxStep float64) *Smoother {
	return &Smoother{Alpha: alpha, MaxStep: maxStep}
}

// Update feeds a raw EV reading and returns the stabilized EV.
func (s *Smoother) Update(raw float64) float64 {
	if !s.set {
		s.value, s.set = raw, true
		return s.value
	}
	target := s.value + s.Alpha*(raw-s.value)
	if d := target - s.value; d > s.MaxStep {
		target = s.value + s.MaxStep
	} else if d < -s.MaxStep {
		target = s.value - s.MaxStep
	}
	s.value = target
	return s.value
}
