package hdrlab

import "math"

// SatClip is the gamma threshold above which a pixel is treated as clipped.
// It sits ~3 noise sigmas below 1.0 so genuinely clipped pixels with noise
// are never mistaken for valid highlights.
const SatClip = 0.95

// HatWeight is the classic Debevec-style hat weight on gamma pixel values:
// trust mid-tones, distrust values near 0 and 1, and discard clipped pixels.
func HatWeight(z float64) float64 {
	if z >= SatClip {
		return 0
	}
	w := z
	if z > 0.5 {
		w = 1 - z
	}
	return w + 1e-3
}

// MergeResult holds the HDR radiance map plus diagnostics.
type MergeResult struct {
	HDR              *Img   // linear radiance, OutW x OutH
	Ghost            []bool // pixels where a non-reference frame was rejected
	Residual         *Img   // per-pixel mean compensated luma disagreement (R channel)
	MotionFraction   float64
	MeanDisagreement float64
}

// MergeHDR aligns the brackets, then blends them with hat weights. Pixels
// where a non-reference frame disagrees with the (aligned) reference by
// more than deghostTol in linear radiance are rejected for that frame, so
// moving objects keep the reference-frame position and leave no ghosts.
// A clipped channel in a non-reference frame is treated as a lower-bound
// radiance: it only agrees when the reference is bright enough to clip too.
func MergeHDR(frames []*Img, ts []float64, shifts []Shift, ref int) *MergeResult {
	const deghostTol = 0.10
	hdr := NewImg(OutW, OutH)
	resid := NewImg(OutW, OutH)
	ghost := make([]bool, OutW*OutH)
	nf := len(frames)
	keep := make([]bool, nf)
	var motion, disN int
	var disSum float64
	for oy := 0; oy < OutH; oy++ {
		for ox := 0; ox < OutW; ox++ {
			x := ox + Margin
			y := oy + Margin
			var lr [3]float64
			for c := 0; c < 3; c++ {
				lr[c] = lin(frames[ref].Pix[(y*SceneW+x)*3+c]) / ts[ref]
			}
			refLum := 0.2126*lr[0] + 0.7152*lr[1] + 0.0722*lr[2]
			refLumSat := frames[ref].Luma(x, y) > SatClip
			var resSum float64
			var resN int
			for i := range frames {
				keep[i] = true
				if i == ref {
					continue
				}
				sx := x - shifts[i].DX
				sy := y - shifts[i].DY
				lumSat := refLumSat || frames[i].Luma(sx, sy) > SatClip
				if !lumSat {
					dl := lin(frames[i].Luma(sx, sy))/ts[i] - refLum
					if dl < 0 {
						dl = -dl
					}
					resSum += dl
					resN++
				}
				// Reject frame i if any channel that is unclipped in BOTH
				// frames disagrees with the reference radiance. Channels
				// clipped in either frame carry no comparable signal.
				for c := 0; c < 3; c++ {
					zi := frames[i].Pix[(sy*SceneW+sx)*3+c]
					zr := frames[ref].Pix[(y*SceneW+x)*3+c]
					if zr > SatClip {
						continue // reference clipped: frame i owns the highlights
					}
					if zi > SatClip {
						if lr[c] < lin(SatClip)/ts[i]-deghostTol {
							keep[i] = false
							break
						}
						continue
					}
					e := lin(zi) / ts[i]
					d := e - lr[c]
					if d < 0 {
						d = -d
					}
					if d > deghostTol {
						keep[i] = false
						break
					}
				}
			}
			if resN > 0 {
				resid.Pix[(oy*OutW+ox)*3] = resSum / float64(resN)
			}
			isGhost := false
			for i := range frames {
				if !keep[i] {
					isGhost = true
					break
				}
			}
			if isGhost {
				ghost[oy*OutW+ox] = true
				motion++
			} else if resN > 0 {
				disSum += resSum / float64(resN)
				disN++
			}
			for c := 0; c < 3; c++ {
				var num, den float64
				clipLB := -1.0
				for i := range frames {
					if !keep[i] {
						continue
					}
					sx := x - shifts[i].DX
					sy := y - shifts[i].DY
					z := frames[i].Pix[(sy*SceneW+sx)*3+c]
					w := HatWeight(z)
					if z >= SatClip {
						// Clipped sample: radiance is at least this high.
						if lb := lin(SatClip) / ts[i]; lb > clipLB {
							clipLB = lb
						}
					}
					num += w * lin(z) / ts[i]
					den += w
				}
				e := lr[c]
				if den > 1e-6 {
					e = num / den
				} else if clipLB > 0 {
					// Every contributing frame clipped here: use the
					// strongest lower bound instead of the reference's
					// own clipped (too dark) value.
					e = clipLB
				}
				hdr.Pix[(oy*OutW+ox)*3+c] = e
			}
		}
	}
	res := &MergeResult{
		HDR:            hdr,
		Ghost:          ghost,
		Residual:       resid,
		MotionFraction: float64(motion) / float64(OutW*OutH),
	}
	if disN > 0 {
		res.MeanDisagreement = disSum / float64(disN)
	}
	return res
}

// AverageFrames averages the gamma frames, optionally after alignment.
// It is the naive baseline: it still clips highlights and keeps ghosts.
func AverageFrames(frames []*Img, shifts []Shift, aligned bool) *Img {
	out := NewImg(OutW, OutH)
	n := float64(len(frames))
	for oy := 0; oy < OutH; oy++ {
		for ox := 0; ox < OutW; ox++ {
			x := ox + Margin
			y := oy + Margin
			for c := 0; c < 3; c++ {
				var sum float64
				for i, f := range frames {
					sx, sy := x, y
					if aligned {
						sx -= shifts[i].DX
						sy -= shifts[i].DY
					}
					sum += f.Pix[(sy*SceneW+sx)*3+c]
				}
				out.Pix[(oy*OutW+ox)*3+c] = sum / n
			}
		}
	}
	return out
}

// Tonemap applies a global photographic exposure curve (1-exp(-L/key)),
// which keeps more highlight contrast than Reinhard, then gamma encodes.
func Tonemap(hdr *Img) *Img {
	out := NewImg(hdr.W, hdr.H)
	for i, v := range hdr.Pix {
		if v < 0 {
			v = 0
		}
		out.Pix[i] = math.Pow(1-math.Exp(-v/1.5), 1.0/2.2)
	}
	return out
}
