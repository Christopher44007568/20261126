package hdrlab

import "math"

// MeterCell is one evaluative-metering zone.
type MeterCell struct {
	Bounds     Rect
	MeanLuma   float64
	EVRelative float64 // EV relative to 18% middle gray
	ClipFrac   float64 // fraction of pixels at highlight clip
	ShadowFrac float64 // fraction of pixels in deep shadow
}

// Metering is a cols x rows evaluative metering readout.
type Metering struct {
	Cols, Rows     int
	Cells          []MeterCell
	CenterEV       float64
	ClipFraction   float64
	ShadowFraction float64
}

// ComputeMetering evaluates the reference (normal) frame on a cols x rows
// grid: mean luma, EV vs 18% gray, clipped and deep-shadow fractions.
func ComputeMetering(frame *Img, cols, rows int) *Metering {
	m := &Metering{Cols: cols, Rows: rows}
	cw := frame.W / cols
	ch := frame.H / rows
	var totClip, totShadow, totN int
	for ry := 0; ry < rows; ry++ {
		for cx := 0; cx < cols; cx++ {
			b := Rect{MinX: cx * cw, MinY: ry * ch}
			if cx == cols-1 {
				b.MaxX = frame.W
			} else {
				b.MaxX = b.MinX + cw
			}
			if ry == rows-1 {
				b.MaxY = frame.H
			} else {
				b.MaxY = b.MinY + ch
			}
			var sum float64
			var clip, shadow, n int
			for y := b.MinY; y < b.MaxY; y++ {
				for x := b.MinX; x < b.MaxX; x++ {
					l := frame.Luma(x, y)
					sum += l
					if l >= 0.98 {
						clip++
					}
					if l <= 0.02 {
						shadow++
					}
					n++
				}
			}
			mean := sum / float64(n)
			m.Cells = append(m.Cells, MeterCell{
				Bounds:     b,
				MeanLuma:   mean,
				EVRelative: math.Log2(mean / 0.18),
				ClipFrac:   float64(clip) / float64(n),
				ShadowFrac: float64(shadow) / float64(n),
			})
			totClip += clip
			totShadow += shadow
			totN += n
		}
	}
	// Center EV: mean of the 2x2 central cells.
	var csum float64
	for _, ry := range []int{rows/2 - 1, rows / 2} {
		for _, cx := range []int{cols/2 - 1, cols / 2} {
			csum += m.Cells[ry*cols+cx].EVRelative
		}
	}
	m.CenterEV = csum / 4
	m.ClipFraction = float64(totClip) / float64(totN)
	m.ShadowFraction = float64(totShadow) / float64(totN)
	return m
}
