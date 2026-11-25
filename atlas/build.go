package atlas

import (
	"fmt"
	"image"
	"image/color"
)

type SpriteImage struct {
	ID    string
	Image image.Image
}

type BuildOptions struct {
	Pack       PackOptions
	MipLevels  int
	Extrude    bool
	Background color.Color
}

type BuiltAtlas struct {
	Layout     *Layout
	Base       *image.RGBA
	Mips       []*image.RGBA
	Sources    map[string]*image.RGBA
	SourceMips map[string][]*image.RGBA
	FrameMips  map[string][]*image.RGBA
}

// Build packs sprites, copies pixels with optional 90-degree rotation,
// extrudes gutters, and creates the validated mip chain.
func BuildAtlas(sprites []SpriteImage, opts BuildOptions) (*BuiltAtlas, error) {
	if len(sprites) == 0 {
		return nil, ErrNoSprites
	}
	if opts.MipLevels < 1 {
		opts.MipLevels = 1
	}
	if opts.Background == nil {
		opts.Background = color.RGBA{R: 18, G: 20, B: 28, A: 255}
	}

	opts.Pack.Policy.MipLevels = opts.MipLevels
	if opts.Extrude && opts.Pack.Policy.Padding < 1 {
		opts.Pack.Policy.Padding = 1
	}
	if opts.Pack.Alignment <= 0 && opts.MipLevels > 1 {
		opts.Pack.Alignment = 1 << uint(opts.MipLevels-1)
	}

	rgbaSprites := make([]SpriteImage, len(sprites))
	specs := make([]Sprite, len(sprites))
	sources := make(map[string]*image.RGBA, len(sprites))
	for i, sp := range sprites {
		if sp.ID == "" {
			sp.ID = fmt.Sprintf("sprite-%02d", i)
		}
		img := ToRGBA(sp.Image)
		rgbaSprites[i] = SpriteImage{ID: sp.ID, Image: img}
		specs[i] = Sprite{ID: sp.ID, W: img.Bounds().Dx(), H: img.Bounds().Dy()}
		sources[sp.ID] = img
	}

	layout, err := Pack(specs, opts.Pack)
	if err != nil {
		return nil, err
	}

	base := NewRGBA(layout.Width, layout.Height, opts.Background)
	for _, sp := range rgbaSprites {
		p, ok := layout.PlacementByID(sp.ID)
		if !ok {
			return nil, fmt.Errorf("%w: placement missing for %s", ErrPackingFailed, sp.ID)
		}
		src := sources[sp.ID]
		if p.Rotated {
			rotated := RotateCW(src)
			copyRectNearest(base, rotated, R(0, 0, rotated.Bounds().Dx(), rotated.Bounds().Dy()), p.Frame)
		} else {
			copyRectNearest(base, src, R(0, 0, src.Bounds().Dx(), src.Bounds().Dy()), p.Frame)
		}
		if opts.Extrude {
			ExtrudePlacement(base, p, layout.PackPadding)
		}
	}

	mips := MipChain(base, opts.MipLevels)
	if opts.Extrude {
		for level := 1; level < len(mips); level++ {
			ExpandMipPadding(mips[level], layout, level)
		}
	}

	sourceMips := make(map[string][]*image.RGBA, len(sources))
	frameMips := make(map[string][]*image.RGBA, len(sources))
	for id, src := range sources {
		sourceMips[id] = MipChain(src, opts.MipLevels)
		p, ok := layout.PlacementByID(id)
		if !ok {
			return nil, fmt.Errorf("%w: placement missing for %s", ErrPackingFailed, id)
		}
		frameSrc := src
		if p.Rotated {
			frameSrc = RotateCW(src)
		}
		frameMips[id] = MipChain(frameSrc, opts.MipLevels)
	}

	return &BuiltAtlas{
		Layout: layout, Base: base, Mips: mips,
		Sources: sources, SourceMips: sourceMips, FrameMips: frameMips,
	}, nil
}
