package render

import (
	"image"
	"image/color"
	"math"

	"aelab/internal/scene"
)

func ToRGBA(frame *scene.HDR, exposureGain float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			c := frame.At(x, y)
			r := acesFilm(float64(c.R) * exposureGain)
			g := acesFilm(float64(c.G) * exposureGain)
			b := acesFilm(float64(c.B) * exposureGain)
			img.SetRGBA(x, y, color.RGBA{
				R: channel(r),
				G: channel(g),
				B: channel(b),
				A: 255,
			})
		}
	}
	return img
}

func acesFilm(x float64) float64 {
	const a, b, c, d, e = 2.51, 0.03, 2.43, 0.59, 0.14
	return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0, 1)
}

func channel(linear float64) uint8 {
	v := sRGB(linear)
	return uint8(math.Round(v * 255))
}

func sRGB(v float64) float64 {
	v = clamp(v, 0, 1)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func MeanLuma(img *image.RGBA) float64 {
	sum := 0.0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
		}
	}
	return sum / float64(bounds.Dx()*bounds.Dy()) / 65535
}

type ROIStats struct {
	Mean       float64
	StdDev     float64
	EdgeEnergy float64
	ClipLow    float64
	ClipHigh   float64
}

func AnalyzeROI(img *image.RGBA, r image.Rectangle) ROIStats {
	r = r.Intersect(img.Bounds())
	var values []float64
	sum := 0.0
	low, high := 0, 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			px := img.RGBAAt(x, y)
			l := (0.2126*float64(px.R) + 0.7152*float64(px.G) + 0.0722*float64(px.B)) / 255
			values = append(values, l)
			sum += l
			if l < 0.035 {
				low++
			}
			if l > 0.965 {
				high++
			}
		}
	}
	mean := sum / float64(len(values))
	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}

	edgeSum := 0.0
	edgeCount := 0
	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		for x := r.Min.X + 1; x < r.Max.X-1; x++ {
			edgeSum += math.Abs(lumaAt(img, x+1, y) - lumaAt(img, x-1, y))
			edgeSum += math.Abs(lumaAt(img, x, y+1) - lumaAt(img, x, y-1))
			edgeCount += 2
		}
	}

	n := float64(len(values))
	return ROIStats{
		Mean:       mean,
		StdDev:     math.Sqrt(variance / n),
		EdgeEnergy: edgeSum / float64(maxInt(1, edgeCount)),
		ClipLow:    float64(low) / n,
		ClipHigh:   float64(high) / n,
	}
}

func lumaAt(img *image.RGBA, x, y int) float64 {
	px := img.RGBAAt(x, y)
	return (0.2126*float64(px.R) + 0.7152*float64(px.G) + 0.0722*float64(px.B)) / 255
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
