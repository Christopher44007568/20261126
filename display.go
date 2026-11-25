package main

import "math"

type Transfer int

const (
	TransferSRGB Transfer = iota
	TransferPQ
)

// Display is a complete description of one target monitor: its gamut,
// white point, peak luminance and electro-optical transfer function.
type Display struct {
	Name      string
	Sub       string
	Primaries Primaries
	White     Chromaticity
	PeakNits  float64
	Transfer  Transfer

	xyzToRGB Mat3 // display-relative XYZ -> display linear RGB
	rgbToXYZ Mat3 // display linear RGB -> display-relative XYZ
	toD65    Mat3 // display white -> sRGB D65 cone adaptation
}

func NewDisplay(name, sub string, p Primaries, w Chromaticity,
	peak float64, t Transfer) *Display {
	rgbToXYZ := rgbToXYZ(p, w)
	return &Display{
		Name:      name,
		Sub:       sub,
		Primaries: p,
		White:     w,
		PeakNits:  peak,
		Transfer:  t,
		xyzToRGB:  rgbToXYZ.inverse(),
		rgbToXYZ:  rgbToXYZ,
		toD65:     adaptation(w, WP_D65),
	}
}

// DefaultDisplays returns the three simulated monitors. They differ in all
// three requested dimensions: brightness, white point and color gamut.
func DefaultDisplays() []*Display {
	return []*Display{
		NewDisplay("SDR WIDE GAMUT", "P3 / D65 / 200 nits / sRGB EOTF",
			P3Primaries, WP_D65, 200, TransferSRGB),
		NewDisplay("PQ HDR", "Rec.2020 / D93 / 1000 nits / SMPTE ST2084",
			Rec2020Prim, WP_D93, 1000, TransferPQ),
		NewDisplay("LOW BRIGHTNESS", "sRGB / 5000K / 80 nits / sRGB EOTF",
			sRGBPrimaries, WP_5K, 80, TransferSRGB),
	}
}

// Rendered is one display's output plus its clipping diagnostics.
type Rendered struct {
	Preview []Vec3 // soft-proofed sRGB-encoded RGB in [0,1]
	Mask    []bool // true where the signal had to be clipped
	ClipPct float64
	MinRGB  Vec3 // worst out-of-gamut channel values (diagnostics)
	MaxRGB  Vec3 // worst over-range values before tone mapping
}

// Render maps a shared, already-exposed XYZ scene onto this display.
// It must not mutate the scene buffer; the same buffer is passed to every
// display, which is how display switches preserve scene exposure.
func (d *Display) Render(exposedXYZ []Vec3, w, h int) *Rendered {
	n := w * h
	out := &Rendered{
		Preview: make([]Vec3, n),
		Mask:    make([]bool, n),
		MinRGB:  Vec3{math.Inf(1), math.Inf(1), math.Inf(1)},
	}
	clipped := 0
	const eps = 1e-4

	for i, xyzAbs := range exposedXYZ {
		// 1. Chromatically adapt the source white to this display's white.
		xyz := d.toD65.inverse().apply(xyzAbs)

		// 2. Convert into display-relative linear RGB and normalize so that
		//    1.0 equals the display peak luminance.
		lin := d.xyzToRGB.apply(xyz)
		lin = Vec3{lin.R / d.PeakNits, lin.G / d.PeakNits, lin.B / d.PeakNits}

		// 3. Gamut check: negative channels are outside this display gamut.
		gamutClip := lin.R < -eps || lin.G < -eps || lin.B < -eps
		out.MinRGB = Vec3{
			math.Min(out.MinRGB.R, lin.R),
			math.Min(out.MinRGB.G, lin.G),
			math.Min(out.MinRGB.B, lin.B),
		}
		out.MaxRGB = Vec3{
			math.Max(out.MaxRGB.R, lin.R),
			math.Max(out.MaxRGB.G, lin.G),
			math.Max(out.MaxRGB.B, lin.B),
		}
		for k := 0; k < 3; k++ {
			v := channel(&lin, k)
			if v < 0 {
				setChannel(&lin, k, 0)
			}
		}

		// 4. Extended Reinhard tone mapping on luminance ratio; a single
		//    factor preserves color relationships while rolling off highlights.
		white := 1.0 // display peak is the tone-map reference white
		lum := d.luminanceRatio(lin)
		factor := 1.0
		if lum > 0 {
			factor = (1 + lum/(white*white)) / (1 + lum)
		}
		mapped := Vec3{lin.R * factor, lin.G * factor, lin.B * factor}

		// 5. Anything still above the code value ceiling is a hard clip.
		overRange := mapped.R > 1+eps || mapped.G > 1+eps || mapped.B > 1+eps
		sig := Vec3{
			clamp01(mapped.R), clamp01(mapped.G), clamp01(mapped.B),
		}

		// 6. Apply this display's native transfer function.
		if d.Transfer == TransferPQ {
			// PQ signal 1.0 == 10000 cd/m^2; rescale from display-relative.
			pqScale := d.PeakNits / 10000
			sig = Vec3{pqEncode(sig.R * pqScale),
				pqEncode(sig.G * pqScale), pqEncode(sig.B * pqScale)}
		} else {
			sig = Vec3{sRGBEncode(sig.R), sRGBEncode(sig.G), sRGBEncode(sig.B)}
		}

		// 7. Soft proof: decode back through the display model, then encode
		//    for the sRGB/D65 canvas so all three panels are viewable together.
		preview := d.softProof(sig)
		out.Preview[i] = preview

		if gamutClip || overRange {
			out.Mask[i] = true
			clipped++
		}
	}

	out.ClipPct = 100 * float64(clipped) / float64(n)
	return out
}

// luminanceRatio weights linear RGB channels via the display's own XYZ row.
func (d *Display) luminanceRatio(lin Vec3) float64 {
	m := d.rgbToXYZ
	return m[1][0]*lin.R + m[1][1]*lin.G + m[1][2]*lin.B
}

// softProof turns an encoded display signal into an sRGB-encoded preview.
func (d *Display) softProof(signal Vec3) Vec3 {
	dec := sRGBDecode
	if d.Transfer == TransferPQ {
		dec = pqDecode
	}
	lin := Vec3{dec(signal.R), dec(signal.G), dec(signal.B)}

	// Rebuild absolute CIE XYZ: PQ signal 1.0 == 10000 nits, sRGB == peak.
	scale := d.PeakNits
	if d.Transfer == TransferPQ {
		scale = 10000
	}
	xyzAbs := d.rgbToXYZ.apply(lin)
	xyzAbs = Vec3{xyzAbs.R * scale, xyzAbs.G * scale, xyzAbs.B * scale}

	xyzD65 := d.toD65.apply(xyzAbs)
	srgbLin := sRGBXYZToRGB.apply(xyzD65)
	srgbLin = Vec3{
		previewNormalize(srgbLin.R),
		previewNormalize(srgbLin.G),
		previewNormalize(srgbLin.B),
	}
	return Vec3{
		sRGBEncode(clamp01(srgbLin.R)),
		sRGBEncode(clamp01(srgbLin.G)),
		sRGBEncode(clamp01(srgbLin.B)),
	}
}

func channel(v *Vec3, i int) float64 {
	switch i {
	case 0:
		return v.R
	case 1:
		return v.G
	default:
		return v.B
	}
}

func setChannel(v *Vec3, i int, val float64) {
	switch i {
	case 0:
		v.R = val
	case 1:
		v.G = val
	default:
		v.B = val
	}
}
