package stab

import (
	"deshake/internal/geom"
	"deshake/internal/synth"
)

// Stabilize3D undoes rolling shutter using per-row camera poses and then
// reprojects onto the smoothed reference pose. Each output pixel is mapped
// back to the exact source row whose readout instant it was captured at, so
// the fast-turn jelly is removed instead of approximated by one translation.
func Stabilize3D(cam geom.Camera, f *synth.Frame) *FrameResult {
	w, h := f.W, f.H
	out := newResult(w, h)
	hRef := cam.WorldToPixel(f.RefPose)
	hRefInv := hRef.Invert()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ox, oy := float64(x)+0.5, float64(y)+0.5

			// World point shown at this output pixel under the reference pose.
			wx, wy := hRefInv.Apply(ox, oy)

			// Fixed-point solve: the source row depends on readout time, which
			// itself depends on the source row. Three iterations converge below
			// 1e-3 px for the motion magnitudes used here.
			rowF := float64(y) / float64(h-1)
			var sx, sy float64
			ok := false
			for it := 0; it < 3; it++ {
				hs := rowHomography(f, rowF)
				sx, sy = hs.Apply(wx, wy)
				next := geom.Clamp((sy-0.5)/float64(h-1), 0, 1)
				if abs(next-rowF) < 1e-4 && it >= 1 {
					rowF = next
					ok = true
					break
				}
				rowF = next
				if it == 2 {
					ok = true
				}
			}

			i := y*w + x
			if !ok || sx < 1 || sy < 1 || sx > float64(w) || sy > float64(h) {
				out.markHole(x, y)
				continue
			}
			r, g, b, sampled := sampleBilinear(f.Img.Pix, w, h, sx, sy)
			if !sampled {
				out.markHole(x, y)
				continue
			}
			out.Valid[i] = true
			j := i * 3
			out.Pix[j], out.Pix[j+1], out.Pix[j+2] = r, g, b
		}
	}

	out.Box = largestRect(out.Valid, w, h)
	out.Edge = boxMargin(w, h, out.Box)
	return out
}

// rowHomography interpolates the per-row source homography at fractional row
// position rowF in [0,1], matching the sub-pixel readout instant.
func rowHomography(f *synth.Frame, rowF float64) geom.Mat3 {
	pos := geom.Clamp(rowF, 0, 1) * float64(len(f.RowH)-1)
	i := int(pos)
	if i >= len(f.RowH)-1 {
		return f.RowH[len(f.RowH)-1]
	}
	a := f.RowH[i].M
	b := f.RowH[i+1].M
	t := pos - float64(i)
	var m [9]float64
	for k := range m {
		m[k] = a[k] + (b[k]-a[k])*t
	}
	return geom.Mat3{M: m}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
