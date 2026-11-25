package demo

import (
	"image"
	"image/color"

	"atlaspad/atlas"
)

type artSpec struct {
	id   string
	w, h int
	base color.RGBA
	kind int
}

func sampleSprites() []atlas.SpriteImage {
	specs := []artSpec{
		{"tex-01-grid", 128, 128, rgba(40, 170, 255), 0},
		{"tex-02-wide", 160, 64, rgba(255, 170, 40), 1},
		{"tex-03-tall", 64, 160, rgba(120, 230, 90), 2},
		{"tex-04-check", 96, 96, rgba(225, 70, 190), 3},
		{"tex-05-diag", 112, 80, rgba(255, 90, 90), 4},
		{"tex-06-dots", 80, 112, rgba(70, 225, 220), 5},
		{"tex-07-block", 64, 64, rgba(235, 220, 70), 6},
		{"tex-08-banner", 176, 48, rgba(160, 110, 255), 7},
		{"tex-09-column", 48, 144, rgba(255, 130, 190), 8},
		{"tex-10-medium", 96, 64, rgba(120, 210, 120), 9},
	}

	out := make([]atlas.SpriteImage, 0, len(specs))
	for i, spec := range specs {
		img := image.NewRGBA(image.Rect(0, 0, spec.w, spec.h))
		edge := edgeColor(spec.base, i)
		for y := 0; y < spec.h; y++ {
			for x := 0; x < spec.w; x++ {
				c := patternColor(spec, x, y)
				if x < 5 || y < 5 || x >= spec.w-5 || y >= spec.h-5 {
					c = edge
				}
				img.SetRGBA(x, y, c)
			}
		}
		out = append(out, atlas.SpriteImage{ID: spec.id, Image: img})
	}
	return out
}

func patternColor(spec artSpec, x, y int) color.RGBA {
	base := spec.base
	switch spec.kind {
	case 0:
		if (x/16+y/16)%2 == 0 {
			return shade(base, 45)
		}
		return shade(base, -35)
	case 1:
		if y/8%2 == 0 {
			return shade(base, 55)
		}
		return shade(base, -45)
	case 2:
		if x/8%2 == 0 {
			return shade(base, 55)
		}
		return shade(base, -45)
	case 3:
		if (x/12+y/12)%2 == 0 {
			return rgba(245, 245, 245)
		}
		return rgba(25, 28, 38)
	case 4:
		if (x+y)/10%2 == 0 {
			return shade(base, 55)
		}
		return shade(base, -50)
	case 5:
		dx := x%20 - 10
		dy := y%20 - 10
		if dx*dx+dy*dy < 36 {
			return rgba(245, 255, 255)
		}
		return shade(base, -35)
	case 6:
		if x < 32 && y < 32 || x >= 32 && y >= 32 {
			return shade(base, 60)
		}
		return shade(base, -45)
	case 7:
		if x/14%2 == 0 {
			return rgba(30, 32, 44)
		}
		return shade(base, 35)
	case 8:
		if y/14%2 == 0 {
			return rgba(30, 32, 44)
		}
		return shade(base, 35)
	default:
		if (x/8+y/8)%3 == 0 {
			return shade(base, 50)
		}
		return base
	}
}

func rgba(r, g, b int) color.RGBA {
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func shade(c color.RGBA, delta int) color.RGBA {
	clamp := func(v int) uint8 {
		if v+delta < 0 {
			return 0
		}
		if v+delta > 255 {
			return 255
		}
		return uint8(v + delta)
	}
	return color.RGBA{R: clamp(int(c.R)), G: clamp(int(c.G)), B: clamp(int(c.B)), A: 255}
}

func edgeColor(c color.RGBA, i int) color.RGBA {
	if i%3 == 0 {
		return rgba(255, 255, 255)
	}
	if i%3 == 1 {
		return rgba(18, 20, 28)
	}
	return color.RGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: 255}
}
