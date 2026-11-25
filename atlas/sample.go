package atlas

import (
	"image"
	"image/color"
	"math"
)

// SampleBilinear performs clamp-to-edge bilinear sampling using normalized
// top-left-origin UVs. Integer texel centers are at (i+0.5)/size.
func SampleBilinear(img *image.RGBA, u, v float64) color.RGBA {
	b := img.Bounds()
	return SampleBilinearSpan(img, u, v, float64(b.Dx()), float64(b.Dy()))
}

// SampleBilinearSpan samples with an explicit texel span for U/V. This lets a
// mip level with ceil dimensions be evaluated using the fractional footprint
// W/2^l implied by the original atlas UV rectangle.
func SampleBilinearSpan(img *image.RGBA, u, v, spanW, spanH float64) color.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return color.RGBA{}
	}
	if spanW <= 0 {
		spanW = float64(w)
	}
	if spanH <= 0 {
		spanH = float64(h)
	}
	u = clampFloat(u, 0, 1)
	v = clampFloat(v, 0, 1)

	x := u*spanW - 0.5
	y := v*spanH - 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	x1 := x0 + 1
	y1 := y0 + 1

	c00 := rgbaAtClamped(img, x0, y0)
	c10 := rgbaAtClamped(img, x1, y0)
	c01 := rgbaAtClamped(img, x0, y1)
	c11 := rgbaAtClamped(img, x1, y1)

	top := lerpPremul(c00, c10, fx)
	bottom := lerpPremul(c01, c11, fx)
	return lerpPremul(top, bottom, fy)
}

type premulColor struct {
	r, g, b, a float64
}

func rgbaAtClamped(img *image.RGBA, x, y int) color.RGBA {
	b := img.Bounds()
	x = clampInt(x-b.Min.X, 0, b.Dx()-1) + b.Min.X
	y = clampInt(y-b.Min.Y, 0, b.Dy()-1) + b.Min.Y
	return img.RGBAAt(x, y)
}

func toPremul(c color.RGBA) premulColor {
	a := float64(c.A) / 255.0
	return premulColor{r: float64(c.R) * a, g: float64(c.G) * a, b: float64(c.B) * a, a: a}
}

func fromPremul(c premulColor) color.RGBA {
	a := clampFloat(c.a, 0, 1)
	if a <= 0.000001 {
		return color.RGBA{A: 0}
	}
	return color.RGBA{
		R: uint8(clampFloat(c.r/a+0.5, 0, 255)),
		G: uint8(clampFloat(c.g/a+0.5, 0, 255)),
		B: uint8(clampFloat(c.b/a+0.5, 0, 255)),
		A: uint8(clampFloat(a*255.0+0.5, 0, 255)),
	}
}

func lerpPremul(a, b color.RGBA, t float64) color.RGBA {
	ca, cb := toPremul(a), toPremul(b)
	return fromPremul(premulColor{
		r: ca.r + (cb.r-ca.r)*t,
		g: ca.g + (cb.g-ca.g)*t,
		b: ca.b + (cb.b-ca.b)*t,
		a: ca.a + (cb.a-ca.a)*t,
	})
}

func clampFloat(v, lo, hi float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// AtlasToSpriteUV maps an atlas UV into a sprite-local UV, undoing packing
// rotation. V uses top-to-bottom image orientation throughout.
func AtlasToSpriteUV(p Placement, atlasW, atlasH int, u, v float64) (float64, float64, bool) {
	u0, v0, u1, v1 := p.UVBounds(atlasW, atlasH)
	if u < u0 || u > u1 || v < v0 || v > v1 {
		return 0, 0, false
	}
	fu := (u - u0) / (u1 - u0)
	fv := (v - v0) / (v1 - v0)
	if !p.Rotated {
		return fu, fv, true
	}
	// Inverse of clockwise rotation: source U follows destination V, and
	// source V is the reverse of destination U.
	return fv, 1.0 - fu, true
}

// SpriteToAtlasUV maps sprite-local UV to atlas UV, applying clockwise
// rotation when needed.
func SpriteToAtlasUV(p Placement, atlasW, atlasH int, lu, lv float64) (float64, float64) {
	u0, v0, u1, v1 := p.UVBounds(atlasW, atlasH)
	if !p.Rotated {
		return u0 + lu*(u1-u0), v0 + lv*(v1-v0)
	}
	fu := 1.0 - lv
	fv := lu
	return u0 + fu*(u1-u0), v0 + fv*(v1-v0)
}

func colorDistance(a, b color.RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	da := float64(a.A) - float64(b.A)
	return math.Sqrt(dr*dr + dg*dg + db*db + 0.25*da*da)
}
