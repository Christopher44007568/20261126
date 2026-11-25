package atlas

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// RuntimeAtlas is the minimal CPU equivalent of uploading the RGBA mip chain
// and UV rectangles to a sampler.
type RuntimeAtlas struct {
	Layout *Layout
	Mips   []*image.RGBA
}

// NewRuntimeAtlas wraps a built atlas for UV lookup and mip sampling.
func NewRuntimeAtlas(b *BuiltAtlas) *RuntimeAtlas {
	return &RuntimeAtlas{Layout: b.Layout, Mips: b.Mips}
}

func (r *RuntimeAtlas) level(level int) *image.RGBA {
	if len(r.Mips) == 0 {
		return nil
	}
	level = clampInt(level, 0, len(r.Mips)-1)
	return r.Mips[level]
}

// SampleMip samples one integer mip by sprite-local UV.
func (r *RuntimeAtlas) SampleMip(id string, u, v float64, level int) (color.RGBA, Placement, error) {
	p, ok := r.Layout.PlacementByID(id)
	if !ok {
		return color.RGBA{}, Placement{}, fmt.Errorf("unknown sprite %q", id)
	}
	img := r.level(level)
	if img == nil {
		return color.RGBA{}, Placement{}, fmt.Errorf("atlas has no mip levels")
	}
	au, av := SpriteToAtlasUV(p, r.Layout.Width, r.Layout.Height, u, v)
	return SampleBilinear(img, au, av), p, nil
}

// SampleLOD performs trilinear interpolation between two mip levels.
func (r *RuntimeAtlas) SampleLOD(id string, u, v, lod float64) (color.RGBA, error) {
	if len(r.Mips) == 0 {
		return color.RGBA{}, fmt.Errorf("atlas has no mip levels")
	}
	maxLevel := float64(len(r.Mips) - 1)
	lod = clampFloat(lod, 0, maxLevel)
	lo := int(math.Floor(lod))
	hi := minInt(lo+1, len(r.Mips)-1)
	t := lod - float64(lo)

	c0, _, err := r.SampleMip(id, u, v, lo)
	if err != nil {
		return color.RGBA{}, err
	}
	c1, _, err := r.SampleMip(id, u, v, hi)
	if err != nil {
		return color.RGBA{}, err
	}
	return lerpPremul(c0, c1, t), nil
}
