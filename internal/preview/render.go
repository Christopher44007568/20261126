// Package preview converts the float HDR image into an 8-bit sRGB buffer and
// annotates it with a luminance scale, code values and conversion notes.
package preview

import (
	"fmt"

	"hdrexport/internal/colorx"
	"hdrexport/internal/hdr"
	"hdrexport/internal/raster"
)

// RenderLDR applies Reinhard + sRGB encoding to every pixel.
func RenderLDR(m *hdr.Image, exposure float64) *raster.Image8 {
	out := raster.NewImage8(m.W, m.H)
	for i := 0; i < len(m.Pix); i += 3 {
		r, g, b := colorx.TonemapTriplet(
			float64(m.Pix[i]), float64(m.Pix[i+1]), float64(m.Pix[i+2]), exposure)
		out.Pix[i], out.Pix[i+1], out.Pix[i+2] = r, g, b
	}
	return out
}

// MaxCodeOfScene reports the byte produced for the scene luminance peak.
func MaxCodeOfScene(s hdr.Stats, exposure float64) uint8 {
	return colorx.MaxCodeValue(s.MaxLum, exposure)
}

// BufFromHDR renders into a raw RGB byte slice (for reuse on comparisons).
func BufFromHDR(m *hdr.Image, exposure float64) []byte {
	buf := make([]byte, len(m.Pix))
	for i := 0; i < len(m.Pix); i += 3 {
		r, g, b := colorx.TonemapTriplet(
			float64(m.Pix[i]), float64(m.Pix[i+1]), float64(m.Pix[i+2]), exposure)
		buf[i], buf[i+1], buf[i+2] = r, g, b
	}
	return buf
}

func f2(v float64) string { return fmt.Sprintf("%.2f", v) }
func f3(v float64) string { return fmt.Sprintf("%.3f", v) }
