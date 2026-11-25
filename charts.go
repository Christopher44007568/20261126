package main

import (
	"fmt"
	"image/color"
)

func renderLayerOrder(canvas Canvas, result TraceResult, tMin, tMax float64) {
	canvas.DrawText(24, 22, "LAYER ORDER AND HIT BUDGET", 2, white)
	baseX, baseY, barHeight := 60, 126, 72
	span := tMax - tMin

	for index, hit := range result.Hits {
		x0 := baseX + int((hit.T0-tMin)/span*600)
		x1 := baseX + int((hit.T1-tMin)/span*600)
		layerColor := colorFromRGB(hit.Transmittance)
		canvas.FillRect(x0, baseY, max(12, x1-x0), barHeight, blend(background, layerColor, 0.75))
		canvas.DrawRect(x0, baseY, max(12, x1-x0), barHeight, white)
		label := fmt.Sprintf("%d %s", index+1, hit.LayerName)
		canvas.DrawText(x0+4, baseY-18, label, 1, white)
		energy := shortRGB(hit.EnergyAfter)
		canvas.DrawText(max(8, x0-12), baseY+barHeight+26, energy, 1, rayColor)
	}

	if result.Unprocessed != nil {
		x0 := baseX + int((result.Unprocessed.T0-tMin)/span*600)
		x1 := baseX + 600
		canvas.DrawHatchedRect(x0, baseY, x1-x0, barHeight, blend(background, amber, 0.35), amber)
		canvas.DrawText(x0+4, baseY+28, "UNPROCESSED", 1, red)
		for index, name := range result.RemainingLayers {
			label := fmt.Sprintf("%d:%s", len(result.Hits)+index+1, name)
			canvas.DrawText(x0+8, baseY+barHeight+54+index*22, label, 1, red)
		}
	}

	canvas.DrawLine(baseX, baseY+barHeight, baseX+600, baseY+barHeight, white)
	canvas.DrawText(baseX, 262, fmt.Sprintf("T RANGE %.1F TO %.1F - EVERY SLAB IS ONE HIT", tMin, tMax), 1, gray)
}

func renderTransmission(canvas Canvas, result TraceResult) {
	canvas.DrawText(24, 22, "CUMULATIVE TRANSMISSION", 2, white)
	left, top, width, height := 70, 70, 600, 210

	for i := 0; i <= 4; i++ {
		y := top + height*i/4
		canvas.DrawLine(left, y, left+width, y, gridColor)
		label := fmt.Sprintf("%.2F", 1-float64(i)/4)
		canvas.DrawText(20, y-5, label, 1, white)
	}
	canvas.DrawLine(left, top, left, top+height, white)
	canvas.DrawLine(left, top+height, left+width, top+height, white)

	series := []struct {
		name  string
		value float64
		color color.RGBA
	}{
		{name: "R", value: result.Energy.R, color: color.RGBA{R: 235, G: 76, B: 76, A: 255}},
		{name: "G", value: result.Energy.G, color: color.RGBA{R: 92, G: 220, B: 120, A: 255}},
		{name: "B", value: result.Energy.B, color: color.RGBA{R: 92, G: 160, B: 240, A: 255}},
	}

	for legendIndex, channel := range series {
		previousX, previousY := left, top+height-int(whiteEnergy().R*float64(height))
		for index, hit := range result.Hits {
			energy := channelEnergy(hit.EnergyAfter, channel.name)
			x := left + (index+1)*width/(len(result.Hits)+1)
			y := top + height - int(energy*float64(height))
			canvas.DrawLine(previousX, previousY, x, y, channel.color)
			canvas.FillRect(x-3, y-3, 6, 6, channel.color)
			previousX, previousY = x, y
		}
		legendY := top + 18 + legendIndex*22
		canvas.DrawText(left+width-92, legendY, channel.name+" FINAL", 1, channel.color)
	}

	for index, hit := range result.Hits {
		x := left + (index+1)*width/(len(result.Hits)+1)
		label := fmt.Sprintf("%d %s", index+1, hit.LayerName)
		canvas.DrawText(x-24, top+height+22, label, 1, white)
	}

	if !result.Complete {
		canvas.DrawText(left, 318, "CHART STOPS AT BUDGET; UNPROCESSED SEGMENT IS NOT FOLDED INTO BACKGROUND", 1, red)
	}
}

func renderRangeMask(canvas Canvas, result TraceResult, layers []Layer) {
	canvas.Fill(nearBlack)
	canvas.DrawText(20, 20, "ENERGY RANGE MASK [0,1]", 2, white)
	hasMask := false

	for index, hit := range result.Hits {
		if hit.OutOfRange {
			hasMask = true
			canvas.FillRect(34+index*138, 64, 112, 220, red)
			label := fmt.Sprintf("%d %s", index+1, hit.LayerName)
			canvas.DrawText(44+index*138, 76, label, 1, white)
		}
	}

	if !hasMask {
		canvas.DrawTextCentered(240, 170, "NO OUT OF RANGE ENERGY", 2, gray)
	}
	canvas.DrawText(20, 326, "RED DENOTES NAN INF OR NEGATIVE OR GREATER THAN ONE ENERGY", 1, white)
}

func channelEnergy(value RGB, channel string) float64 {
	switch channel {
	case "R":
		return value.R
	case "G":
		return value.G
	default:
		return value.B
	}
}
