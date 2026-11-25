package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func clampColor(v float64) uint16 {
	v = math.Max(0, math.Min(1, v))
	return uint16(math.Round(v * 65535))
}

func normalColor(n Vec3) color.RGBA64 {
	return color.RGBA64{
		R: clampColor(n.X*0.5 + 0.5),
		G: clampColor(n.Y*0.5 + 0.5),
		B: clampColor(n.Z*0.5 + 0.5),
		A: 65535,
	}
}

func mixChannel(a, b, t float64) float64 {
	return a + (b-a)*t
}

func heatColor(t float64) color.RGBA64 {
	t = math.Max(0, math.Min(1, t))
	stops := [][3]float64{
		{0.04, 0.09, 0.35},
		{0.02, 0.55, 0.82},
		{0.18, 0.78, 0.34},
		{0.98, 0.82, 0.16},
		{0.86, 0.12, 0.08},
	}
	scaled := t * float64(len(stops)-1)
	index := int(math.Floor(scaled))
	if index >= len(stops)-1 {
		index = len(stops) - 2
	}
	local := scaled - float64(index)
	lo, hi := stops[index], stops[index+1]
	return color.RGBA64{
		R: clampColor(mixChannel(lo[0], hi[0], local)),
		G: clampColor(mixChannel(lo[1], hi[1], local)),
		B: clampColor(mixChannel(lo[2], hi[2], local)),
		A: 65535,
	}
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func renderNormalSphere(samples [][]SphereSample, compressed bool) *image.RGBA64 {
	size := len(samples)
	img := image.NewRGBA64(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sample := samples[y][x]
			if !sample.Inside {
				img.Set(x, y, color.RGBA64{A: 65535})
				continue
			}
			normal := sample.Original
			if compressed {
				normal = sample.Compressed
			}
			img.Set(x, y, normalColor(normal))
		}
	}
	return img
}

func renderErrorHeatmap(samples [][]SphereSample, maxError float64) *image.RGBA64 {
	size := len(samples)
	img := image.NewRGBA64(image.Rect(0, 0, size, size))
	if maxError == 0 {
		maxError = 1
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sample := samples[y][x]
			if !sample.Inside {
				img.Set(x, y, color.RGBA64{
					R: 800, G: 800, B: 1000, A: 65535,
				})
				continue
			}
			img.Set(x, y, heatColor(sample.AngularErr/maxError))
		}
	}
	return img
}

func shadeNormal(n Vec3) Vec3 {
	light := Normalize(Vec3{X: -0.45, Y: 0.65, Z: 0.8})
	view := Vec3{Z: 1}
	diffuse := math.Max(0, Dot(n, light))
	halfVector := Normalize(Vec3{
		X: light.X + view.X,
		Y: light.Y + view.Y,
		Z: light.Z + view.Z,
	})
	specular := math.Pow(math.Max(0, Dot(n, halfVector)), 42)
	intensity := 0.08 + 0.82*diffuse + 0.65*specular
	base := Vec3{X: 0.22, Y: 0.46, Z: 0.78}
	return Vec3{
		X: base.X * intensity,
		Y: base.Y * intensity,
		Z: base.Z * intensity,
	}
}

func renderLightingComparison(samples [][]SphereSample) *image.RGBA64 {
	size := len(samples)
	img := image.NewRGBA64(image.Rect(0, 0, size*2, size))
	background := color.RGBA64{R: 2000, G: 2000, B: 2600, A: 65535}

	for y := 0; y < size; y++ {
		for x := 0; x < size*2; x++ {
			img.Set(x, y, background)
		}
		img.Set(size, y, color.RGBA64{R: 65535, G: 65535, B: 65535, A: 65535})
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sample := samples[y][x]
			if !sample.Inside {
				continue
			}
			original := shadeNormal(sample.Original)
			compressed := shadeNormal(sample.Compressed)
			img.Set(x, y, color.RGBA64{
				R: clampColor(original.X), G: clampColor(original.Y),
				B: clampColor(original.Z), A: 65535,
			})
			img.Set(size+1+x, y, color.RGBA64{
				R: clampColor(compressed.X), G: clampColor(compressed.Y),
				B: clampColor(compressed.Z), A: 65535,
			})
		}
	}
	return img
}

func renderPackedChannels(samples [][]SphereSample, bits int) *image.RGBA64 {
	size := len(samples)
	img := image.NewRGBA64(image.Rect(0, 0, size, size))
	scale := uint16(1 << (16 - bits))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sample := samples[y][x]
			if !sample.Inside {
				img.Set(x, y, color.RGBA64{A: 65535})
				continue
			}
			img.Set(x, y, color.RGBA64{
				R: sample.Packed.U * scale,
				G: sample.Packed.V * scale,
				B: 0,
				A: 65535,
			})
		}
	}
	return img
}
