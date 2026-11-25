package vt

import (
	"image"
	"image/color"
	"image/draw"
)

// Page is one immutable-sized block of a virtual texture mip.
type Page struct {
	ID    PageID
	Image *image.RGBA
}

// Texture is the complete high-resolution source plus generated mip chain.
type Texture struct {
	Info Info
	Mips []*image.RGBA
}

// NewProceduralTexture creates a deterministic stone/brick wall. The strong
// page borders and per-page tint make accidental neighbor substitution obvious.
func NewProceduralTexture(info Info) *Texture {
	base := image.NewRGBA(image.Rect(0, 0, info.BaseSize, info.BaseSize))
	for y := 0; y < info.BaseSize; y++ {
		for x := 0; x < info.BaseSize; x++ {
			pageX := x / info.PageSize
			pageY := y / info.PageSize
			localX := x % info.PageSize
			localY := y % info.PageSize

			row := y / 32
			offset := 0
			if row&1 == 1 {
				offset = 32
			}
			brickX := (x + offset) / 64
			mortar := (x+offset)%64 < 3 || y%32 < 3

			noise := int(hashByte(x/2, y/2, 7)) - 128
			brickTint := int(hashByte(brickX, row, 19)) - 128
			r := 142 + noise/5 + brickTint/7 + (pageX*9)%24
			g := 134 + noise/6 + brickTint/8 + (pageY*7)%20
			b := 122 + noise/7 + brickTint/9 + ((pageX+pageY)*5)%18
			if mortar {
				r, g, b = 78+noise/9, 76+noise/9, 74+noise/9
			}
			if localX < 2 || localY < 2 {
				r, g, b = 70, 214, 235
			}
			if localX >= info.PageSize-5 && localY >= info.PageSize-5 {
				r, g, b = 245, 196, 66
			}
			base.SetRGBA(x, y, color.RGBA{R: clampByte(r), G: clampByte(g), B: clampByte(b), A: 255})
		}
	}

	mips := []*image.RGBA{base}
	for mip := 1; mip <= info.MaxMip; mip++ {
		prev := mips[mip-1]
		size := prev.Bounds().Dx() / 2
		next := image.NewRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				p0 := prev.RGBAAt(2*x, 2*y)
				p1 := prev.RGBAAt(2*x+1, 2*y)
				p2 := prev.RGBAAt(2*x, 2*y+1)
				p3 := prev.RGBAAt(2*x+1, 2*y+1)
				next.SetRGBA(x, y, color.RGBA{
					R: uint8((int(p0.R) + int(p1.R) + int(p2.R) + int(p3.R)) / 4),
					G: uint8((int(p0.G) + int(p1.G) + int(p2.G) + int(p3.G)) / 4),
					B: uint8((int(p0.B) + int(p1.B) + int(p2.B) + int(p3.B)) / 4),
					A: 255,
				})
			}
		}
		mips = append(mips, next)
	}
	return &Texture{Info: info, Mips: mips}
}

// LoadPage copies one fixed-size page from the requested mip level.
func (t *Texture) LoadPage(id PageID) (*Page, error) {
	if !t.Info.Valid(id) {
		return nil, ErrInvalidPage
	}
	src := t.Mips[id.Mip]
	origin := image.Pt(id.X*t.Info.PageSize, id.Y*t.Info.PageSize)
	pageImage := image.NewRGBA(image.Rect(0, 0, t.Info.PageSize, t.Info.PageSize))
	draw.Draw(pageImage, pageImage.Bounds(), src, origin, draw.Src)
	return &Page{ID: id, Image: pageImage}, nil
}

// SampleUV is the oracle used by correctness checks.
func (t *Texture) SampleUV(u, v float64, mip int) color.RGBA {
	u, v = ClampUV(u, v)
	mip = t.Info.ClampMip(mip)
	img := t.Mips[mip]
	d := float64(img.Bounds().Dx())
	return sampleBilinear(img, u*d-0.5, v*d-0.5)
}

func sampleBilinear(img *image.RGBA, x, y float64) color.RGBA {
	x0 := intFloor(x)
	y0 := intFloor(y)
	x1raw := x0 + 1
	y1raw := y0 + 1
	fx := x - float64(x0)
	fy := y - float64(y0)
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	c00 := img.RGBAAt(clampInt(x0, 0, w-1), clampInt(y0, 0, h-1))
	c10 := img.RGBAAt(clampInt(x1raw, 0, w-1), clampInt(y0, 0, h-1))
	c01 := img.RGBAAt(clampInt(x0, 0, w-1), clampInt(y1raw, 0, h-1))
	c11 := img.RGBAAt(clampInt(x1raw, 0, w-1), clampInt(y1raw, 0, h-1))

	top := lerpRGBA(c00, c10, fx)
	bottom := lerpRGBA(c01, c11, fx)
	return lerpRGBA(top, bottom, fy)
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t + 0.5),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t + 0.5),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t + 0.5),
		A: 255,
	}
}

func intFloor(v float64) int {
	i := int(v)
	if float64(i) > v {
		i--
	}
	return i
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampByte(v int) uint8 { return uint8(clampInt(v, 0, 255)) }

func hashByte(x, y, seed int) byte {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	return byte(h >> 24)
}
