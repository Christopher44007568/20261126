package main

import (
	"fmt"
	"hash/fnv"
)

type DiffStats struct {
	RefHits         int     `json:"ref_hits"`
	DegradedHits    int     `json:"degraded_hits"`
	MissingPixels   int     `json:"missing_pixels"`
	FalseHitPixels  int     `json:"false_hit_pixels"`
	DifferentPixels int     `json:"different_pixels"`
	BorderPixels    int     `json:"border_pixels"`
	ErrorRate       float64 `json:"error_rate"`
}

func CompareMasks(ref, degraded []bool) DiffStats {
	stats := DiffStats{}
	for i := range ref {
		if ref[i] {
			stats.RefHits++
		}
		if i < len(degraded) && degraded[i] {
			stats.DegradedHits++
		}
		switch {
		case ref[i] && !degraded[i]:
			stats.MissingPixels++
		case !ref[i] && degraded[i]:
			stats.FalseHitPixels++
		}
	}
	stats.DifferentPixels = stats.MissingPixels + stats.FalseHitPixels
	if len(ref) > 0 {
		stats.ErrorRate = float64(stats.DifferentPixels) / float64(len(ref))
	}
	return stats
}

// AnnotateSilhouetteErrors draws a thick two-color contour through every
// coverage boundary that differs from the full-quality reference. The border
// intentionally follows both sides of the silhouette so missing spikes and
// coarse overruns remain visible in the emitted PPM.
func AnnotateSilhouetteErrors(img *Image, refMask, degradedMask []bool, thickness int) DiffStats {
	stats := CompareMasks(refMask, degradedMask)
	if thickness < 1 {
		thickness = 2
	}
	w, h := img.Width, img.Height
	edge := make([]bool, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if !img.inBounds(nx, ny) {
						continue
					}
					j := ny*w + nx
					if refMask[i] != refMask[j] || degradedMask[i] != degradedMask[j] {
						edge[i] = true
					}
				}
			}
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			best := thickness + 2
			radius := thickness + 1
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					nx, ny := x+dx, y+dy
					if !img.inBounds(nx, ny) {
						continue
					}
					j := ny*w + nx
					if edge[j] {
						d := maxInt(absInt(dx), absInt(dy))
						if d < best {
							best = d
						}
					}
				}
			}
			if best > thickness {
				continue
			}
			stats.BorderPixels++
			if best == 0 {
				img.Pixels[i] = colorErrorCore
			} else {
				img.Pixels[i] = colorErrorOuter
			}
		}
	}
	return stats
}

func DrawTierFrame(img *Image, color RGB, width int) {
	if width <= 0 {
		return
	}
	for t := 0; t < width; t++ {
		for x := t; x < img.Width-t; x++ {
			img.Set(x, t, color)
			img.Set(x, img.Height-1-t, color)
		}
		for y := t; y < img.Height-t; y++ {
			img.Set(t, y, color)
			img.Set(img.Width-1-t, y, color)
		}
	}
}

func Fingerprint(img *Image) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%dx%d", img.Width, img.Height)
	for _, p := range img.Pixels {
		_, _ = h.Write([]byte{p.R, p.G, p.B})
	}
	return h.Sum64()
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
