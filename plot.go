package main

import "math"

const (
	plotW, plotH = 760, 600
	plotLeft     = 70
	plotRight    = 720
	plotTop      = 60
	plotBottom   = 460
)

// PlotResponse renders g(z) = ln(E*t) against digital level z. Recovered
// segments are colored by status: green=ok, red=non-monotonic,
// orange=extrapolated/saturated. The known true curve is drawn in blue.
func PlotResponse(g, gTrue []float64, nonMonotonic, extrapolated []bool, title string) *RGB8 {
	img := NewRGB8(plotW, plotH)
	fill(img, 245, 245, 245)

	minV, maxV := g[0], g[0]
	for _, v := range g {
		minV, maxV = math.Min(minV, v), math.Max(maxV, v)
	}
	for _, v := range gTrue {
		minV, maxV = math.Min(minV, v), math.Max(maxV, v)
	}
	pad := (maxV - minV) * 0.08
	minV -= pad
	maxV += pad

	mapX := func(z int) int {
		return plotLeft + (plotRight-plotLeft)*z/255
	}
	mapY := func(v float64) int {
		f := (v - minV) / (maxV - minV)
		return plotBottom - int(f*(plotBottom-plotTop))
	}

	drawGrid(img, minV, maxV, mapX, mapY)
	drawAxes(img, minV, maxV, mapX, mapY)

	// Known true response (dashed blue, drawn first so recovered curve wins).
	for z := 1; z < 256; z++ {
		drawLine(img, mapX(z-1), mapY(gTrue[z-1]), mapX(z), mapY(gTrue[z]),
			60, 90, 220)
	}

	// Recovered response, colored per status. Red takes priority over orange.
	for z := 1; z < 256; z++ {
		r, gg, b := uint8(30), uint8(150), uint8(60)
		if extrapolated[z] {
			r, gg, b = 240, 150, 30
		}
		if nonMonotonic[z] {
			r, gg, b = 220, 40, 40
		}
		drawLine(img, mapX(z-1), mapY(g[z-1]), mapX(z), mapY(g[z]), r, gg, b)
	}

	DrawText(img, plotLeft, 20, title, 20, 20, 20)
	DrawText(img, plotLeft, 38, "G(Z) = LN(E*T) VS DIGITAL LEVEL Z", 70, 70, 70)
	legend(img)
	return img
}

func drawGrid(img *RGB8, minV, maxV float64, mapX func(int) int, mapY func(float64) int) {
	for z := 0; z <= 255; z += 32 {
		x := mapX(z)
		drawLine(img, x, plotTop, x, plotBottom, 225, 225, 225)
	}
	steps := 6
	for i := 0; i <= steps; i++ {
		v := minV + (maxV-minV)*float64(i)/float64(steps)
		y := mapY(v)
		drawLine(img, plotLeft, y, plotRight, y, 225, 225, 225)
	}
}

func drawAxes(img *RGB8, minV, maxV float64, mapX func(int) int, mapY func(float64) int) {
	drawLine(img, plotLeft, plotTop, plotLeft, plotBottom, 0, 0, 0)
	drawLine(img, plotLeft, plotBottom, plotRight, plotBottom, 0, 0, 0)
	for z := 0; z <= 255; z += 32 {
		x := mapX(z)
		DrawText(img, x-9, plotBottom+8, itoa(z), 0, 0, 0)
	}
	DrawText(img, plotLeft+(plotRight-plotLeft)/2-40, plotBottom+28, "DIGITAL LEVEL Z", 0, 0, 0)

	steps := 6
	for i := 0; i <= steps; i++ {
		v := minV + (maxV-minV)*float64(i)/float64(steps)
		y := mapY(v)
		label := ftoa(v, 1)
		DrawText(img, 8, y-3, label, 0, 0, 0)
	}
	DrawText(img, 10, plotTop-14, "LN(E*T)", 0, 0, 0)
}

func legend(img *RGB8) {
	y := 520
	items := []struct {
		text    string
		r, g, b uint8
	}{
		{"OK", 30, 150, 60},
		{"NON-MONO", 220, 40, 40},
		{"EXTRAPOLATED", 240, 150, 30},
		{"TRUE CURVE", 60, 90, 220},
	}
	x := plotLeft
	for _, it := range items {
		drawLine(img, x, y+3, x+22, y+3, it.r, it.g, it.b)
		DrawText(img, x+28, y, it.text, 40, 40, 40)
		x += 28 + 6*len(it.text) + 34
	}
}

func fill(img *RGB8, r, g, b uint8) {
	for i := 0; i < len(img.Pix); i += 3 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2] = r, g, b
	}
}

// drawLine is an integer Bresenham line.
func drawLine(img *RGB8, x0, y0, x1, y1 int, r, g, b uint8) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && x0 < img.W && y0 >= 0 && y0 < img.H {
			img.Set(x0, y0, r, g, b)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func ftoa(v float64, digits int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	scale := math.Pow(10, float64(digits))
	scaled := int(math.Round(v * scale))
	whole := scaled / int(scale)
	frac := scaled % int(scale)
	s := itoa(whole) + "."
	for i := digits - 1; i > 0; i-- {
		if frac < int(math.Pow10(i)) {
			s += "0"
		}
	}
	s += itoa(frac)
	if neg {
		return "-" + s
	}
	return s
}
