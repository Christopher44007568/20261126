package autofocus

import (
	"image"
	"math"
)

// Measurement is computed from the configured focus ROI.
type Measurement struct {
	Sharpness float64 `json:"sharpness"`
	Contrast  float64 `json:"contrast"`
	Motion    float64 `json:"motion"`
	ROI       image.Rectangle
}

// CalculateMeasurement converts the frame to luminance and computes a
// tenengrad focus score. The score is the mean squared Sobel gradient; only
// gradients above a small noise floor contribute.
func CalculateMeasurement(img image.Image, roi ROI, previous []float64) (Measurement, []float64) {
	b := img.Bounds()
	r := pixelROI(b, roi)
	luma := luminanceROI(img, r)
	motion := frameMotion(luma, previous)
	sharp, contrast := tenengrad(luma, r.Dx())
	return Measurement{
		Sharpness: sharp,
		Contrast:  contrast,
		Motion:    motion,
		ROI:       r,
	}, luma
}

func pixelROI(b image.Rectangle, roi ROI) image.Rectangle {
	x0 := b.Min.X + int(math.Round(roi.X*float64(b.Dx())))
	y0 := b.Min.Y + int(math.Round(roi.Y*float64(b.Dy())))
	x1 := b.Min.X + int(math.Round((roi.X+roi.W)*float64(b.Dx())))
	y1 := b.Min.Y + int(math.Round((roi.Y+roi.H)*float64(b.Dy())))
	return image.Rect(x0, y0, x1, y1).Intersect(b)
}

func luminanceROI(img image.Image, r image.Rectangle) []float64 {
	out := make([]float64, r.Dx()*r.Dy())
	gray, isGray := img.(*image.Gray)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			var v float64
			if isGray {
				v = float64(gray.GrayAt(x, y).Y) / 255
			} else {
				r8, g8, b8, _ := img.At(x, y).RGBA()
				v = (0.2126*float64(r8) + 0.7152*float64(g8) + 0.0722*float64(b8)) / 65535
			}
			out[(y-r.Min.Y)*r.Dx()+(x-r.Min.X)] = v
		}
	}
	return out
}

func tenengrad(luma []float64, width int) (sharpness, contrast float64) {
	height := len(luma) / width
	if width < 3 || height < 3 {
		return 0, 0
	}
	var sumSq float64
	var gradientSum float64
	var gradientN int
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			i := y*width + x
			gx := luma[i+1] - luma[i-1]
			gy := luma[i+width] - luma[i-width]
			g := math.Hypot(gx, gy)
			sumSq += g * g
			gradientSum += luma[i]
			gradientN++
		}
	}
	if gradientN == 0 {
		return 0, 0
	}
	mean := gradientSum / float64(gradientN)
	var variance float64
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			d := luma[y*width+x] - mean
			variance += d * d
		}
	}
	variance /= float64(gradientN)
	return sumSq / float64(gradientN), math.Sqrt(variance)
}

func frameMotion(current, previous []float64) float64 {
	if len(previous) != len(current) || len(current) == 0 {
		return 0
	}
	var sum float64
	for i := range current {
		sum += math.Abs(current[i] - previous[i])
	}
	return sum / float64(len(current))
}
