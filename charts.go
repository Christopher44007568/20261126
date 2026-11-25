package main

import (
	"fmt"
	"path/filepath"
)

const (
	chartWidth  = 900
	chartHeight = 520
	plotLeft    = 78
	plotRight   = 840
	plotTop     = 72
	plotBottom  = 420
)

func writeCharts(outputDir string, report RunReport) error {
	if err := writePPM(filepath.Join(outputDir, "cct_trajectory.ppm"), cctTrajectory(report)); err != nil {
		return err
	}
	if err := writePPM(filepath.Join(outputDir, "neutral_error_map.ppm"), neutralErrorMap(report)); err != nil {
		return err
	}
	return nil
}

func cctTrajectory(report RunReport) *Canvas {
	canvas := NewCanvas(chartWidth, chartHeight)
	canvas.Fill(RGB8{245, 245, 242})
	drawText(canvas, 260, 18, "CCT TRAJECTORY AFTER AWB", black)

	minCCT, maxCCT := 2000.0, 9000.0
	for i := 0; i <= 7; i++ {
		value := minCCT + float64(i)*(maxCCT-minCCT)/7
		y := plotY(value, minCCT, maxCCT)
		drawLine(canvas, plotLeft, y, plotRight, y, RGB8{218, 218, 214})
		drawText(canvas, 12, y-2, fmt.Sprintf("%d", int(value)), gray8)
	}

	for _, frame := range []int{10, 14, 24, 28} {
		x := frameX(frame)
		drawLine(canvas, x, plotTop, x, plotBottom, RGB8{202, 202, 198})
	}
	drawRect(canvas, plotLeft, plotTop, plotRight-plotLeft, plotBottom-plotTop, gray8)
	drawText(canvas, plotLeft, plotBottom+18, "FRAME 0", gray8)
	drawText(canvas, plotRight-50, plotBottom+18, "FRAME 39", gray8)

	series := []struct {
		Name  string
		Color RGB8
		Value func(FrameReport) float64
	}{
		{"TRUE", black, func(f FrameReport) float64 { return f.TrueCCT }},
		{"GRAY", red8, func(f FrameReport) float64 { return f.GrayCCT }},
		{"WHITE", blue8, func(f FrameReport) float64 { return f.WhiteCCT }},
		{"SMOOTH", green8, func(f FrameReport) float64 { return f.SmoothCCT }},
	}

	for _, item := range series {
		var previousX, previousY int
		for i, frame := range report.Frames {
			value := item.Value(frame)
			x := frameX(i)
			y := plotY(value, minCCT, maxCCT)
			if i > 0 {
				drawLine(canvas, previousX, previousY, x, y, item.Color)
			}
			canvas.FillRect(x-2, y-2, 5, 5, item.Color)
			previousX, previousY = x, y
		}
	}

	legendX := 120
	for _, item := range series {
		drawLine(canvas, legendX, 462, legendX+24, 462, item.Color)
		drawText(canvas, legendX+30, 459, item.Name, black)
		legendX += 175
	}
	drawText(canvas, plotLeft+10, plotTop+14, "BANDS MARK LIGHT TRANSITIONS", gray8)
	return canvas
}

func neutralErrorMap(report RunReport) *Canvas {
	cellWidth, cellHeight := 20, 42
	canvasWidth := plotLeft + frameCount*cellWidth + 30
	canvasHeight := 330
	canvas := NewCanvas(canvasWidth, canvasHeight)
	canvas.Fill(RGB8{245, 245, 242})
	drawText(canvas, 250, 18, "NEUTRAL COLOR RMSE MAP", black)

	rows := []struct {
		Name  string
		Error func(FrameReport) float64
	}{
		{"INPUT", func(f FrameReport) float64 { return f.InputError }},
		{"GRAY", func(f FrameReport) float64 { return f.GrayError }},
		{"WHITE", func(f FrameReport) float64 { return f.WhiteError }},
		{"SMOOTH", func(f FrameReport) float64 { return f.SmoothError }},
	}
	maxError := 0.001
	for _, frame := range report.Frames {
		maxError = maxFloat(maxError, frame.InputError, frame.GrayError, frame.WhiteError, frame.SmoothError)
	}

	for rowIndex, row := range rows {
		y := 58 + rowIndex*58
		drawText(canvas, 12, y+16, row.Name, black)
		for i, frame := range report.Frames {
			value := row.Error(frame)
			color := heatColor(value, maxError)
			x := plotLeft + i*cellWidth
			canvas.FillRect(x, y, cellWidth-2, cellHeight, color)
			drawRect(canvas, x, y, cellWidth-2, cellHeight, RGB8{235, 235, 232})
		}
	}

	for i := 0; i <= 4; i++ {
		x := plotLeft + i*frameCount*cellWidth/4
		drawText(canvas, x-6, 280, fmt.Sprintf("%d", i*frameCount/4), gray8)
	}
	for i := 0; i < 10; i++ {
		value := maxError * float64(i) / 9
		color := heatColor(value, maxError)
		x := 610 + i*22
		canvas.FillRect(x, 246, 22, 22, color)
	}
	drawText(canvas, 610, 280, "LOW", gray8)
	drawText(canvas, 792, 280, "HIGH", gray8)
	return canvas
}

func frameX(index int) int {
	return plotLeft + index*(plotRight-plotLeft)/(frameCount-1)
}

func plotY(value, minValue, maxValue float64) int {
	normalized := (value - minValue) / (maxValue - minValue)
	return plotBottom - int(normalized*(plotBottom-plotTop))
}

func maxFloat(values ...float64) float64 {
	result := values[0]
	for _, value := range values[1:] {
		if value > result {
			result = value
		}
	}
	return result
}
