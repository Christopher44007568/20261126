package main

import "math"

type SphereSample struct {
	Original   Vec3
	Compressed Vec3
	Packed     PackedNormal
	AngularErr float64
	Inside     bool
}

type SphereMetrics struct {
	Samples     int
	MeanRadians float64
	MaxRadians  float64
	MaxSample   SphereSample
}

func sampleSphereAt(x, y, size int, bits int) SphereSample {
	center := (float64(size) - 1) / 2
	radius := math.Floor(float64(size)*0.47 + 0.5)
	dx := (float64(x) - center) / radius
	dy := (center - float64(y)) / radius
	radial := math.Sqrt(dx*dx + dy*dy)

	sample := SphereSample{Inside: radial <= 1}
	if !sample.Inside {
		return sample
	}

	if radial == 0 {
		sample.Original = Vec3{Z: 1}
		sample.Packed = PackNormal(sample.Original, bits)
		sample.Compressed = sample.Packed.Unpack(bits)
		sample.AngularErr = math.Acos(math.Max(-1, math.Min(1, Dot(sample.Original, sample.Compressed))))
		return sample
	}

	theta := radial * math.Pi
	sinTheta := math.Sin(theta)
	normal := Normalize(Vec3{
		X: sinTheta * dx / radial,
		Y: sinTheta * dy / radial,
		Z: math.Cos(theta),
	})

	packed := PackNormal(normal, bits)
	reconstructed := packed.Unpack(bits)
	cosAngle := math.Max(-1, math.Min(1, Dot(normal, reconstructed)))

	sample.Original = normal
	sample.Compressed = reconstructed
	sample.Packed = packed
	sample.AngularErr = math.Acos(cosAngle)
	return sample
}

func BuildSphere(size, bits int) ([][]SphereSample, SphereMetrics) {
	samples := make([][]SphereSample, size)
	metrics := SphereMetrics{}

	for y := 0; y < size; y++ {
		samples[y] = make([]SphereSample, size)
		for x := 0; x < size; x++ {
			sample := sampleSphereAt(x, y, size, bits)
			samples[y][x] = sample
			if !sample.Inside {
				continue
			}

			metrics.Samples++
			metrics.MeanRadians += sample.AngularErr
			if sample.AngularErr > metrics.MaxRadians {
				metrics.MaxRadians = sample.AngularErr
				metrics.MaxSample = sample
			}
		}
	}

	if metrics.Samples > 0 {
		metrics.MeanRadians /= float64(metrics.Samples)
	}
	return samples, metrics
}
