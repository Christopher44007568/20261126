package export

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"aelab/internal/pipeline"
)

type pngChart struct {
	img                      *image.RGBA
	w, h                     int
	left, top, right, bottom int
	minV, maxV               float64
}

func WriteExposurePNG(path string, result *pipeline.Result) error {
	minV, maxV := math.Inf(1), math.Inf(-1)
	for _, p := range result.Profiles {
		for _, r := range p.Records {
			minV = math.Min(minV, math.Min(r.ActualEV, r.TargetEV))
			maxV = math.Max(maxV, math.Max(r.ActualEV, r.TargetEV))
		}
	}
	minV, maxV = paddedRange(minV, maxV, 0.10)
	chart := newPNGChart(1200, 620, minV, maxV)
	chart.background()
	drawPNGSeries(chart, result.Profiles[0].Records, func(r pipeline.FrameRecord) float64 { return r.TargetEV }, color.RGBA{R: 122, G: 122, B: 122, A: 255}, false)
	for i, p := range result.Profiles {
		drawPNGSeries(chart, p.Records, func(r pipeline.FrameRecord) float64 { return r.ActualEV }, hexColor(p.Profile.Color), true)
		drawLegendPNG(chart.img, 650+i*180, 24, hexColor(p.Profile.Color), legendName(p.Profile.ID))
	}
	chart.axes("EXPOSURE EV  GRAY LINE IS METER TARGET")
	return savePNG(path, chart.img)
}

func WriteOvershootPNG(path string, result *pipeline.Result) error {
	limit := 0.0
	for _, p := range result.Profiles {
		for _, r := range p.Records {
			limit = math.Max(limit, math.Abs(r.OvershootEV))
		}
	}
	limit *= 1.12
	chart := newPNGChart(1200, 620, -limit, limit)
	chart.background()
	y0 := chart.y(0)
	for x := chart.left; x < chart.w-chart.right; x++ {
		chart.img.Set(x, y0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	}
	for i, p := range result.Profiles {
		drawPNGSeries(chart, p.Records, func(r pipeline.FrameRecord) float64 { return r.OvershootEV }, hexColor(p.Profile.Color), true)
		drawLegendPNG(chart.img, 650+i*180, 24, hexColor(p.Profile.Color), legendName(p.Profile.ID))
	}
	chart.axes("OVERSHOOT EV  POSITIVE MEANS TOO DARK")
	return savePNG(path, chart.img)
}

func newPNGChart(w, h int, minV, maxV float64) *pngChart {
	return &pngChart{img: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h, left: 76, top: 58, right: 48, bottom: 64, minV: minV, maxV: maxV}
}

func (c *pngChart) background() {
	fillRect(c.img, 0, 0, c.w, c.h, color.RGBA{R: 251, G: 250, B: 247, A: 255})
	plot := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	fillRect(c.img, c.left, c.top, c.w-c.right, c.h-c.bottom, plot)
	turnX0 := c.x(0.80)
	turnX1 := c.x(pipeline.TransitionEnd)
	fillRect(c.img, turnX0, c.top, turnX1, c.h-c.bottom, color.RGBA{R: 250, G: 238, B: 205, A: 255})
	grid := color.RGBA{R: 232, G: 228, B: 218, A: 255}
	for i := 0; i <= 8; i++ {
		y := c.top + (c.h-c.top-c.bottom)*i/8
		hLine(c.img, c.left, c.w-c.right, y, grid)
	}
	for t := 0; t <= 10; t++ {
		x := c.x(float64(t))
		vLine(c.img, x, c.top, c.h-c.bottom, grid)
	}
}

func (c *pngChart) axes(title string) {
	axis := color.RGBA{R: 80, G: 80, B: 80, A: 255}
	hLine(c.img, c.left, c.w-c.right, c.h-c.bottom, axis)
	vLine(c.img, c.left, c.top, c.h-c.bottom, axis)
	for i := 0; i <= 4; i++ {
		value := c.maxV - (c.maxV-c.minV)*float64(i)/4
		y := c.top + (c.h-c.top-c.bottom)*i/4
		label := fmt.Sprintf("%.1f", value)
		drawTinyText(c.img, c.left-14-len(label)*6, y-4, label, axis, 1)
		hLine(c.img, c.left-5, c.left, y, axis)
	}
	drawTinyText(c.img, c.left, 24, title, color.RGBA{R: 32, G: 33, B: 36, A: 255}, 2)
	for t := 0; t <= 10; t += 2 {
		x := c.x(float64(t))
		label := itoa(t)
		drawTinyText(c.img, x-3*len(label), c.h-c.bottom+12, label, axis, 1)
	}
	drawTinyText(c.img, c.left+(c.w-c.left-c.right)/2-70, c.h-24, "time seconds", axis, 1)
}

func legendName(id string) string {
	switch id {
	case "rapid_breathing":
		return "RAPID"
	case "fast_smooth":
		return "SMOOTH"
	default:
		return "SLOW"
	}
}

func (c *pngChart) x(t float64) int {
	plotW := c.w - c.left - c.right
	return c.left + int(t/10.0*float64(plotW))
}

func (c *pngChart) y(v float64) int {
	plotH := c.h - c.top - c.bottom
	return c.top + int((c.maxV-v)/(c.maxV-c.minV)*float64(plotH))
}

func drawPNGSeries(c *pngChart, records []pipeline.FrameRecord, value func(pipeline.FrameRecord) float64, col color.RGBA, thick bool) {
	for i := 1; i < len(records); i++ {
		x0, y0 := c.x(records[i-1].TimeS), c.y(value(records[i-1]))
		x1, y1 := c.x(records[i].TimeS), c.y(value(records[i]))
		drawLinePNG(c.img, x0, y0, x1, y1, col)
		if thick {
			drawLinePNG(c.img, x0, y0+1, x1, y1+1, col)
		}
	}
}

func savePNG(path string, img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func hexColor(s string) color.RGBA {
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{R: 80, G: 80, B: 80, A: 255}
	}
	var r, g, b uint8
	parseHexByte(s[1:3], &r)
	parseHexByte(s[3:5], &g)
	parseHexByte(s[5:7], &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func parseHexByte(s string, out *uint8) {
	var v uint8
	for i := 0; i < 2; i++ {
		ch := s[i]
		n := uint8(0)
		switch {
		case ch >= '0' && ch <= '9':
			n = ch - '0'
		case ch >= 'a' && ch <= 'f':
			n = ch - 'a' + 10
		case ch >= 'A' && ch <= 'F':
			n = ch - 'A' + 10
		}
		v = v*16 + n
	}
	*out = v
}
