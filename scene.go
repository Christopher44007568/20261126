package main

import "math"

// Scene dimensions (each display panel renders this same buffer).
const (
	SceneW = 460
	SceneH = 340
)

// Scene holds one shared, exposure-independent scene in absolute CIE XYZ
// (Y in cd/m^2). It is generated once and reused by every simulated display,
// so switching displays never rebuilds or re-exposes the scene.
type Scene struct {
	W, H     int
	xyz      []Vec3
	Exposure float64 // exposure value in EV stops, shared across displays
}

// xyY builds an absolute-luminance XYZ color from chromaticity and Y (nits).
func xyY(p Chromaticity, yNits float64) Vec3 {
	return Vec3{
		R: p.X * yNits / p.Y,
		G: yNits,
		B: (1 - p.X - p.Y) * yNits / p.Y,
	}
}

func smoothEdge(dist, radius, soft float64) float64 {
	t := clamp01((radius - dist) / soft)
	return t * t * (3 - 2*t)
}

func (s *Scene) blend(x, y int, c Vec3, a float64) {
	idx := y*s.W + x
	bg := s.xyz[idx]
	s.xyz[idx] = Vec3{
		R: bg.R*(1-a) + c.R*a,
		G: bg.G*(1-a) + c.G*a,
		B: bg.B*(1-a) + c.B*a,
	}
}

func (s *Scene) fillRect(x0, y0, x1, y1 int, c Vec3) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && x < s.W && y >= 0 && y < s.H {
				s.blend(x, y, c, 1)
			}
		}
	}
}

func (s *Scene) disc(cx, cy, radius, soft float64, c Vec3) {
	r0 := int(cx - radius - soft)
	r1 := int(cx + radius + soft)
	for y := r0; y <= r1; y++ {
		for x := r0; x <= r1; x++ {
			if x < 0 || x >= s.W || y < 0 || y >= s.H {
				continue
			}
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if a := smoothEdge(d, radius, soft); a > 0 {
				s.blend(x, y, c, a)
			}
		}
	}
}

// NewScene renders the shared master scene. All values are physical
// luminances; display-specific tone mapping happens later.
func NewScene() *Scene {
	s := &Scene{W: SceneW, H: SceneH, xyz: make([]Vec3, SceneW*SceneH)}

	skyTop := xyY(Chromaticity{0.27, 0.30}, 12)
	skyHor := xyY(Chromaticity{0.32, 0.34}, 42)
	groundFar := xyY(Chromaticity{0.36, 0.37}, 26)
	groundNear := xyY(Chromaticity{0.34, 0.35}, 7)
	horizon := int(math.Floor(0.52 * float64(SceneH)))

	for y := 0; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			var c Vec3
			switch {
			case y < horizon:
				t := float64(y) / float64(horizon)
				c = lerp(skyTop, skyHor, t)
			default:
				t := float64(y-horizon) / float64(SceneH-horizon)
				c = lerp(groundFar, groundNear, t)
			}
			s.xyz[y*SceneW+x] = c
		}
	}

	// Bright warm sun: 4000 cd/m^2, wide enough to exceed every panel peak.
	s.disc(float64(SceneW)*0.74, float64(SceneH)*0.26, 36, 8,
		xyY(Chromaticity{0.44, 0.41}, 4000))

	// Three saturated self-luminous orbs (600 nits) stress gamut mapping.
	orbs := []struct {
		cx float64
		wp Chromaticity
	}{
		{0.16, Chromaticity{0.66, 0.32}},
		{0.30, Chromaticity{0.24, 0.68}},
		{0.44, Chromaticity{0.15, 0.08}},
	}
	for _, o := range orbs {
		s.disc(float64(SceneW)*o.cx, float64(SceneH)*0.60, 26, 5, xyY(o.wp, 600))
	}

	// Six-patch color checker row at ordinary reflectance levels.
	checker := []Chromaticity{
		{0.60, 0.33}, {0.37, 0.58}, {0.16, 0.17},
		{0.50, 0.44}, {0.30, 0.30}, {0.45, 0.20},
	}
	pw, ph := 48, 36
	x0 := fint(float64(SceneW) * 0.58)
	y0 := fint(float64(SceneH) * 0.62)
	for i, wp := range checker {
		s.fillRect(x0+i*(pw+6), y0, x0+i*(pw+6)+pw, y0+ph, xyY(wp, 55))
	}

	// Neutral D65 luminance ramp 0..300 nits: clips first on the dim display.
	rampY0, rampY1 := fint(float64(SceneH)*0.80), fint(float64(SceneH)*0.86)
	rampX0, rampX1 := fint(float64(SceneW)*0.08), fint(float64(SceneW)*0.52)
	for y := rampY0; y < rampY1; y++ {
		for x := rampX0; x < rampX1; x++ {
			t := float64(x-rampX0) / float64(rampX1-rampX0)
			s.blend(x, y, xyY(WP_D65, 300*t), 1)
		}
	}
	return s
}

func lerp(a, b Vec3, t float64) Vec3 {
	return Vec3{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
	}
}

// fint converts a (possibly constant) fractional layout value to pixels.
func fint(v float64) int { return int(math.Floor(v)) }

// Exposed returns the scene XYZ buffer with the shared exposure applied.
// The same buffer is handed to every display; displays never own exposure.
func (s *Scene) Exposed() []Vec3 {
	gain := math.Pow(2, s.Exposure)
	out := make([]Vec3, len(s.xyz))
	for i, c := range s.xyz {
		out[i] = Vec3{c.R * gain, c.G * gain, c.B * gain}
	}
	return out
}
