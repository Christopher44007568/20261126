package atlas

import (
	"image"
	"image/color"
)

// DownsampleBox halves an RGBA image with an alpha-correct 2x2 box filter.
// Odd borders clamp the missing sample to the edge, which keeps repeated
// padding and source mip generation on the same convention.
func DownsampleBox(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dw := maxInt(1, (sw+1)/2)
	dh := maxInt(1, (sh+1)/2)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			var sumR, sumG, sumB, sumA uint32
			for oy := 0; oy < 2; oy++ {
				sy := b.Min.Y + clampInt(2*dy+oy, 0, sh-1)
				for ox := 0; ox < 2; ox++ {
					sx := b.Min.X + clampInt(2*dx+ox, 0, sw-1)
					c := src.RGBAAt(sx, sy)
					a := uint32(c.A)
					sumR += uint32(c.R) * a
					sumG += uint32(c.G) * a
					sumB += uint32(c.B) * a
					sumA += a
				}
			}
			a := uint8((sumA + 2) / 4)
			var r, g, b8 uint8
			if sumA != 0 {
				r = uint8(clampInt(int(sumR/sumA), 0, 255))
				g = uint8(clampInt(int(sumG/sumA), 0, 255))
				b8 = uint8(clampInt(int(sumB/sumA), 0, 255))
			}
			dst.SetRGBA(dx, dy, color.RGBA{R: r, G: g, B: b8, A: a})
		}
	}
	return dst
}

// MipChain returns src followed by box-downsampled levels. The requested
// count is clamped once both dimensions reach one texel.
func MipChain(src *image.RGBA, levels int) []*image.RGBA {
	if levels < 1 {
		levels = 1
	}
	chain := []*image.RGBA{src}
	for len(chain) < levels {
		last := chain[len(chain)-1]
		b := last.Bounds()
		if b.Dx() == 1 && b.Dy() == 1 {
			chain = append(chain, cloneRGBA(last))
			continue
		}
		chain = append(chain, DownsampleBox(last))
	}
	return chain
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	copy(dst.Pix, src.Pix)
	return dst
}

// ScaledRect maps a level-zero rect to ceil-sized mip coordinates. Ceil keeps
// the partial final block that contains the replicated edge texel.
func ScaledRect(r Rect, level int) Rect {
	if level <= 0 {
		return r
	}
	scale := 1 << uint(level)
	w := (r.W + scale - 1) / scale
	h := (r.H + scale - 1) / scale
	return R(r.X>>uint(level), r.Y>>uint(level), maxInt(1, w), maxInt(1, h))
}

// ExpandMipPadding paints the scaled reserved gutter for every placement at
// one mip level. The gutter width in mip texels is PackPadding / 2^level.
func ExpandMipPadding(mip *image.RGBA, layout *Layout, level int) int {
	if layout == nil || layout.PackPadding <= 0 {
		return 0
	}
	levelPad := layout.PackPadding >> uint(level)
	if levelPad <= 0 {
		return 0
	}
	painted := 0
	for _, p := range layout.Placements {
		frame := ScaledRect(p.Frame, level)
		painted += ExtrudeRect(mip, frame, frame.Expand(levelPad))
	}
	return painted
}
