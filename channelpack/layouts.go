package main

// buildSeparate stores each source map in its own RGBA texture. Scalar maps
// are replicated into RGB; unused alpha slots are filled with 255. Every
// value survives the round trip byte-for-byte.
func buildSeparate(m *Maps) []*Texture {
	w, h := m.W, m.H
	albedo := newTexture(w, h, "texture 1")
	normal := newTexture(w, h, "texture 2")
	rough := newTexture(w, h, "texture 3")
	ao := newTexture(w, h, "texture 4")
	for i := 0; i < w*h; i++ {
		c := i * 3
		texSet(albedo.Pix, i, m.Albedo[c], m.Albedo[c+1], m.Albedo[c+2], 255)
		texSet(normal.Pix, i, m.Normal[c], m.Normal[c+1], m.Normal[c+2], 255)
		r := m.Roughness[i]
		texSet(rough.Pix, i, r, r, r, 255)
		a := m.Occlusion[i]
		texSet(ao.Pix, i, a, a, a, 255)
	}
	return []*Texture{albedo, normal, rough, ao}
}

func decodeSeparate(tex []*Texture, w, h int) *Maps {
	out := emptyMaps(w, h)
	for i := 0; i < w*h; i++ {
		c := i * 3
		out.Albedo[c] = texGet(tex[0].Pix, i, 0)
		out.Albedo[c+1] = texGet(tex[0].Pix, i, 1)
		out.Albedo[c+2] = texGet(tex[0].Pix, i, 2)
		out.Normal[c] = texGet(tex[1].Pix, i, 0)
		out.Normal[c+1] = texGet(tex[1].Pix, i, 1)
		out.Normal[c+2] = texGet(tex[1].Pix, i, 2)
		out.Roughness[i] = texGet(tex[2].Pix, i, 0)
		out.Occlusion[i] = texGet(tex[3].Pix, i, 0)
	}
	return out
}

// buildPacked merges the eight source channels into two RGBA textures without
// any quantization: texture 1 carries albedo RGB plus occlusion in alpha,
// texture 2 carries normal XYZ plus roughness in alpha.
func buildPacked(m *Maps) []*Texture {
	w, h := m.W, m.H
	t1 := newTexture(w, h, "texture 1")
	t2 := newTexture(w, h, "texture 2")
	for i := 0; i < w*h; i++ {
		c := i * 3
		texSet(t1.Pix, i, m.Albedo[c], m.Albedo[c+1], m.Albedo[c+2], m.Occlusion[i])
		texSet(t2.Pix, i, m.Normal[c], m.Normal[c+1], m.Normal[c+2], m.Roughness[i])
	}
	return []*Texture{t1, t2}
}

func decodePacked(tex []*Texture, w, h int) *Maps {
	out := emptyMaps(w, h)
	for i := 0; i < w*h; i++ {
		c := i * 3
		out.Albedo[c] = texGet(tex[0].Pix, i, 0)
		out.Albedo[c+1] = texGet(tex[0].Pix, i, 1)
		out.Albedo[c+2] = texGet(tex[0].Pix, i, 2)
		out.Occlusion[i] = texGet(tex[0].Pix, i, 3)
		out.Normal[c] = texGet(tex[1].Pix, i, 0)
		out.Normal[c+1] = texGet(tex[1].Pix, i, 1)
		out.Normal[c+2] = texGet(tex[1].Pix, i, 2)
		out.Roughness[i] = texGet(tex[1].Pix, i, 3)
	}
	return out
}

// buildCompact trades one channel slot for explicit scalar compression. The
// two scalar maps share alpha through 4-bit nibbles, while every albedo and
// normal channel remains a full, independent 8-bit byte. Quantization is
// therefore mathematically incapable of touching normals or albedo.
func buildCompact(m *Maps) []*Texture {
	w, h := m.W, m.H
	t1 := newTexture(w, h, "texture 1")
	t2 := newTexture(w, h, "texture 2")
	for i := 0; i < w*h; i++ {
		c := i * 3
		rough4 := m.Roughness[i] >> 4
		ao4 := m.Occlusion[i] >> 4
		shared := rough4<<4 | ao4
		texSet(t1.Pix, i, m.Albedo[c], m.Albedo[c+1], m.Albedo[c+2], byte(shared))
		texSet(t2.Pix, i, m.Normal[c], m.Normal[c+1], m.Normal[c+2], 255)
	}
	return []*Texture{t1, t2}
}

func decodeCompact(tex []*Texture, w, h int) *Maps {
	out := emptyMaps(w, h)
	for i := 0; i < w*h; i++ {
		c := i * 3
		out.Albedo[c] = texGet(tex[0].Pix, i, 0)
		out.Albedo[c+1] = texGet(tex[0].Pix, i, 1)
		out.Albedo[c+2] = texGet(tex[0].Pix, i, 2)
		shared := texGet(tex[0].Pix, i, 3)
		// Mid-scale dequantization limits each scalar error to <= 8 levels.
		out.Roughness[i] = (shared>>4)<<4 | 0x08
		out.Occlusion[i] = (shared&0x0F)<<4 | 0x08
		out.Normal[c] = texGet(tex[1].Pix, i, 0)
		out.Normal[c+1] = texGet(tex[1].Pix, i, 1)
		out.Normal[c+2] = texGet(tex[1].Pix, i, 2)
	}
	return out
}

// decodeMisaligned intentionally uses the wrong slot table against packed
// textures: the two alpha scalars are read as albedo/normal channels and two
// normal channels feed albedo. It reproduces the classic binding mistake that
// makes a material ball display impossible colors and broken lighting.
func decodeMisaligned(tex []*Texture, w, h int) *Maps {
	out := emptyMaps(w, h)
	for i := 0; i < w*h; i++ {
		c := i * 3
		// Wrong table: alpha scalars and color channels are bound to the wrong
		// slots, so occlusion/roughness leak into color and normal data.
		out.Albedo[c] = texGet(tex[1].Pix, i, 1)    // normal Y as red
		out.Albedo[c+1] = texGet(tex[0].Pix, i, 3)  // occlusion as green
		out.Albedo[c+2] = texGet(tex[1].Pix, i, 3)  // roughness as blue
		out.Normal[c] = texGet(tex[0].Pix, i, 2)    // albedo B as normal X
		out.Normal[c+1] = texGet(tex[1].Pix, i, 3)  // roughness as normal Y
		out.Normal[c+2] = texGet(tex[0].Pix, i, 3)  // occlusion as normal Z
		out.Roughness[i] = texGet(tex[1].Pix, i, 0) // normal X as roughness
		out.Occlusion[i] = texGet(tex[0].Pix, i, 1) // albedo G as occlusion
	}
	return out
}

func emptyMaps(w, h int) *Maps {
	n := w * h
	return &Maps{
		W:         w,
		H:         h,
		Albedo:    make([]byte, n*3),
		Normal:    make([]byte, n*3),
		Roughness: make([]byte, n),
		Occlusion: make([]byte, n),
	}
}

func allLayouts() []*Layout {
	return []*Layout{
		{Name: "separate", Encode: buildSeparate, Decode: decodeSeparate},
		{Name: "packed", Encode: buildPacked, Decode: decodePacked},
		{Name: "compact", Encode: buildCompact, Decode: decodeCompact},
	}
}
