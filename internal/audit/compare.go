package audit

import (
	"fmt"
	"math"
	"sort"
)

type DifferenceMetrics struct {
	Mean            float64 `json:"mean_absolute_error"`
	RMSE            float64 `json:"rmse"`
	P95             float64 `json:"p95"`
	P99             float64 `json:"p99"`
	Max             float64 `json:"max"`
	ChangedPixels   int     `json:"changed_pixels"`
	ChangedFraction float64 `json:"changed_fraction"`
	TotalPixels     int     `json:"total_pixels"`
}

// BuildErrorMap compares final display values. The heat map amplifies local
// differences by gain so small color-space errors remain visible. Exact,
// unamplified statistics are returned alongside the image.
func BuildErrorMap(correct, misread *Image, gain float64) (*Image, DifferenceMetrics, error) {
	if correct.Width != misread.Width || correct.Height != misread.Height {
		return nil, DifferenceMetrics{}, fmt.Errorf("cannot compare images with different dimensions")
	}
	if gain <= 0 {
		gain = 1
	}

	picture := NewImage(correct.Width, correct.Height)
	magnitudes := make([]float64, len(correct.Pixels))
	metrics := DifferenceMetrics{TotalPixels: len(correct.Pixels)}
	sum := 0.0
	sumSquares := 0.0

	for index, correctValue := range correct.Pixels {
		misreadValue := misread.Pixels[index]
		deltaRed := math.Abs(correctValue.X - misreadValue.X)
		deltaGreen := math.Abs(correctValue.Y - misreadValue.Y)
		deltaBlue := math.Abs(correctValue.Z - misreadValue.Z)
		magnitude := (deltaRed + deltaGreen + deltaBlue) / 3
		magnitudes[index] = magnitude
		sum += magnitude
		sumSquares += magnitude * magnitude
		if magnitude > metrics.Max {
			metrics.Max = magnitude
		}
		if magnitude > 1.0/255.0 {
			metrics.ChangedPixels++
		}

		amplified := clamp01(magnitude * gain)
		picture.Pixels[index] = heatColor(amplified)
	}

	if metrics.TotalPixels > 0 {
		metrics.Mean = sum / float64(metrics.TotalPixels)
		metrics.RMSE = math.Sqrt(sumSquares / float64(metrics.TotalPixels))
		metrics.ChangedFraction = float64(metrics.ChangedPixels) / float64(metrics.TotalPixels)
	}
	metrics.P95 = percentile(magnitudes, 0.95)
	metrics.P99 = percentile(magnitudes, 0.99)
	return picture, metrics, nil
}

func percentile(values []float64, fraction float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(math.Ceil(fraction*float64(len(sorted)))) - 1
	index = int(clamp(float64(index), 0, float64(len(sorted)-1)))
	return sorted[index]
}

func heatColor(amount float64) Vec3 {
	amount = clamp01(amount)
	stops := []struct {
		position float64
		color    Vec3
	}{
		{0.00, Vec3{0.010, 0.014, 0.026}},
		{0.22, Vec3{0.055, 0.120, 0.620}},
		{0.46, Vec3{0.000, 0.680, 0.820}},
		{0.68, Vec3{0.920, 0.610, 0.050}},
		{0.85, Vec3{1.000, 0.330, 0.035}},
		{1.00, Vec3{1.000, 0.960, 0.520}},
	}
	for index := 1; index < len(stops); index++ {
		if amount <= stops[index].position {
			width := stops[index].position - stops[index-1].position
			local := (amount - stops[index-1].position) / width
			return blendColor(stops[index-1].color, stops[index].color, local)
		}
	}
	return stops[len(stops)-1].color
}

func BuildComparison(left, right *Image, leftLabel, rightLabel string) (*Image, error) {
	if left.Width != right.Width || left.Height != right.Height {
		return nil, fmt.Errorf("comparison images must have matching dimensions")
	}
	const (
		margin    = 24
		gap       = 18
		header    = 50
		border    = 1
		labelSize = 2
	)
	width := margin*2 + left.Width*2 + gap
	height := margin*2 + header + left.Height
	picture := NewImage(width, height)
	background := Vec3{0.043, 0.049, 0.060}
	for index := range picture.Pixels {
		picture.Pixels[index] = background
	}

	leftX := margin
	rightX := margin + left.Width + gap
	imageY := margin + header
	drawText(picture, leftX, margin+10, leftLabel, labelSize, Vec3{0.90, 0.92, 0.95})
	drawText(picture, rightX, margin+10, rightLabel, labelSize, Vec3{1.00, 0.72, 0.38})
	copyImage(picture, left, leftX, imageY)
	copyImage(picture, right, rightX, imageY)

	borderColor := Vec3{0.18, 0.20, 0.24}
	drawBorder(picture, leftX-border, imageY-border, left.Width+2, left.Height+2, borderColor)
	drawBorder(picture, rightX-border, imageY-border, right.Width+2, right.Height+2, borderColor)
	return picture, nil
}

func copyImage(target, source *Image, originX, originY int) {
	for row := 0; row < source.Height; row++ {
		for column := 0; column < source.Width; column++ {
			targetX, targetY := originX+column, originY+row
			if targetX >= 0 && targetX < target.Width && targetY >= 0 && targetY < target.Height {
				target.Set(targetX, targetY, source.At(column, row))
			}
		}
	}
}

func drawBorder(picture *Image, x, y, width, height int, color Vec3) {
	for column := 0; column < width; column++ {
		setIfInside(picture, x+column, y, color)
		setIfInside(picture, x+column, y+height-1, color)
	}
	for row := 0; row < height; row++ {
		setIfInside(picture, x, y+row, color)
		setIfInside(picture, x+width-1, y+row, color)
	}
}

func setIfInside(picture *Image, x, y int, color Vec3) {
	if x >= 0 && x < picture.Width && y >= 0 && y < picture.Height {
		picture.Set(x, y, color)
	}
}
