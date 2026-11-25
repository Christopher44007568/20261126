package main

import (
	"fmt"
	"image/color"
)

var (
	white      = color.RGBA{R: 245, G: 248, B: 245, A: 255}
	nearBlack  = color.RGBA{R: 18, G: 24, B: 30, A: 255}
	gridColor  = color.RGBA{R: 58, G: 70, B: 82, A: 255}
	rayColor   = color.RGBA{R: 255, G: 236, B: 120, A: 255}
	red        = color.RGBA{R: 230, G: 32, B: 38, A: 255}
	amber      = color.RGBA{R: 238, G: 174, B: 45, A: 255}
	gray       = color.RGBA{R: 120, G: 130, B: 138, A: 255}
	background = color.RGBA{R: 35, G: 48, B: 58, A: 255}
)

func Render(result TraceResult, layers []Layer, ray Ray, tMin, tMax float64, prefix, directory string) ([]string, error) {
	side := NewCanvas(1000, 420)
	side.Fill(background)
	renderSideView(side, result, layers, ray, tMin, tMax)

	camera := NewCanvas(480, 360)
	renderCameraView(camera, result, layers)

	order := NewCanvas(720, 300)
	order.Fill(background)
	renderLayerOrder(order, result, tMin, tMax)

	transmission := NewCanvas(720, 340)
	transmission.Fill(background)
	renderTransmission(transmission, result)

	mask := NewCanvas(480, 360)
	renderRangeMask(mask, result, layers)

	files := []string{
		directory + "/" + prefix + "_side.png",
		directory + "/" + prefix + "_camera.png",
		directory + "/" + prefix + "_layer_order.png",
		directory + "/" + prefix + "_transmission.png",
		directory + "/" + prefix + "_range_mask.png",
	}
	canvases := []Canvas{side, camera, order, transmission, mask}
	for index, canvas := range canvases {
		if err := canvas.Save(files[index]); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func renderSideView(canvas Canvas, result TraceResult, layers []Layer, ray Ray, tMin, tMax float64) {
	canvas.DrawText(24, 20, "SIDE VIEW: X DEPTH AND Y HEIGHT", 2, white)
	left, top, width, height := 70, 70, 880, 270

	for i := 0; i <= 10; i++ {
		x := left + width*i/10
		canvas.DrawLine(x, top, x, top+height, gridColor)
	}
	for i := 0; i <= 6; i++ {
		y := top + height*i/6
		canvas.DrawLine(left, y, left+width, y, gridColor)
	}

	for index, layer := range layers {
		x0 := mapX(layer.Min.X, left, width)
		x1 := mapX(layer.Max.X, left, width)
		y0 := mapY(layer.Max.Y, top, height)
		y1 := mapY(layer.Min.Y, top, height)
		fill := blend(background, colorFromRGB(layer.Tint), 0.34)
		canvas.FillRect(x0, y0, x1-x0, y1-y0, fill)
		canvas.DrawRect(x0, y0, x1-x0, y1-y0, white)
		label := fmt.Sprintf("%d:%s", index+1, layer.Name)
		canvas.DrawTextCentered((x0+x1)/2, y0+8, label, 1, white)
	}

	startX := mapX(ray.At(tMin).X, left, width)
	endX := mapX(ray.At(tMax).X, left, width)
	rayY := mapY(ray.Origin.Y, top, height)
	canvas.DrawLine(startX, rayY, endX, rayY, rayColor)

	for index, hit := range result.Hits {
		x0 := mapX(hit.Entry.X, left, width)
		x1 := mapX(hit.Exit.X, left, width)
		canvas.DrawLine(x0, rayY-12, x0, rayY+12, white)
		canvas.DrawLine(x1, rayY-12, x1, rayY+12, gray)
		label := fmt.Sprintf("%d %s E=%s", index+1, hit.LayerName, shortRGB(hit.EnergyAfter))
		labelY := rayY - 48
		if index%2 == 1 {
			labelY = rayY + 18
		}
		canvas.DrawText(max(12, min(900, x0-30)), labelY, label, 1, white)
		if hit.OutOfRange {
			canvas.FillRect(x0-5, rayY-18, 10, 36, red)
		}
	}

	if result.Unprocessed != nil {
		x0 := mapX(result.Unprocessed.Start.X, left, width)
		x1 := mapX(result.Unprocessed.End.X, left, width)
		canvas.DrawHatchedRect(x0, rayY-24, x1-x0, 48, blend(background, amber, 0.35), amber)
		canvas.DrawText(x0+6, top+height+18, "UNPROCESSED SEGMENT: BUDGET EXHAUSTED", 1, red)
	}

	canvas.DrawText(left, top+height+34, "X: -2 TO 2.5", 1, white)
	canvas.DrawText(left, 55, fmt.Sprintf("INITIAL E=%s FINAL E=%s", shortRGB(whiteEnergy()), shortRGB(result.Energy)), 1, rayColor)
}

func renderCameraView(canvas Canvas, result TraceResult, layers []Layer) {
	canvas.Fill(background)
	canvas.DrawText(20, 18, "CAMERA VIEW", 2, white)

	if len(result.Hits) > 0 {
		last := result.Hits[len(result.Hits)-1]
		layer := layers[last.LayerIndex]
		colorValue := RGB{
			R: last.EnergyAfter.R * (0.35 + 0.65*layer.Tint.R),
			G: last.EnergyAfter.G * (0.35 + 0.65*layer.Tint.G),
			B: last.EnergyAfter.B * (0.35 + 0.65*layer.Tint.B),
		}
		canvas.FillRect(0, 48, canvas.Image.Bounds().Dx(), canvas.Image.Bounds().Dy()-48, colorFromRGB(colorValue))
	}

	if result.Unprocessed != nil {
		canvas.DrawHatchedRect(0, 48, canvas.Image.Bounds().Dx(), canvas.Image.Bounds().Dy()-48,
			blend(background, amber, 0.25), amber)
		canvas.DrawTextCentered(240, 174, "UNPROCESSED", 3, red)
		canvas.DrawTextCentered(240, 205, "SEGMENT KEPT", 2, red)
	}

	for index, hit := range result.Hits {
		label := fmt.Sprintf("%d %s BEFORE %s AFTER %s", index+1, hit.LayerName,
			shortRGB(hit.EnergyBefore), shortRGB(hit.EnergyAfter))
		canvas.DrawText(14, 78+index*28, label, 1, white)
		if hit.OutOfRange {
			canvas.FillRect(0, 112+index*28, canvas.Image.Bounds().Dx(), 18, red)
		}
	}

	status := "COMPLETE"
	if !result.Complete {
		status = "BUDGET LIMIT: UNPROCESSED REMAINS"
	}
	canvas.DrawText(14, 330, status, 1, rayColor)
}

func mapX(value float64, left, width int) int {
	return left + int((value+2.0)/4.5*float64(width))
}

func mapY(value float64, top, height int) int {
	return top + int((2.0-value)/4.0*float64(height))
}

func whiteEnergy() RGB {
	return RGB{R: 1, G: 1, B: 1}
}

func shortRGB(value RGB) string {
	return fmt.Sprintf("(%.2f %.2f %.2f)", value.R, value.G, value.B)
}
