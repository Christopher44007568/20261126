// Package reload verifies the PFM round trip: it decodes the written floating
// point file, re-renders the 8-bit preview, and builds a before/after
// comparison sheet including a diff view and a double-gamma demonstration.
package reload

import (
	"fmt"
	"math"

	"hdrexport/internal/colorx"
	"hdrexport/internal/font"
	"hdrexport/internal/hdr"
	"hdrexport/internal/preview"
	"hdrexport/internal/raster"
)

// Metrics summarizes round-trip fidelity between two float images.
type Metrics struct {
	MaxAbsErr  float64
	MeanAbsErr float64
	MaxRelErr  float64
	ByteDiffs  int
	MaxByteD   int
	Pixels     int
}

// Compare measures float round-trip error and re-rendered byte differences.
func Compare(before, after *hdr.Image, exposure float64) (Metrics, []byte, []byte) {
	if before.W != after.W || before.H != after.H {
		panic("reload: image dimensions differ")
	}
	var m Metrics
	m.Pixels = before.W * before.H
	beforeLDR := preview.BufFromHDR(before, exposure)
	afterLDR := preview.BufFromHDR(after, exposure)

	for i := 0; i < len(before.Pix); i++ {
		a, b := float64(before.Pix[i]), float64(after.Pix[i])
		d := math.Abs(a - b)
		if d > m.MaxAbsErr {
			m.MaxAbsErr = d
		}
		m.MeanAbsErr += d
		if a > 1e-6 {
			if r := d / a; r > m.MaxRelErr {
				m.MaxRelErr = r
			}
		}
	}
	m.MeanAbsErr /= float64(len(before.Pix))

	for i := 0; i < len(beforeLDR); i++ {
		d := int(beforeLDR[i]) - int(afterLDR[i])
		if d < 0 {
			d = -d
		}
		if d != 0 {
			m.ByteDiffs++
		}
		if d > m.MaxByteD {
			m.MaxByteD = d
		}
	}
	return m, beforeLDR, afterLDR
}

const (
	sheetPad = 12
	headerH  = 26
	labelH   = 18
	gammaH   = 44
	darkR    = uint8(16)
	darkG    = uint8(18)
	darkB    = uint8(24)
	whiteR   = uint8(232)
	whiteG   = uint8(234)
	whiteB   = uint8(240)
	goldR    = uint8(255)
	goldG    = uint8(200)
	goldB    = uint8(100)
)

// BuildSheet lays out before/reload/diff previews and the gamma demo strip.
func BuildSheet(beforeLDR, afterLDR []byte, w, h int, m Metrics) *raster.Image8 {
	imgH := h
	sheetW := 3*w + 4*sheetPad
	sheetH := headerH + labelH + imgH + gammaH + 3*sheetPad
	sheet := raster.NewImage8(sheetW, sheetH)
	sheet.FillRect(0, 0, sheetW, sheetH, darkR, darkG, darkB)

	font.DrawText(sheet, sheetPad, 8, 1,
		fmt.Sprintf("PFM RELOAD CHECK: MAX FLOAT ERR %.3g  MEAN %.3g  BYTE DIFFS %d (MAX %d)",
			m.MaxAbsErr, m.MeanAbsErr, m.ByteDiffs, m.MaxByteD),
		goldR, goldG, goldB)

	imgY := headerH + labelH
	x0 := sheetPad
	x1 := sheetPad*2 + w
	x2 := sheetPad*3 + 2*w

	font.DrawText(sheet, x0, headerH+4, 1, "BEFORE RELOAD (IN MEMORY HDR)", whiteR, whiteG, whiteB)
	font.DrawText(sheet, x1, headerH+4, 1, "AFTER RELOAD (PFM -> HDR)", whiteR, whiteG, whiteB)
	font.DrawText(sheet, x2, headerH+4, 1, "BYTE DIFFERENCE (X8 GAIN)", whiteR, whiteG, whiteB)

	sheet.PasteHDR(x0, imgY, beforeLDR, w, h)
	sheet.PasteHDR(x1, imgY, afterLDR, w, h)

	diff := raster.NewImage8(w, h)
	for i := 0; i < len(diff.Pix); i += 3 {
		d := int(beforeLDR[i]) - int(afterLDR[i])
		v := d * 8
		if v < 0 {
			diff.Pix[i] = 0
			diff.Pix[i+1] = 0
			diff.Pix[i+2] = uint8(clamp(-v))
		} else {
			diff.Pix[i] = uint8(clamp(v))
			diff.Pix[i+1] = uint8(clamp(v))
			diff.Pix[i+2] = 0
		}
	}
	sheet.Blit(x2, imgY, diff)

	drawGammaDemo(sheet, sheetPad, imgY+h+sheetPad, sheetW-2*sheetPad)
	return sheet
}

func clamp(v int) int {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return v
}

// drawGammaDemo shows what happens when software ignores the PNG metadata
// and applies the sRGB gamma a second time.
func drawGammaDemo(sheet *raster.Image8, x, y, width int) {
	font.DrawText(sheet, x, y, 1,
		"DOUBLE GAMMA HAZARD: LEFT = CORRECT SRGB (ONE OETF)  MIDDLE = GAMMA APPLIED TWICE  RIGHT = UNENCODED LINEAR SHOWN RAW",
		goldR, goldG, goldB)
	barY := y + 14
	barH := 22
	third := width / 3
	for i := 0; i < third; i++ {
		lin := float64(i) / float64(third-1)
		correct := colorx.SRGBEncode(lin)
		// Second gamma: re-encoding an already-encoded value.
		double := colorx.SRGBEncode(float64(correct) / 255.0)
		raw := uint8(clamp(int(lin * 255)))
		sheet.FillRect(x+i, barY, 1, barH, correct, correct, correct)
		sheet.FillRect(x+third+i, barY, 1, barH, double, double, double)
		sheet.FillRect(x+2*third+i, barY, 1, barH, raw, raw, raw)
	}
	sheet.StrokeRect(x, barY, third, barH, whiteR, whiteG, whiteB)
	sheet.StrokeRect(x+third, barY, third, barH, whiteR, whiteG, whiteB)
	sheet.StrokeRect(x+2*third, barY, third, barH, whiteR, whiteG, whiteB)
}
