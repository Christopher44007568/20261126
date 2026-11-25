package mipmap

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
)

var chartLineColors = map[string]color.RGBA{
	"wrap_normalized":     {R: 72, G: 226, B: 158, A: 255},
	"naive_zero_pad":      {R: 255, G: 92, B: 92, A: 255},
	"inside_renormalized": {R: 255, G: 196, B: 72, A: 255},
}

func WriteErrorChart(path string, report Report) error {
	const width = 980
	const height = 620
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fillBackground(dst, color.RGBA{R: 18, G: 21, B: 27, A: 255})

	plot := image.Rect(82, 86, width-42, height-92)
	DrawText(dst, "AVERAGE COLOR ERROR", 42, 30, 3, textColor)
	DrawText(dst, "LOG10 RMS VS BASE MEAN", 44, 62, 2, mutedColor)
	drawChartFrame(dst, plot)

	methods := []string{"naive_zero_pad", "inside_renormalized", "wrap_normalized"}
	maxLevel := 1
	for _, metric := range report.ColorMetrics {
		if metric.Level > maxLevel {
			maxLevel = metric.Level
		}
	}
	drawLogErrorAxes(dst, plot, maxLevel)
	for _, method := range methods {
		points := make([]image.Point, 0, maxLevel+1)
		for _, metric := range report.ColorMetrics {
			if metric.Method != method {
				continue
			}
			logError := math.Log10(math.Max(metric.ColorErrorWorkingRMS, 1e-16))
			x := plot.Min.X + int(float64(metric.Level)/float64(maxLevel)*float64(plot.Dx()))
			y := plot.Min.Y + int((-logError)/16.0*float64(plot.Dy()))
			y = clampInt(y, plot.Min.Y, plot.Max.Y-1)
			points = append(points, image.Pt(x, y))
		}
		lineColor := chartLineColors[method]
		drawPolyline(dst, points, lineColor)
		for _, point := range points {
			fillRect(dst, image.Rect(point.X-3, point.Y-3, point.X+4, point.Y+4), lineColor)
		}
	}
	drawChartLegend(dst, methods, width)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return WritePNG(path, dst)
}

func drawChartFrame(dst *image.RGBA, plot image.Rectangle) {
	fillRect(dst, plot, color.RGBA{R: 25, G: 30, B: 40, A: 255})
	drawRectangleBorder(dst, plot, borderColor)
}

func drawLogErrorAxes(dst *image.RGBA, plot image.Rectangle, maxLevel int) {
	for exponent := 0; exponent <= 16; exponent += 4 {
		y := plot.Min.Y + int(float64(exponent)/16.0*float64(plot.Dy()))
		grid := image.Rect(plot.Min.X, y, plot.Max.X, y+1)
		fillRect(dst, grid, color.RGBA{R: 42, G: 50, B: 64, A: 255})
		label := "-" + itoa(exponent)
		if exponent == 0 {
			label = "0"
		}
		DrawText(dst, label, plot.Min.X-48, y-5, 1, mutedColor)
	}
	step := 1
	if maxLevel > 10 {
		step = 2
	}
	for level := 0; level <= maxLevel; level += step {
		x := plot.Min.X + int(float64(level)/float64(maxLevel)*float64(plot.Dx()))
		fillRect(dst, image.Rect(x, plot.Min.Y, x+1, plot.Max.Y), color.RGBA{R: 34, G: 41, B: 54, A: 255})
		DrawText(dst, "L"+itoa(level), x-8, plot.Max.Y+12, 1, mutedColor)
	}
	if maxLevel%step != 0 {
		x := plot.Max.X
		DrawText(dst, "L"+itoa(maxLevel), x-24, plot.Max.Y+12, 1, mutedColor)
	}
}

func drawChartLegend(dst *image.RGBA, methods []string, width int) {
	x := 470
	y := 34
	for _, method := range methods {
		c := chartLineColors[method]
		fillRect(dst, image.Rect(x, y, x+18, y+8), c)
		DrawText(dst, legendName(method), x+26, y-2, 1, textColor)
		x += 165
		if x > width-180 {
			x = 82
			y += 20
		}
	}
}

func legendName(method string) string {
	switch method {
	case "wrap_normalized":
		return "WRAP"
	case "naive_zero_pad":
		return "NAIVE"
	default:
		return "INSIDE"
	}
}

func drawPolyline(dst *image.RGBA, points []image.Point, c color.RGBA) {
	for index := 1; index < len(points); index++ {
		drawLine(dst, points[index-1], points[index], c)
	}
}

func drawLine(dst *image.RGBA, from, to image.Point, c color.RGBA) {
	dx := math.Abs(float64(to.X - from.X))
	dy := math.Abs(float64(to.Y - from.Y))
	steps := int(math.Max(dx, dy))
	if steps == 0 {
		dst.SetRGBA(from.X, from.Y, c)
		return
	}
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		x := int(math.Round(float64(from.X) + t*float64(to.X-from.X)))
		y := int(math.Round(float64(from.Y) + t*float64(to.Y-from.Y)))
		for oy := -1; oy <= 1; oy++ {
			for ox := -1; ox <= 1; ox++ {
				if ox*ox+oy*oy <= 2 {
					dst.SetRGBA(x+ox, y+oy, c)
				}
			}
		}
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
