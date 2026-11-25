package internal

// multiply does component-wise color multiplication.
func multiply(a, b RGBA) RGBA {
	return RGBA{a.R * b.R, a.G * b.G, a.B * b.B, a.A}
}

// schlick is the Schlick Fresnel approximation for a conductor-ish surface.
func schlick(cosTheta float64) float64 {
	const f0 = 0.04
	return f0 + (1-f0)*(1-cosTheta)*(1-cosTheta)*(1-cosTheta)*(1-cosTheta)*(1-cosTheta)
}

// shadeGround adds simple directional + hemispheric lighting to the albedo.
func (r *Renderer) shadeGround(albedo RGBA, h hitInfo) RGBA {
	sun := Clamp(h.n.Dot(r.Scene.SunDir), 0, 1)
	hemi := 0.25 + 0.25*h.n.Y
	k := hemi + 0.7*sun
	return RGBA{
		R: Clamp(albedo.R*k, 0, 1),
		G: Clamp(albedo.G*k, 0, 1),
		B: Clamp(albedo.B*k, 0, 1),
		A: 1,
	}
}

// lodPalette assigns a distinct hue to each integer mip level. Fractional LOD
// is blended between adjacent level colors, so continuous LOD switching shows
// up as smooth color transitions rather than hard bands.
var lodPalette = []RGBA{
	{0.10, 0.10, 0.40, 1}, // 0 indigo
	{0.15, 0.45, 0.85, 1}, // 1 blue
	{0.10, 0.70, 0.75, 1}, // 2 teal
	{0.20, 0.75, 0.35, 1}, // 3 green
	{0.65, 0.80, 0.25, 1}, // 4 lime
	{0.95, 0.80, 0.20, 1}, // 5 yellow
	{0.95, 0.55, 0.15, 1}, // 6 orange
	{0.90, 0.30, 0.20, 1}, // 7 red
	{0.70, 0.25, 0.55, 1}, // 8 magenta
	{0.55, 0.35, 0.75, 1}, // 9 purple
}

// LODColor maps a (fractional) LOD to a palette color, cycling if the texture
// has more levels than the palette defines.
func LODColor(lod, maxLOD float64) RGBA {
	lo := int(lod)
	frac := lod - float64(lo)
	pick := func(i int) RGBA {
		return lodPalette[i%len(lodPalette)]
	}
	c0 := pick(lo)
	if lo >= int(maxLOD) {
		return c0
	}
	c1 := pick(lo + 1)
	return RGBA{
		Lerp(c0.R, c1.R, frac),
		Lerp(c0.G, c1.G, frac),
		Lerp(c0.B, c1.B, frac),
		1,
	}
}
