package stab

import (
	"deshake/internal/geom"
	"deshake/internal/synth"
)

// Stabilize2D applies a single global translation per frame, derived from the
// mid-exposure pose. It cancels whole-frame 2-D shake but cannot undo the
// per-row rolling-shutter geometry, so jelly/wavy content remains.
func Stabilize2D(cam geom.Camera, f *synth.Frame) *FrameResult {
	w, h := f.W, f.H
	out := newResult(w, h)

	// Where the reference and actual mid poses place the image center.
	hAct := cam.WorldToPixel(f.MidPose)
	hRef := cam.WorldToPixel(f.RefPose)
	ax, ay := hAct.Apply(0, 0)
	rx, ry := hRef.Apply(0, 0)
	dx, dy := ax-rx, ay-ry // source pixel offset relative to stabilized output

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Output pixel center samples the same frame at the offset center.
			sx := float64(x) + 0.5 + dx
			sy := float64(y) + 0.5 + dy
			r, g, b, ok := sampleBilinear(f.Img.Pix, w, h, sx, sy)
			i := y*w + x
			if !ok {
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

// largestRect finds the maximal axis-aligned all-valid rectangle by scanning
// per-row valid runs and intersecting them across contiguous row bands.
func largestRect(valid []bool, w, h int) Rect {
	best := Rect{}
	bestArea := 0
	for top := 0; top < h; top++ {
		l, r := 0, w
		for bottom := top; bottom < h; bottom++ {
			rowL, rowR := -1, -1
			for x := 0; x < w; x++ {
				if valid[bottom*w+x] {
					if rowL < 0 {
						rowL = x
					}
					rowR = x + 1
				}
			}
			if rowL < 0 {
				break
			}
			if bottom == top {
				l, r = rowL, rowR
			} else {
				if rowL > l {
					l = rowL
				}
				if rowR < r {
					r = rowR
				}
			}
			if r <= l {
				break
			}
			area := (r - l) * (bottom - top + 1)
			if area > bestArea {
				bestArea = area
				best = Rect{X0: l, Y0: top, X1: r, Y1: bottom + 1}
			}
		}
	}
	return best
}

func boxMargin(w, h int, r Rect) Margin {
	return Margin{
		Top:    float64(r.Y0),
		Bottom: float64(h - r.Y1),
		Left:   float64(r.X0),
		Right:  float64(w - r.X1),
	}
}
