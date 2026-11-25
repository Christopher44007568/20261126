package main

import "math"

type DiffStats struct {
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	PixelCount      int     `json:"pixel_count"`
	DifferentPixels int     `json:"different_pixels"`
	MaxR            float64 `json:"max_r"`
	MaxG            float64 `json:"max_g"`
	MaxB            float64 `json:"max_b"`
	MaxLuma         float64 `json:"max_luma"`
	Gain            float64 `json:"gain"`
	Identical       bool    `json:"identical"`
}

func BuildDiff(a, b *Canvas, gain float64) (*Canvas, DiffStats) {
	if a.Width != b.Width || a.Height != b.Height {
		panic("diff canvas size mismatch")
	}
	out := NewCanvas(a.Width, a.Height)
	stats := DiffStats{Width: a.Width, Height: a.Height, PixelCount: a.Width * a.Height, Gain: gain}

	for i := range a.Pixels {
		dr := math.Abs(a.Pixels[i].X - b.Pixels[i].X)
		dg := math.Abs(a.Pixels[i].Y - b.Pixels[i].Y)
		db := math.Abs(a.Pixels[i].Z - b.Pixels[i].Z)
		if dr > stats.MaxR {
			stats.MaxR = dr
		}
		if dg > stats.MaxG {
			stats.MaxG = dg
		}
		if db > stats.MaxB {
			stats.MaxB = db
		}
		luma := 0.2126*dr + 0.7152*dg + 0.0722*db
		if luma > stats.MaxLuma {
			stats.MaxLuma = luma
		}
		if dr > 1e-12 || dg > 1e-12 || db > 1e-12 {
			stats.DifferentPixels++
		}
		out.Pixels[i] = V(clamp01(dr*gain), clamp01(dg*gain), clamp01(db*gain))
	}
	stats.Identical = stats.DifferentPixels == 0
	return out, stats
}
