package vt

import (
	"fmt"
	"image/color"
)

// Verify checks routing and sampled content for every framebuffer pixel.
// A mismatch means either an illegal same-mip neighbor substitution or sampled
// page pixels that do not match the authoritative mip chain.
func (f *Frame) Verify(tex *Texture) error {
	if len(f.Pixels) != f.Metrics.Width*f.Metrics.Height {
		return fmt.Errorf("diagnostic length %d does not match frame %dx%d",
			len(f.Pixels), f.Metrics.Width, f.Metrics.Height)
	}
	for i, pixel := range f.Pixels {
		x := i % f.Metrics.Width
		y := i / f.Metrics.Width
		mask := f.MissMask.GrayAt(x, y).Y
		switch pixel.Status {
		case StatusExact:
			if mask != 0 {
				return fmt.Errorf("pixel %d,%d exact page has white miss mask", x, y)
			}
			if pixel.Resolved != pixel.Requested {
				return fmt.Errorf("pixel %d,%d exact status resolves %s != %s",
					x, y, pixel.Resolved, pixel.Requested)
			}
		case StatusAncestor:
			if mask != 255 {
				return fmt.Errorf("pixel %d,%d fallback has black miss mask", x, y)
			}
			expected := tex.Info.Ancestor(pixel.Requested, pixel.Resolved.Mip)
			if expected != pixel.Resolved || pixel.Resolved.Mip < pixel.Requested.Mip {
				return fmt.Errorf("pixel %d,%d used illegal substitute %s for %s",
					x, y, pixel.Resolved, pixel.Requested)
			}
		case StatusMissing:
			if mask != 255 {
				return fmt.Errorf("pixel %d,%d missing status has black miss mask", x, y)
			}
			continue
		default:
			return fmt.Errorf("pixel %d,%d has unknown status %d", x, y, pixel.Status)
		}

		expectedColor := samplePageClampedOracle(tex, pixel.Resolved, pixel.U, pixel.V)
		actualColor := f.Image.RGBAAt(x, y)
		if colorDistance(actualColor, expectedColor) > 2 {
			return fmt.Errorf(
				"pixel %d,%d content mismatch: uv=(%.5f,%.5f) requested=%s resolved=%s actual=%v expected=%v",
				x, y, pixel.U, pixel.V, pixel.Requested, pixel.Resolved,
				rgbaTriple(actualColor), rgbaTriple(expectedColor),
			)
		}
	}
	return nil
}

// samplePageClampedOracle mirrors the renderer contract: bilinear taps are
// clamped to the resolved page rectangle. It must never blend the adjacent
// same-mip page into a missing or fallback result.
func samplePageClampedOracle(tex *Texture, id PageID, u, v float64) color.RGBA {
	img := tex.Mips[id.Mip]
	dim := float64(img.Bounds().Dx())
	gx := u*dim - 0.5
	gy := v*dim - 0.5
	x0 := intFloor(gx)
	y0 := intFloor(gy)
	x1raw := x0 + 1
	y1raw := y0 + 1
	fx := gx - float64(x0)
	fy := gy - float64(y0)

	originX := id.X * tex.Info.PageSize
	originY := id.Y * tex.Info.PageSize
	maxX := originX + tex.Info.PageSize - 1
	maxY := originY + tex.Info.PageSize - 1

	c00 := img.RGBAAt(clampInt(x0, originX, maxX), clampInt(y0, originY, maxY))
	c10 := img.RGBAAt(clampInt(x1raw, originX, maxX), clampInt(y0, originY, maxY))
	c01 := img.RGBAAt(clampInt(x0, originX, maxX), clampInt(y1raw, originY, maxY))
	c11 := img.RGBAAt(clampInt(x1raw, originX, maxX), clampInt(y1raw, originY, maxY))
	top := lerpRGBA(c00, c10, fx)
	bottom := lerpRGBA(c01, c11, fx)
	return lerpRGBA(top, bottom, fy)
}

func colorDistance(a, b color.RGBA) int {
	dr := absInt(int(a.R) - int(b.R))
	dg := absInt(int(a.G) - int(b.G))
	db := absInt(int(a.B) - int(b.B))
	if dg > dr {
		dr = dg
	}
	if db > dr {
		dr = db
	}
	return dr
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func rgbaTriple(c color.RGBA) [3]uint8 { return [3]uint8{c.R, c.G, c.B} }
