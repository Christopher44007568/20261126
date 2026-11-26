package main

// Maps holds the four source material maps. Every map is WxH and stores
// unsigned 8-bit channel values, matching an 8-bit RGBA texture pipeline.
type Maps struct {
	W, H      int
	Albedo    []byte // 3 bytes per pixel: R, G, B
	Normal    []byte // 3 bytes per pixel: tangent-space X, Y, Z
	Roughness []byte // 1 byte per pixel
	Occlusion []byte // 1 byte per pixel
}

// Texture is one packed or standalone RGBA texture.
type Texture struct {
	Name string
	Pix  []byte // 4 bytes per pixel: R, G, B, A
}

// Layout describes a channel-packing scheme and provides its encode/decode
// binding. The binding table is deliberately explicit so that a swapped
// decoder can demonstrate a slot-binding (swizzle) defect.
type Layout struct {
	Name     string
	Textures []*Texture
	Encode   func(m *Maps) []*Texture
	Decode   func(tex []*Texture, w, h int) *Maps
}

func (m *Maps) idx(x, y int) int { return (y*m.W + x) * 3 }

func (m *Maps) scalarAt(s []byte, x, y int) byte {
	return s[y*m.W+x]
}

func newTexture(w, h int, name string) *Texture {
	return &Texture{Name: name, Pix: make([]byte, w*h*4)}
}

func texSet(pix []byte, i int, r, g, b, a byte) {
	pix[i*4] = r
	pix[i*4+1] = g
	pix[i*4+2] = b
	pix[i*4+3] = a
}

func texGet(pix []byte, i, c int) byte { return pix[i*4+c] }

func clampByte(v float64) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}
