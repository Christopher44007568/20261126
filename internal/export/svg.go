package export

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"aelab/internal/pipeline"
	"aelab/internal/scene"
)

const (
	svgW = 1040
	svgH = 560
	padL = 76
	padR = 28
	padT = 42
	padB = 62
)

func WriteExposureSVG(path string, result *pipeline.Result) error {
	minV, maxV := math.Inf(1), math.Inf(-1)
	for _, p := range result.Profiles {
		for _, r := range p.Records {
			minV = math.Min(minV, math.Min(r.ActualEV, r.TargetEV))
			maxV = math.Max(maxV, math.Max(r.ActualEV, r.TargetEV))
		}
	}
	minV, maxV = paddedRange(minV, maxV, 0.12)
	var b strings.Builder
	writeSVGHeader(&b, "Auto exposure temporal responses — exposure EV")
	writePlotBackground(&b, minV, maxV, "EV (log2 gain)")
	writeTargetLine(&b, result, minV, maxV)
	for _, p := range result.Profiles {
		writeRecordLine(&b, p.Records, minV, maxV, func(r pipeline.FrameRecord) float64 { return r.ActualEV }, p.Profile.Color, 3.0)
		writeLegend(&b, p.Profile.Color, p.Profile.Label, 150+float64(indexOf(result, p.Profile.ID))*270, 24)
	}
	writeSVGTimeAxis(&b)
	writeSVGFooter(&b)
	return writeFile(path, b.String())
}

func WriteOvershootSVG(path string, result *pipeline.Result) error {
	minV, maxV := math.Inf(1), math.Inf(-1)
	for _, p := range result.Profiles {
		for _, r := range p.Records {
			minV = math.Min(minV, r.OvershootEV)
			maxV = math.Max(maxV, r.OvershootEV)
		}
	}
	limit := math.Max(math.Abs(minV), math.Abs(maxV)) * 1.12
	minV, maxV = -limit, limit
	var b strings.Builder
	writeSVGHeader(&b, "Overshoot / convergence curve — final target minus actual EV")
	writePlotBackground(&b, minV, maxV, "signed overshoot EV")
	writeZeroLine(&b, minV, maxV)
	for _, p := range result.Profiles {
		writeRecordLine(&b, p.Records, minV, maxV, func(r pipeline.FrameRecord) float64 { return r.OvershootEV }, p.Profile.Color, 3.0)
		writeLegend(&b, p.Profile.Color, p.Profile.Label, 150+float64(indexOf(result, p.Profile.ID))*270, 24)
	}
	writeSVGTimeAxis(&b)
	writeSVGFooter(&b)
	return writeFile(path, b.String())
}

func writeSVGHeader(b *strings.Builder, title string) {
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1040" height="560" viewBox="0 0 1040 560" font-family="Inter, Arial, sans-serif">`)
	b.WriteString(`<rect width="1040" height="560" fill="#fbfaf7"/>`)
	fmt.Fprintf(b, `<text x="34" y="27" font-size="20" font-weight="700" fill="#202124">%s</text>`, title)
}

func writePlotBackground(b *strings.Builder, minV, maxV float64, yLabel string) {
	plotW := svgW - padL - padR
	plotH := svgH - padT - padB
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="#ffffff" stroke="#d8d4cc"/>`, padL, padT, plotW, plotH)
	for i := 0; i <= 8; i++ {
		y := float64(padT) + float64(plotH)*float64(i)/8
		value := maxV - (maxV-minV)*float64(i)/8
		fmt.Fprintf(b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#ece8df"/>`, padL, y, svgW-padR, y)
		fmt.Fprintf(b, `<text x="%d" y="%.1f" text-anchor="end" font-size="12" fill="#5f6368">%+.2f</text>`, padL-10, y+4, value)
	}
	for t := 0; t <= 10; t++ {
		x := xForTime(float64(t))
		fmt.Fprintf(b, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#f0ece3"/>`, x, padT, x, svgH-padB)
		fmt.Fprintf(b, `<text x="%.1f" y="%d" text-anchor="middle" font-size="12" fill="#5f6368">%ds</text>`, x, svgH-padB+22, t)
	}
	turnX := xForTime(pipeline.TransitionStart)
	endX := xForTime(pipeline.TransitionEnd)
	fmt.Fprintf(b, `<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="#f4c95d" opacity="0.13"/>`, turnX, padT, endX-turnX, plotH)
	fmt.Fprintf(b, `<text x="18" y="%d" transform="rotate(-90 18 %d)" text-anchor="middle" font-size="13" fill="#3c4043">%s</text>`, padT+plotH/2, padT+plotH/2, yLabel)
}

func writeTargetLine(b *strings.Builder, result *pipeline.Result, minV, maxV float64) {
	records := result.Profiles[0].Records
	points := make([][2]float64, 0, len(records))
	for _, r := range records {
		points = append(points, [2]float64{xForTime(r.TimeS), yForValue(r.TargetEV, minV, maxV)})
	}
	writePolyline(b, points, "#7a7a7a", 2.0, "6 5")
	fmt.Fprintf(b, `<text x="%d" y="%d" font-size="12" fill="#666">dashed = metered target EV</text>`, padL+12, padT+20)
}

func writeZeroLine(b *strings.Builder, minV, maxV float64) {
	y := yForValue(0, minV, maxV)
	fmt.Fprintf(b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#777" stroke-width="1.4"/>`, padL, y, svgW-padR, y)
}

func writeRecordLine(b *strings.Builder, records []pipeline.FrameRecord, minV, maxV float64, value func(pipeline.FrameRecord) float64, color string, width float64) {
	points := make([][2]float64, 0, len(records))
	for _, r := range records {
		points = append(points, [2]float64{xForTime(r.TimeS), yForValue(value(r), minV, maxV)})
	}
	writePolyline(b, points, color, width, "")
}

func writePolyline(b *strings.Builder, points [][2]float64, color string, width float64, dash string) {
	parts := make([]string, 0, len(points))
	for i, p := range points {
		if i == 0 {
			parts = append(parts, fmt.Sprintf("M%.1f %.1f", p[0], p[1]))
		} else {
			parts = append(parts, fmt.Sprintf("L%.1f %.1f", p[0], p[1]))
		}
	}
	dashAttr := ""
	if dash != "" {
		dashAttr = ` stroke-dasharray="` + dash + `"`
	}
	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-linejoin="round" stroke-linecap="round"%s/>`, strings.Join(parts, " "), color, width, dashAttr)
}

func writeLegend(b *strings.Builder, color, label string, x, y float64) {
	fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="4" stroke-linecap="round"/>`, x, y-5, x+30, y-5, color)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="14" fill="#202124">%s</text>`, x+38, y, label)
}

func writeSVGTimeAxis(b *strings.Builder) {
	fmt.Fprintf(b, `<text x="%d" y="%d" text-anchor="middle" font-size="14" fill="#3c4043">time (seconds); yellow band = camera pan dark interior → window</text>`, padL+(svgW-padL-padR)/2, svgH-18)
}

func writeSVGFooter(b *strings.Builder) {
	b.WriteString(`</svg>`)
}

func xForTime(t float64) float64 {
	plotW := svgW - padL - padR
	return float64(padL) + t/scene.Duration*float64(plotW)
}

func yForValue(v, minV, maxV float64) float64 {
	plotH := svgH - padT - padB
	return float64(padT) + (maxV-v)/(maxV-minV)*float64(plotH)
}

func paddedRange(minV, maxV, pad float64) (float64, float64) {
	span := maxV - minV
	if span < 0.001 {
		span = 1
	}
	return minV - span*pad, maxV + span*pad
}

func indexOf(result *pipeline.Result, id string) int {
	for i, p := range result.Profiles {
		if p.Profile.ID == id {
			return i
		}
	}
	return 0
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
