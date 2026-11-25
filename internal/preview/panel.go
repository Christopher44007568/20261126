package preview

import (
	"fmt"

	"hdrexport/internal/colorx"
	"hdrexport/internal/font"
	"hdrexport/internal/hdr"
	"hdrexport/internal/pngx"
	"hdrexport/internal/raster"
)

const (
	panelW   = 300
	panelMgn = 14
	bgR      = uint8(18)
	bgG      = uint8(20)
	bgB      = uint8(26)
	fgR      = uint8(228)
	fgG      = uint8(230)
	fgB      = uint8(236)
	dimR     = uint8(150)
	dimG     = uint8(155)
	dimB     = uint8(165)
	accentR  = uint8(255)
	accentG  = uint8(196)
	accentB  = uint8(96)
)

// AnnotatedImage places the LDR preview next to the information panel.
func AnnotatedImage(ldr *raster.Image8, s hdr.Stats, exposure float64, peakCode uint8) *raster.Image8 {
	out := raster.NewImage8(ldr.W+panelW, ldr.H)
	out.FillRect(0, 0, out.W, out.H, bgR, bgG, bgB)
	out.Blit(0, 0, ldr)
	out.FillRect(ldr.W, 0, panelW, ldr.H, bgR+6, bgG+6, bgB+8)
	drawPanel(out, ldr.W, ldr.H, s, exposure, peakCode)
	return out
}

func drawPanel(out *raster.Image8, x0, h int, s hdr.Stats, exposure float64, peakCode uint8) {
	x := x0 + panelMgn
	y := 16
	line := font.LineHeight(1)

	font.DrawText(out, x, y, 1, "HDR EXPORT REPORT", accentR, accentG, accentB)
	y += line + 6

	rows := []struct{ k, v string }{
		{"HDR FORMAT", "PFM (FLOAT PPM, RGB F32)"},
		{"LDR FORMAT", "PNG 8BIT RGB, SRGB"},
		{"SCENE PEAK LUM", f3(s.MaxLum) + " (LINEAR)"},
		{"SDR WHITE", "1.000 (LINEAR)"},
		{"PEAK CHANNEL", f3(s.MaxChannel)},
		{"MEAN LUM", f3(s.MeanLum)},
		{"PIXELS OVER 1.0", fmt.Sprintf("%d / %d", s.OverWhite, s.TotalPx)},
		{"EXPOSURE K", f2(exposure)},
		{"MAX CODE VALUE", fmt.Sprintf("%d / 255", peakCode)},
	}
	for _, row := range rows {
		font.DrawText(out, x, y, 1, row.k, dimR, dimG, dimB)
		font.DrawText(out, x+128, y, 1, row.v, fgR, fgG, fgB)
		y += line
	}

	y += 4
	font.DrawText(out, x, y, 1, "LUMINANCE SCALE", accentR, accentG, accentB)
	y += line + 4
	y = drawLuminanceScale(out, x, y, panelW-2*panelMgn, s.MaxLum, exposure)

	y += 8
	font.DrawText(out, x, y, 1, "CONVERSION PIPELINE", accentR, accentG, accentB)
	y += line
	notes := []string{
		"1. LINEAR FLOAT HDR -> X*K",
		"2. REINHARD X/(1+X)",
		"3. SRGB OETF (GAMMA 2.4)",
		"4. QUANTIZE TO 8 BITS",
	}
	for _, n := range notes {
		font.DrawText(out, x, y, 1, n, fgR, fgG, fgB)
		y += line
	}

	y += 4
	font.DrawText(out, x, y, 1, "GAMMA SAFETY", accentR, accentG, accentB)
	y += line
	safety := []string{
		"PNG CARRIES SRGB/GAMA/CHRM",
		"METADATA CHUNKS, SO VIEWERS",
		"APPLY GAMMA EXACTLY ONCE.",
		"PFM IS RAW FLOAT: NO GAMMA,",
		"NO METADATA - TREAT AS LINEAR.",
	}
	for _, n := range safety {
		font.DrawText(out, x, y, 1, n, dimR, dimG, dimB)
		y += line
	}

	_ = h
}

// drawLuminanceScale paints a horizontal bar mapping linear luminance to the
// same tonemap+sRGB curve used on the image. Tick labels are linear values.
func drawLuminanceScale(out *raster.Image8, x, y, width int, peak, exposure float64) int {
	const barH = 16
	for i := 0; i < width; i++ {
		lin := peak * float64(i) / float64(width-1)
		c := colorx.SRGBEncode(colorx.Reinhard(lin, exposure))
		out.FillRect(x+i, y, 1, barH, c, c, c)
	}
	out.StrokeRect(x, y, width, barH, fgR, fgG, fgB)

	ticks := []float64{0, 0.5, 1.0, 2.0, 5.0, peak}
	var placed []int
	for _, t := range ticks {
		if t > peak {
			continue
		}
		tx := x + int(float64(width-1)*t/peak)
		for dy := 0; dy < 5; dy++ {
			out.Set(tx, y+barH+dy, fgR, fgG, fgB)
		}
		label := tickLabel(t, peak)
		lw := font.Measure(label, 1)
		lx := tx - lw/2
		if lx < x {
			lx = x
		}
		if lx+lw > x+width {
			lx = x + width - lw
		}
		// Collision avoidance: drop a label if it overlaps the previous one.
		if len(placed) > 0 && lx < placed[len(placed)-1]+6 {
			continue
		}
		font.DrawText(out, lx, y+barH+7, 1, label, dimR, dimG, dimB)
		placed = append(placed, lx)
	}

	// SDR white marker (linear 1.0).
	whiteX := x + int(float64(width-1)*1.0/peak)
	for dy := -2; dy < barH+2; dy++ {
		out.Set(whiteX, y+dy, accentR, accentG, accentB)
	}
	font.DrawText(out, x, y+barH+20, 1, "TICKS: LINEAR LUM  GOLD=SDR WHITE 1.0", dimR, dimG, dimB)
	return y + barH + 32
}

func tickLabel(t, peak float64) string {
	switch {
	case t == 0:
		return "0"
	case t == 1:
		return "1.0"
	case t >= peak:
		return fmt.Sprintf("%.1f", peak)
	default:
		return fmt.Sprintf("%g", t)
	}
}

// PNGNotes builds the tEXt chunks embedded in the PNG so the conversion
// provenance travels with the file itself.
func PNGNotes(s hdr.Stats, exposure float64, peakCode uint8) []pngx.TextEntry {
	return []pngx.TextEntry{
		{Keyword: "Title", Text: "HDR screenshot - sRGB 8 bit preview"},
		{Keyword: "Source", Text: "scene-linear float32 PFM, no gamma"},
		{Keyword: "ToneMap", Text: fmt.Sprintf("Reinhard x/(1+x), exposure %.3f", exposure)},
		{Keyword: "Transfer", Text: "IEC 61966-2-1 sRGB OETF applied once"},
		{Keyword: "PeakLuminance", Text: fmt.Sprintf("linear %.4f -> code %d/255", s.MaxLum, peakCode)},
		{Keyword: "OverWhitePixels", Text: fmt.Sprintf("%d of %d pixels exceed linear 1.0", s.OverWhite, s.TotalPx)},
	}
}
