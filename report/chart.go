package report

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"autofocus/autofocus"
)

// SharpnessCurve renders the contrast-detection curve, lock marker, hysteresis
// band, and the sampled focus positions.
func SharpnessCurve(samples []autofocus.Sample, cfg autofocus.Config, snap autofocus.Snapshot) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 1000, 600))
	fill(img, img.Bounds(), colorBackground)
	drawText(img, 28, 18, "CONTRAST DETECT / SHARPNESS CURVE", 2, colorText)
	drawLegend(img, 690, 20)

	plot := image.Rect(96, 92, 944, 430)
	fill(img, plot, colorPanel)
	points := CurvePoints(samples)
	yMax := curveMax(points, snap)
	if yMax <= 0 {
		yMax = 1
	}
	yMax *= 1.12

	if snap.LockScore > 0 {
		floor := snap.LockScore * (1 - cfg.LockHysteresis)
		y0 := scoreY(floor, yMax, plot)
		y1 := scoreY(snap.LockScore, yMax, plot)
		fill(img, image.Rect(plot.Min.X, y1, plot.Max.X, y0), color.RGBA{R: 18, G: 55, B: 39, A: 255})
		dottedHorizontal(img, plot.Min.X, plot.Max.X, y0, colorGreen, 3, 4)
		dottedHorizontal(img, plot.Min.X, plot.Max.X, y1, colorGreen, 3, 4)
		drawText(img, plot.Max.X-textWidth(hysteresisLabel(cfg), 1)-8, y0+5,
			hysteresisLabel(cfg), 1, colorGreen)
	}

	for i := 0; i <= 4; i++ {
		y := plot.Min.Y + i*plot.Dy()/4
		drawLine(img, plot.Min.X, y, plot.Max.X, y, colorGrid)
		value := yMax * float64(4-i) / 4
		drawText(img, 26, y-2, fmt.Sprintf("%.3f", value), 1, colorMuted)
	}
	for i := 0; i <= 4; i++ {
		x := plot.Min.X + i*plot.Dx()/4
		drawLine(img, x, plot.Min.Y, x, plot.Max.Y, colorGrid)
		pos := cfg.MinPosition + i*(cfg.MaxPosition-cfg.MinPosition)/4
		drawText(img, x-textWidth(fmt.Sprint(pos), 1)/2, plot.Max.Y+8,
			fmt.Sprint(pos), 1, colorMuted)
	}
	drawText(img, plot.Min.X, plot.Max.Y+30, "FOCUS POSITION", 1, colorMuted)
	drawText(img, 26, plot.Min.Y-15, "SCORE", 1, colorMuted)

	for i := 1; i < len(points); i++ {
		p0, p1 := points[i-1], points[i]
		drawLine(img, posX(p0.Position, cfg, plot), scoreY(p0.Sharpness, yMax, plot),
			posX(p1.Position, cfg, plot), scoreY(p1.Sharpness, yMax, plot), colorMuted)
	}
	for _, p := range points {
		x, y := posX(p.Position, cfg, plot), scoreY(p.Sharpness, yMax, plot)
		drawPoint(img, x, y, phaseColor(p.Phase))
	}

	bestX := posX(snap.BestPosition, cfg, plot)
	if snap.BestPosition >= cfg.MinPosition && snap.BestPosition <= cfg.MaxPosition {
		dottedVertical(img, bestX, plot.Min.Y, plot.Max.Y, colorText, 3, 4)
		label := fmt.Sprintf("BEST %d", snap.BestPosition)
		drawText(img, clampInt(bestX-textWidth(label, 1)/2, plot.Min.X, plot.Max.X-textWidth(label, 1)),
			plot.Min.Y+7, label, 1, colorText)
	}
	if snap.LockedPosition >= 0 {
		lx := posX(snap.LockedPosition, cfg, plot)
		fill(img, image.Rect(lx-1, plot.Max.Y-9, lx+1, plot.Max.Y+1), colorGreen)
		drawText(img, clampInt(lx-textWidth("LOCK", 1)/2, plot.Min.X, plot.Max.X-textWidth("LOCK", 1)),
			plot.Max.Y-20, "LOCK", 1, colorGreen)
	}

	drawCoverage(img, image.Rect(96, 500, 944, 516), points, cfg, snap)
	drawText(img, 28, 570, fmt.Sprintf(
		"MOVES %d   REVERSALS %d   MOVES AFTER LOCK %d   SAMPLES %d",
		snap.Moves, snap.Reversals, snap.MovesAfterLock, len(samples)), 2, colorText)
	drawText(img, 28, 548, "FOCUS POSITION COVERAGE", 1, colorMuted)
	return img
}

func drawLegend(img *image.RGBA, x, y int) {
	items := []struct {
		label string
		phase string
	}{
		{"COARSE", "coarse"},
		{"FINE", "fine"},
		{"CONFIRM", "confirm"},
	}
	for _, item := range items {
		fill(img, image.Rect(x, y+2, x+8, y+8), phaseColor(item.phase))
		drawText(img, x+13, y, item.label, 1, colorMuted)
		x += textWidth(item.label, 1) + 34
	}
}

func drawCoverage(img *image.RGBA, r image.Rectangle, points []CurvePoint,
	cfg autofocus.Config, snap autofocus.Snapshot) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, colorGrid)
	for _, p := range points {
		x := posX(p.Position, cfg, r)
		drawLine(img, x, r.Min.Y-2, x, r.Min.Y+2, phaseColor(p.Phase))
	}
	if snap.LockedPosition >= 0 {
		x := posX(snap.LockedPosition, cfg, r)
		drawLine(img, x, r.Min.Y-5, x, r.Min.Y+5, colorGreen)
	}
}

func drawPoint(img *image.RGBA, x, y int, c color.RGBA) {
	fill(img, image.Rect(x-2, y-2, x+3, y+3), c)
}

func dottedHorizontal(img *image.RGBA, x0, x1, y int, c color.RGBA, on, off int) {
	for x := x0; x <= x1; x++ {
		if (x-x0)%(on+off) < on {
			set(img, x, y, c)
		}
	}
}

func curveMax(points []CurvePoint, snap autofocus.Snapshot) float64 {
	max := snap.LockScore
	if snap.BestSharpness > max {
		max = snap.BestSharpness
	}
	for _, p := range points {
		if p.Sharpness > max {
			max = p.Sharpness
		}
	}
	return max
}

func scoreY(score, max float64, plot image.Rectangle) int {
	if max <= 0 {
		return plot.Max.Y
	}
	y := plot.Max.Y - int(math.Round(score/max*float64(plot.Dy())))
	return clampInt(y, plot.Min.Y, plot.Max.Y)
}

func posX(pos int, cfg autofocus.Config, plot image.Rectangle) int {
	if cfg.MaxPosition <= cfg.MinPosition {
		return plot.Min.X
	}
	x := plot.Min.X + (pos-cfg.MinPosition)*(plot.Dx()-1)/(cfg.MaxPosition-cfg.MinPosition)
	return clampInt(x, plot.Min.X, plot.Max.X-1)
}

func phaseColor(phase string) color.RGBA {
	switch phase {
	case "coarse":
		return colorGreen
	case "fine":
		return colorCyan
	case "confirm":
		return colorOrange
	default:
		return colorMuted
	}
}

func hysteresisLabel(cfg autofocus.Config) string {
	return fmt.Sprintf("HYST %.0f%%", cfg.LockHysteresis*100)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
