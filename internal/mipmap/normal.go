package mipmap

import (
	"image"
	"image/color"
	"math"
)

// HeightField is a single-channel periodic height map.
type HeightField struct {
	Width  int
	Height int
	Pix    []float64
}

func NewHeightField(width, height int) *HeightField {
	return &HeightField{Width: width, Height: height, Pix: make([]float64, width*height)}
}

func (h *HeightField) Set(x, y int, value float64) {
	h.Pix[y*h.Width+x] = value
}

func (h *HeightField) HeightAt(x, y int) float64 {
	return h.Pix[y*h.Width+x]
}

func (h *HeightField) WrapHeight(x, y int) float64 {
	x = Mod(x, h.Width)
	y = Mod(y, h.Height)
	return h.HeightAt(x, y)
}

func (h *HeightField) Clone() *HeightField {
	clone := NewHeightField(h.Width, h.Height)
	copy(clone.Pix, h.Pix)
	return clone
}

// DownsampleWrap averages a 4x4 wrapped footprint with unit mass.
func (h *HeightField) DownsampleWrap() *HeightField {
	dstWidth := nextMipSize(h.Width)
	dstHeight := nextMipSize(h.Height)
	dst := NewHeightField(dstWidth, dstHeight)
	offsets := [4]int{-1, 0, 1, 2}

	for y := 0; y < dstHeight; y++ {
		for x := 0; x < dstWidth; x++ {
			var sum float64
			for _, oy := range offsets {
				for _, ox := range offsets {
					sum += h.WrapHeight(2*x+ox, 2*y+oy)
				}
			}
			dst.Set(x, y, sum/16.0)
		}
	}
	return dst
}

func GenerateHeightMipChain(base *HeightField) []*HeightField {
	levels := []*HeightField{base.Clone()}
	for levels[len(levels)-1].Width != 1 || levels[len(levels)-1].Height != 1 {
		levels = append(levels, levels[len(levels)-1].DownsampleWrap())
	}
	return levels
}

// HeightToNormal reconstructs a unit-length normal at every mip level.
// Normals are rebuilt from the reduced height field rather than averaged,
// avoiding shortened vectors and dark mip normals.
func HeightToNormal(height *HeightField, strength float64) *RGBImage {
	normals := NewRGBImage(height.Width, height.Height)
	for y := 0; y < height.Height; y++ {
		for x := 0; x < height.Width; x++ {
			tl := height.WrapHeight(x-1, y-1)
			tc := height.WrapHeight(x, y-1)
			tr := height.WrapHeight(x+1, y-1)
			ml := height.WrapHeight(x-1, y)
			mr := height.WrapHeight(x+1, y)
			bl := height.WrapHeight(x-1, y+1)
			bc := height.WrapHeight(x, y+1)
			br := height.WrapHeight(x+1, y+1)

			dx := (tr + 2*mr + br - tl - 2*ml - bl) / 4.0 * strength
			dy := (bl + 2*bc + br - tl - 2*tc - tr) / 4.0 * strength
			nx := -dx
			ny := -dy
			nz := 1.0
			length := math.Hypot(math.Hypot(nx, ny), nz)
			normals.Set(x, y, nx/length, ny/length, nz/length)
		}
	}
	return normals
}

// ToNormalRGBA stores vector components as linear data, not color-managed sRGB.
func (img *RGBImage) ToNormalRGBA() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			nx, ny, nz := img.RGB(x, y)
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round((clamp01(nx*0.5 + 0.5)) * 255)),
				G: uint8(math.Round((clamp01(ny*0.5 + 0.5)) * 255)),
				B: uint8(math.Round((clamp01(nz*0.5 + 0.5)) * 255)),
				A: 255,
			})
		}
	}
	return dst
}

// NormalMapMipChain is a convenience wrapper used by the CLI and tests.
func NormalMapMipChain(heightLevels []*HeightField, strength float64) []*RGBImage {
	result := make([]*RGBImage, 0, len(heightLevels))
	for _, level := range heightLevels {
		result = append(result, HeightToNormal(level, strength))
	}
	return result
}
