package main

import "math"

// FrameParams captures the per-frame glass state.
type FrameParams struct {
	Index       int
	SlabThick   float64 // geometric thickness along z
	AbsorbScale float64 // multiplies the tint-derived absorption
	Tint        Vec3    // absorption color (sRGB-ish 0..1 components)
	Clear       bool    // zero absorption: fully transparent glass
}

// FrameParamsFor cycles thickness, tint hue and absorption so the sequence
// contains a fully transparent frame (scale 0) and a near-opaque frame.
func FrameParamsFor(i, n int) FrameParams {
	thick := 0.35 + 0.45*(1+math.Cos(2*math.Pi*float64(i)/float64(n)-math.Pi))/2
	hue := float64(i) / float64(n)
	tint := HSV(hue, 0.85, 0.95)
	// Cosine easing: frame 0 is perfectly clear; mid frames reach max density.
	// At peak scale the strongest absorbed channel transmits under 2% over
	// the full slab thickness, visually restoring an almost solid occluder.
	scale := 5.0 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n))) / 2
	return FrameParams{
		Index:       i,
		SlabThick:   thick,
		AbsorbScale: scale,
		Tint:        tint,
		Clear:       scale < 1e-6,
	}
}

// BuildScene rebuilds the static layout each frame with the animated glass.
func BuildScene(p FrameParams) *Scene {
	lightDir := Vec3{-0.45, 0.8, -0.4}.Normalize()
	scene := &Scene{LightDir: lightDir}

	// Colored opaque objects behind the glass, viewed through the slab.
	scene.Primitives = append(scene.Primitives,
		&Sphere{Center: Vec3{-0.9, 0.6, -3.4}, Radius: 0.6, Albedo: Vec3{0.85, 0.18, 0.15}},
		&Sphere{Center: Vec3{0.2, 0.45, -3.7}, Radius: 0.45, Albedo: Vec3{0.2, 0.75, 0.25}},
		&Sphere{Center: Vec3{1.15, 0.75, -3.3}, Radius: 0.75, Albedo: Vec3{0.2, 0.35, 0.9}},
		Ground{},
	)

	// Absorption coefficients: tint color sets the spectral shape, scale the
	// strength. A coefficient near zero in one channel means that channel
	// passes freely (classic colored glass look).
	abs := Vec3{}
	if !p.Clear {
		abs = Vec3{
			1.4 - p.Tint.X,
			1.4 - p.Tint.Y,
			1.4 - p.Tint.Z,
		}.Scale(p.AbsorbScale)
	}
	slabGlass := &Glass{ID: 1, IOR: 1.5, Absorption: abs}

	zc := -2.0
	slab := &Slab{
		Min: Vec3{-1.0, 0.4, zc - p.SlabThick/2},
		Max: Vec3{1.0, 2.2, zc + p.SlabThick/2},
		G:   slabGlass,
	}
	scene.Primitives = append(scene.Primitives, slab)

	// A separate glass sphere demonstrates curved refraction and shares the
	// same animated material state.
	scene.Primitives = append(scene.Primitives, &Sphere{
		Center: Vec3{-2.1, 0.75, 1.0},
		Radius: 0.7,
		Glass:  &Glass{ID: 2, IOR: 1.5, Absorption: abs},
	})

	return scene
}

// HSV converts a hue in [0,1), saturation/value in [0,1] to RGB in [0,1].
func HSV(h, s, v float64) Vec3 {
	h = math.Mod(h, 1)
	i := math.Floor(h * 6)
	f := h*6 - i
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch int(i) % 6 {
	case 0:
		return Vec3{v, t, p}
	case 1:
		return Vec3{q, v, p}
	case 2:
		return Vec3{p, v, t}
	case 3:
		return Vec3{p, q, v}
	case 4:
		return Vec3{t, p, v}
	default:
		return Vec3{v, p, q}
	}
}
