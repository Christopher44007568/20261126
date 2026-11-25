package mipchain

import (
	"errors"
	"fmt"
	"image"
	"math"

	"texturelab/colorspace"
	"texturelab/texcontainer"
)

// BuildChain downsamples src count-1 times using exact 2x2 box filters.
//
// sRGB images are averaged after linearization to prevent gamma-darkened mips.
// Linear normal maps are averaged as vectors and renormalized.
func BuildChain(src *image.RGBA, count int, space colorspace.Space, normalMap bool) ([]*image.RGBA, error) {
	if src == nil {
		return nil, errors.New("mipchain: nil source image")
	}
	if count < 1 {
		return nil, fmt.Errorf("mipchain: level count must be positive, got %d", count)
	}
	levels := []*image.RGBA{cloneRGBA(src)}
	for len(levels) < count {
		next, err := Downsample2x(levels[len(levels)-1], space, normalMap)
		if err != nil {
			return nil, err
		}
		levels = append(levels, next)
	}
	return levels, nil
}

// Downsample2x performs a deterministic 2x2 box reduction.
func Downsample2x(src *image.RGBA, space colorspace.Space, normalMap bool) (*image.RGBA, error) {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 2 || height < 2 || width%2 != 0 || height%2 != 0 {
		return nil, fmt.Errorf("mipchain: expected even dimensions >= 2, got %dx%d", width, height)
	}

	dst := image.NewRGBA(image.Rect(0, 0, width/2, height/2))
	for y := 0; y < height/2; y++ {
		for x := 0; x < width/2; x++ {
			x0 := bounds.Min.X + 2*x
			y0 := bounds.Min.Y + 2*y
			i00 := src.PixOffset(x0, y0)
			i10 := src.PixOffset(x0+1, y0)
			i01 := src.PixOffset(x0, y0+1)
			i11 := src.PixOffset(x0+1, y0+1)

			out := averageFour(src.Pix, [4]int{i00, i10, i01, i11}, space, normalMap)
			offset := dst.PixOffset(x, y)
			dst.Pix[offset+0] = out[0]
			dst.Pix[offset+1] = out[1]
			dst.Pix[offset+2] = out[2]
			dst.Pix[offset+3] = out[3]
		}
	}
	return dst, nil
}

func averageFour(pix []byte, offsets [4]int, space colorspace.Space, normalMap bool) [4]uint8 {
	var alpha float64
	if normalMap {
		var nx, ny, nz float64
		for _, offset := range offsets {
			nx += 2*float64(pix[offset+0])/255.0 - 1
			ny += 2*float64(pix[offset+1])/255.0 - 1
			nz += 2*float64(pix[offset+2])/255.0 - 1
			alpha += float64(pix[offset+3]) / 255.0
		}
		nx /= 4
		ny /= 4
		nz /= 4
		length := math.Sqrt(nx*nx + ny*ny + nz*nz)
		if length > 1e-12 {
			nx /= length
			ny /= length
			nz /= length
		} else {
			nx, ny, nz = 0, 0, 1
		}
		return [4]uint8{
			clampByte((nx*0.5 + 0.5) * 255),
			clampByte((ny*0.5 + 0.5) * 255),
			clampByte((nz*0.5 + 0.5) * 255),
			clampByte(alpha / 4 * 255),
		}
	}

	if space == colorspace.SRGB {
		var linear [3]float64
		for _, offset := range offsets {
			sample := colorspace.LinearizeRGBA(colorspace.SRGB,
				pix[offset+0], pix[offset+1], pix[offset+2], pix[offset+3])
			linear[0] += sample[0]
			linear[1] += sample[1]
			linear[2] += sample[2]
			alpha += sample[3]
		}
		return [4]uint8{
			colorspace.LinearToSRGB(linear[0] / 4),
			colorspace.LinearToSRGB(linear[1] / 4),
			colorspace.LinearToSRGB(linear[2] / 4),
			clampByte(alpha / 4 * 255),
		}
	}

	var sum [4]float64
	for _, offset := range offsets {
		for channel := 0; channel < 4; channel++ {
			sum[channel] += float64(pix[offset+channel])
		}
	}
	return [4]uint8{
		clampByte(sum[0] / 4),
		clampByte(sum[1] / 4),
		clampByte(sum[2] / 4),
		clampByte(sum[3] / 4),
	}
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	clone := image.NewRGBA(src.Bounds())
	copy(clone.Pix, src.Pix)
	return clone
}

// ToContainerLevels copies tightly packed RGBA images into container levels.
func ToContainerLevels(images []*image.RGBA) ([]texcontainer.Level, error) {
	levels := make([]texcontainer.Level, 0, len(images))
	for _, img := range images {
		if img.Stride != img.Rect.Dx()*4 {
			return nil, fmt.Errorf("mipchain: unsupported padded stride %d for width %d", img.Stride, img.Rect.Dx())
		}
		levels = append(levels, texcontainer.Level{
			Width:  img.Rect.Dx(),
			Height: img.Rect.Dy(),
			Pixels: append([]byte(nil), img.Pix...),
		})
	}
	return levels, nil
}
