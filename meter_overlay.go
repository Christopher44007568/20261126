package hdrlab

import "fmt"

// evTint maps a cell's relative EV to a zone color: blue = under-exposed,
// green = near 18% gray, orange = hot.
func evTint(ev float64) (r, g, b float64) {
	switch {
	case ev < -1:
		return 0.2, 0.45, 1.0
	case ev > 1:
		return 1.0, 0.55, 0.1
	default:
		return 0.1, 0.9, 0.35
	}
}

// BuildMeteringImage overlays the evaluative metering grid on the HDR
// preview: per-cell EV tint, hatch on clipped cells, center-cell marker.
func BuildMeteringImage(preview *Img, m *Metering) *Img {
	out := preview.Clone()
	for idx, c := range m.Cells {
		b := Rect{
			MinX: c.Bounds.MinX - Margin,
			MinY: c.Bounds.MinY - Margin,
			MaxX: c.Bounds.MaxX - Margin,
			MaxY: c.Bounds.MaxY - Margin,
		}
		r, g, bb := evTint(c.EVRelative)
		out.BlendRect(b.MinX, b.MinY, b.MaxX, b.MaxY, r, g, bb, 0.20)
		if c.ClipFrac > 0.5 {
			out.Hatch(b.MinX, b.MinY, b.MaxX, b.MaxY, 6, 0.9, 0.15, 0.15, 0.5)
		}
		col := idx % m.Cols
		row := idx / m.Cols
		if (row == m.Rows/2-1 || row == m.Rows/2) && (col == m.Cols/2-1 || col == m.Cols/2) {
			out.Rect(b.MinX, b.MinY, b.MaxX, b.MaxY, 1, 1, 1, 0.35)
		}
	}
	drawMeterGrid(out, m, 0.55)
	out.DrawTextBox(4, 4, fmt.Sprintf("EVAL METER %dX%d", m.Cols, m.Rows), 1, 1, 1)
	out.DrawTextBox(4, 16, fmt.Sprintf("CENTER EV %+.1f GREEN=OK BLUE=DARK ORANGE=BRIGHT", m.CenterEV), 1, 1, 1)
	out.DrawTextBox(4, OutH-12, fmt.Sprintf("CLIP %.1f%%", m.ClipFraction*100), 1, 0.5, 0.4)
	out.DrawTextBox(110, OutH-12, fmt.Sprintf("SHADOW %.1f%%", m.ShadowFraction*100), 0.5, 0.7, 1)
	return out
}

func drawMeterGrid(out *Img, m *Metering, a float64) {
	for _, c := range m.Cells {
		x := c.Bounds.MinX - Margin
		y := c.Bounds.MinY - Margin
		out.Line(x, 0, x, OutH-1, 0.1, 0.9, 0.4, a)
		out.Line(0, y, OutW-1, y, 0.1, 0.9, 0.4, a)
	}
	out.Line(OutW-1, 0, OutW-1, OutH-1, 0.1, 0.9, 0.4, a)
	out.Line(0, OutH-1, OutW-1, OutH-1, 0.1, 0.9, 0.4, a)
}

// BuildOverlayImage combines the HDR preview with the metering grid, the
// alignment-error heat map, and ghost-cluster outlines.
func BuildOverlayImage(preview, resid *Img, clusters []Cluster, m *Metering, scale float64) *Img {
	out := preview.Clone()
	for oy := 0; oy < OutH; oy++ {
		for ox := 0; ox < OutW; ox++ {
			t := resid.Pix[(oy*OutW+ox)*3] / scale
			if t > 0.06 {
				r, g, b := Jet(t)
				out.Blend(ox, oy, r, g, b, 0.42)
			}
		}
	}
	drawMeterGrid(out, m, 0.4)
	drawClusterEllipses(out, clusters, 1, 0.25, 0.7, 0.95)
	out.DrawTextBox(4, 4, "HDR: METER + ALIGN ERROR", 1, 1, 1)
	out.DrawTextBox(4, 16, "GREEN METER / HEAT ERROR / PINK GHOST", 1, 1, 1)
	return out
}
