package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
)

var (
	chartBG    = color.RGBA{R: 9, G: 13, B: 19, A: 255}
	panelBG    = color.RGBA{R: 17, G: 23, B: 32, A: 255}
	panelEdge  = color.RGBA{R: 38, G: 49, B: 64, A: 255}
	textBright = color.RGBA{R: 234, G: 240, B: 246, A: 255}
	textMuted  = color.RGBA{R: 143, G: 160, B: 180, A: 255}
	textDim    = color.RGBA{R: 98, G: 113, B: 132, A: 255}
	passInk    = color.RGBA{R: 72, G: 220, B: 120, A: 255}
	failInk    = color.RGBA{R: 255, G: 92, B: 122, A: 255}
	warnInk    = color.RGBA{R: 255, G: 184, B: 62, A: 255}
)

type panelGeometry struct {
	imageRect image.Rectangle
	canvasW   int
	canvasH   int
}

func renderChart(m *Manifest, a *Analysis, amplification int) *image.RGBA {
	srcB := a.Source.Bounds()
	imgW, imgH := fitSize(srcB.Dx(), srcB.Dy(), 360, 278)

	const (
		margin      = 30
		gap         = 20
		headerH     = 132
		panelTitleH = 30
		captionH    = 28
		footerH     = 112
	)
	panelW := maxInt(imgW, 320)
	chartW := margin*2 + panelW*3 + gap*2
	panelTop := headerH + 26
	imgTop := panelTop + panelTitleH
	footerTop := imgTop + imgH + captionH + 22
	chartH := footerTop + footerH + margin

	chart := image.NewRGBA(image.Rect(0, 0, chartW, chartH))
	fillRect(chart, chart.Bounds(), chartBG)
	fillRect(chart, image.Rect(0, 0, chartW, 4), color.RGBA{R: 68, G: 224, B: 180, A: 255})

	drawHeader(chart, m, a)
	errorImage := buildErrorImage(a.Source, a.Target, amplification)
	images := []*image.RGBA{
		scaleImage(a.Source, imgW, imgH),
		scaleImage(a.Target, imgW, imgH),
		scaleImage(errorImage, imgW, imgH),
	}
	titles := []string{"01 / ORIGINAL", "02 / TARGET", "03 / ERROR AMP X" + fmt.Sprint(amplification)}
	captions := []string{"FULL SOURCE FRAME", "LOCAL FAILURES STAY VISIBLE", "BRIGHT = ERROR / COLOR = THRESHOLD"}
	geoms := make([]panelGeometry, 3)
	for i := 0; i < 3; i++ {
		x := margin + i*(panelW+gap)
		fillRect(chart, image.Rect(x, panelTop, x+panelW, imgTop+imgH+captionH), panelBG)
		drawText(chart, x+14, panelTop+9, titles[i], textBright, 2)
		imgX := x + (panelW-imgW)/2
		imgRect := image.Rect(imgX, imgTop, imgX+imgW, imgTop+imgH)
		drawRect(chart, imgRect, color.RGBA{R: 3, G: 5, B: 8, A: 255})
		drawImageInto(chart, imgRect.Min, images[i])
		drawText(chart, x+14, imgTop+imgH+8, captions[i], textMuted, 1)
		geoms[i] = panelGeometry{imageRect: imgRect, canvasW: m.Canvas.Width, canvasH: m.Canvas.Height}
	}
	drawPatchFrames(chart, geoms[0], a.Patches, false)
	drawPatchFrames(chart, geoms[1], a.Patches, true)
	drawPatchFrames(chart, geoms[2], a.Patches, true)
	drawFooter(chart, footerTop, chart.Bounds().Max.X, a)
	return chart
}

func drawHeader(dst *image.RGBA, m *Manifest, a *Analysis) {
	left := 30
	drawText(dst, left, 24, truncateText(m.Title, dst.Bounds().Dx()-330, 3), textBright, 3)

	subtitle := "LOCAL PATCH VERDICT WITH RETAINED SOURCE + TARGET"
	if a.Summary.AveragePass && !a.Summary.LocalPass {
		subtitle = "AVERAGE PASS, LOCAL FAILURE - BOTH FRAMES RETAINED"
	}
	subtitleColor := textMuted
	if a.Summary.AveragePass && !a.Summary.LocalPass {
		subtitleColor = warnInk
	}
	drawText(dst, left, 55, subtitle, subtitleColor, 2)

	drawText(dst, left, 86, fmt.Sprintf("MEAN DE %.2f / LIMIT %.2f", a.Summary.MeanDeltaE, m.Thresholds.DeltaE), textMuted, 2)
	drawText(dst, left, 104, fmt.Sprintf("AVG DL %.2f / LIMIT %.2f   PATCHES %d/%d PASS",
		a.Summary.MeanDeltaL, m.Thresholds.Brightness, a.Summary.Passed, a.Summary.PatchCount), textMuted, 2)

	boxW := 190
	box := image.Rect(dst.Bounds().Dx()-30-boxW, 24, dst.Bounds().Dx()-30, 112)
	fillRect(dst, box, color.RGBA{R: 22, G: 29, B: 39, A: 255})
	strokeRect(dst, box, panelEdge, 1)
	drawText(dst, box.Min.X+14, box.Min.Y+12, "RESULT", textDim, 2)
	result := "PASS"
	ink := passInk
	if !a.Summary.LocalPass {
		result = "FAIL"
		ink = failInk
	}
	drawText(dst, box.Min.X+14, box.Min.Y+34, result, ink, 3)
	avg := "AVG PASS"
	avgInk := passInk
	if !a.Summary.AveragePass {
		avg = "AVG FAIL"
		avgInk = failInk
	}
	drawText(dst, box.Min.X+14, box.Min.Y+72, avg, avgInk, 1)
	drawTextRight(dst, box.Max.X-14, box.Min.Y+72, fmt.Sprintf("LOCAL FAIL %d", a.Summary.Failed), failInk, 1)
}

func drawFooter(dst *image.RGBA, top, right int, a *Analysis) {
	drawLine(dst, 30, top-18, right-30, top-18, panelEdge)
	drawText(dst, 30, top, "THRESHOLD OVERLAY", textBright, 2)
	x := 30
	for _, item := range []struct {
		c     color.RGBA
		label string
	}{
		{failureCategoryColor("delta_e"), "DELTA E / BRIGHTNESS"},
		{failureCategoryColor("gamut_clip"), "GAMUT CLIP"},
		{failureCategoryColor("monotonicity"), "MONOTONICITY"},
		{passInk, "PASS"},
	} {
		fillRect(dst, image.Rect(x, top+27, x+14, top+41), item.c)
		drawText(dst, x+20, top+29, item.label, textMuted, 2)
		x += 30 + textWidth(item.label, 2)
	}
	drawText(dst, 30, top+54, "FAILED PATCHES", textDim, 1)
	failed := 0
	for _, p := range a.Patches {
		if p.Pass {
			continue
		}
		if failed >= 4 {
			break
		}
		drawText(dst, 30+failed*280, top+68, patchFailureSummary(p), failInk, 2)
		failed++
	}
	if failed == 0 {
		drawText(dst, 30, top+68, "NONE - ALL LOCAL RULES PASSED", passInk, 2)
	}
	drawText(dst, 30, top+96, "SOURCE AND TARGET ARE NEVER REPLACED BY A SUMMARY TABLE ALONE.", textMuted, 1)
}

func patchFailureSummary(p PatchResult) string {
	codes := make([]string, 0, len(p.Failures))
	for _, f := range p.Failures {
		label := map[string]string{
			"delta_e":      "DE",
			"brightness":   "DL",
			"gamut_clip":   "CLIP",
			"monotonicity": "ORDER",
		}[f.Code]
		if label == "" {
			label = strings.ToUpper(f.Code)
		}
		codes = append(codes, label)
	}
	return strings.ToUpper(p.ID) + ": " + strings.Join(codes, ", ")
}

func drawPatchFrames(dst *image.RGBA, geom panelGeometry, patches []PatchResult, showFailure bool) {
	for _, p := range patches {
		r := mapPatchRect(geom, p.Rect)
		base := color.RGBA{R: 49, G: 62, B: 80, A: 255}
		if showFailure && !p.Pass {
			categories := patchFailureCategories(p)
			primary := failureCategoryColor(categories[0])
			blendRect(dst, r, primary, 0.22)
			hatchRect(dst, r, primary, 0.70, 9)
			strokeRect(dst, r, primary, 3)
			for i, code := range categories[1:] {
				strokeRect(dst, r.Inset(4+i*4), failureCategoryColor(code), 2)
			}
			drawPatchTag(dst, r, p.ID, primary, categories)
		} else {
			strokeRect(dst, r, base, 1)
		}
	}
}

func drawPatchTag(dst *image.RGBA, r image.Rectangle, id string, ink color.RGBA, categories []string) {
	label := strings.ToUpper(id)
	w := textWidth(label, 1) + 8 + len(categories)*8
	h := 14
	x, y := r.Min.X+4, r.Min.Y+4
	if x+w > dst.Bounds().Max.X {
		x = dst.Bounds().Max.X - w
	}
	fillRect(dst, image.Rect(x, y, x+w, y+h), color.RGBA{R: 5, G: 8, B: 12, A: 255})
	strokeRect(dst, image.Rect(x, y, x+w, y+h), ink, 1)
	drawText(dst, x+4, y+4, label, ink, 1)
	chipX := x + 4 + textWidth(label, 1) + 2
	for _, code := range categories {
		fillRect(dst, image.Rect(chipX, y+4, chipX+6, y+10), failureCategoryColor(code))
		chipX += 8
	}
}

func patchFailureCategories(p PatchResult) []string {
	seen := make(map[string]bool)
	var categories []string
	for _, f := range p.Failures {
		code := f.Code
		if code == "brightness" {
			code = "delta_e"
		}
		if seen[code] {
			continue
		}
		seen[code] = true
		categories = append(categories, code)
	}
	if len(categories) == 0 {
		categories = append(categories, "pass")
	}
	return categories
}

func mapPatchRect(geom panelGeometry, p RectSpec) image.Rectangle {
	r := geom.imageRect
	sx := float64(r.Dx()) / float64(geom.canvasW)
	sy := float64(r.Dy()) / float64(geom.canvasH)
	return image.Rect(
		r.Min.X+int(math.Round(float64(p.X)*sx)),
		r.Min.Y+int(math.Round(float64(p.Y)*sy)),
		r.Min.X+int(math.Round(float64(p.X+p.W)*sx)),
		r.Min.Y+int(math.Round(float64(p.Y+p.H)*sy)),
	)
}

func buildErrorImage(source, target image.Image, amplification int) *image.RGBA {
	b := source.Bounds().Intersect(target.Bounds())
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	drawRect(out, out.Bounds(), color.RGBA{R: 6, G: 9, B: 13, A: 255})
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			s := rgbFromColor(source.At(x, y))
			t := rgbFromColor(target.At(x, y))
			d := maxInt(absInt(int(t.R)-int(s.R)), absInt(int(t.G)-int(s.G)), absInt(int(t.B)-int(s.B)))
			v := d * amplification
			if v > 255 {
				v = 255
			}
			g := uint8(v)
			out.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{R: g, G: g, B: g, A: 255})
		}
	}
	return out
}

func fitSize(w, h, maxW, maxH int) (int, int) {
	if w <= 0 || h <= 0 {
		return 1, 1
	}
	scale := math.Min(float64(maxW)/float64(w), float64(maxH)/float64(h))
	if scale > 1 {
		scale = 1
	}
	return maxInt(1, int(math.Round(float64(w)*scale))), maxInt(1, int(math.Round(float64(h)*scale)))
}

func scaleImage(src image.Image, w, h int) *image.RGBA {
	if src.Bounds().Dx() == w && src.Bounds().Dy() == h {
		return flattenImage(src, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)*float64(sb.Dy())/float64(h) - 0.5
		y0, y1, wy := blendCoords(fy, sb.Min.Y, sb.Max.Y)
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*float64(sb.Dx())/float64(w) - 0.5
			x0, x1, wx := blendCoords(fx, sb.Min.X, sb.Max.X)
			c00 := rgbFromColor(src.At(x0, y0))
			c10 := rgbFromColor(src.At(x1, y0))
			c01 := rgbFromColor(src.At(x0, y1))
			c11 := rgbFromColor(src.At(x1, y1))
			topR := float64(c00.R)*(1-wx) + float64(c10.R)*wx
			topG := float64(c00.G)*(1-wx) + float64(c10.G)*wx
			topB := float64(c00.B)*(1-wx) + float64(c10.B)*wx
			botR := float64(c01.R)*(1-wx) + float64(c11.R)*wx
			botG := float64(c01.G)*(1-wx) + float64(c11.G)*wx
			botB := float64(c01.B)*(1-wx) + float64(c11.B)*wx
			out.SetRGBA(x, y, color.RGBA{
				R: clampByte(topR*(1-wy) + botR*wy),
				G: clampByte(topG*(1-wy) + botG*wy),
				B: clampByte(topB*(1-wy) + botB*wy),
				A: 255,
			})
		}
	}
	return out
}

func blendCoords(v float64, min, max int) (int, int, float64) {
	i := int(math.Floor(v))
	w := v - float64(i)
	i0 := maxInt(min, minInt(i, max-1))
	i1 := maxInt(min, minInt(i+1, max-1))
	return i0, i1, w
}

func drawImageInto(dst *image.RGBA, at image.Point, src image.Image) {
	sb := src.Bounds()
	for y := 0; y < sb.Dy(); y++ {
		for x := 0; x < sb.Dx(); x++ {
			dst.Set(x+at.X, y+at.Y, src.At(sb.Min.X+x, sb.Min.Y+y))
		}
	}
}

func blendRect(dst *image.RGBA, r image.Rectangle, c color.RGBA, alpha float64) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			blendPixel(dst, x, y, c, alpha)
		}
	}
}

func hatchRect(dst *image.RGBA, r image.Rectangle, c color.RGBA, alpha float64, period int) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if (x+y)%period < 2 {
				blendPixel(dst, x, y, c, alpha)
			}
		}
	}
}

func blendPixel(dst *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	base := dst.RGBAAt(x, y)
	inv := 1 - alpha
	dst.SetRGBA(x, y, color.RGBA{
		R: clampByte(float64(base.R)*inv + float64(c.R)*alpha),
		G: clampByte(float64(base.G)*inv + float64(c.G)*alpha),
		B: clampByte(float64(base.B)*inv + float64(c.B)*alpha),
		A: 255,
	})
}

func failureCategoryColor(code string) color.RGBA {
	switch code {
	case "delta_e", "brightness":
		return color.RGBA{R: 255, G: 45, B: 141, A: 255}
	case "gamut_clip":
		return color.RGBA{R: 39, G: 199, B: 255, A: 255}
	case "monotonicity":
		return color.RGBA{R: 255, G: 176, B: 32, A: 255}
	default:
		return passInk
	}
}

func truncateText(text string, maxWidth, scale int) string {
	if textWidth(text, scale) <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 3 {
		runes = runes[:len(runes)-1]
		if textWidth(string(runes)+"...", scale) <= maxWidth {
			break
		}
	}
	return string(runes) + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
