package dds

// Image is a decoded, tightly packed RGBA8 image. RGBA always means the
// memory channel order R,G,B,A, regardless of the source DDS channel masks.
type Image struct {
	Width  int
	Height int
	RGBA   []byte
}

// Face contains a complete mip chain. Faces are ordered +X, -X, +Y, -Y,
// +Z, -Z for cubemaps; ordinary textures have exactly one face.
type Face struct {
	Mips []Image
}

// Texture is only returned after the complete byte range has been validated.
// No partially decoded scanline can therefore leak into a preview.
type Texture struct {
	Width       int
	Height      int
	MipCount    int
	FaceCount   int
	IsCubemap   bool
	Format      string
	PixelFormat PixelFormat
	Faces       []Face
	DataOffset  int64
	DataSize    int64
	Trailing    int64
}

func (t *Texture) Base(face int) Image {
	return t.Faces[face].Mips[0]
}

// PixelFormat describes the on-disk samples. Masks are exposed so diagnostics
// can point at the exact channel definitions used for decoding.
type PixelFormat struct {
	Name       string `json:"name"`
	FourCC     string `json:"fourCC,omitempty"`
	BitCount   int    `json:"bitCount"`
	RMask      uint32 `json:"rMask,omitempty"`
	GMask      uint32 `json:"gMask,omitempty"`
	BMask      uint32 `json:"bMask,omitempty"`
	AMask      uint32 `json:"aMask,omitempty"`
	DXGI       uint32 `json:"dxgiFormat,omitempty"`
	Compressed bool   `json:"compressed"`
}

func (t *Texture) Summary() TextureSummary {
	s := TextureSummary{
		Width: t.Width, Height: t.Height, MipCount: t.MipCount,
		FaceCount: t.FaceCount, Cubemap: t.IsCubemap,
		Format: t.Format, PixelFormat: t.PixelFormat,
		DataOffset: t.DataOffset, DataSize: t.DataSize,
		TrailingBytes: t.Trailing,
	}
	for fi := range t.Faces {
		face := FaceSummary{Face: fi}
		for mi := range t.Faces[fi].Mips {
			im := &t.Faces[fi].Mips[mi]
			face.Mips = append(face.Mips, MipSummary{
				Level: mi, Width: im.Width, Height: im.Height,
			})
		}
		s.Faces = append(s.Faces, face)
	}
	return s
}

type TextureSummary struct {
	Width         int           `json:"width"`
	Height        int           `json:"height"`
	MipCount      int           `json:"mipCount"`
	FaceCount     int           `json:"faceCount"`
	Cubemap       bool          `json:"cubemap"`
	Format        string        `json:"format"`
	PixelFormat   PixelFormat   `json:"pixelFormat"`
	DataOffset    int64         `json:"dataOffset"`
	DataSize      int64         `json:"dataSize"`
	TrailingBytes int64         `json:"trailingBytes"`
	Faces         []FaceSummary `json:"faces"`
}

type FaceSummary struct {
	Face int          `json:"face"`
	Mips []MipSummary `json:"mips"`
}

type MipSummary struct {
	Level  int `json:"level"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
