package main

import (
	"math"
	"math/rand"
)

const imgW = 128
const imgH = 128

// SyntheticIrradiance builds a deterministic grayscale linear irradiance field.
// Values span roughly five orders of magnitude so every exposure clips or goes
// black somewhere, which exercises saturated and under-exposed pixels.
func SyntheticIrradiance(seed int64) []float64 {
	rng := rand.New(rand.NewSource(seed))
	e := make([]float64, imgW*imgH)
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			u := float64(x) / float64(imgW-1)
			v := float64(y) / float64(imgH-1)
			logE := -3.0 + 6.0*u // 1e-3 .. 1e3
			blobA := 7.0 * math.Exp(-((u-0.30)*(u-0.30)+(v-0.35)*(v-0.35))/0.010)
			blobB := 4.0 * math.Exp(-((u-0.72)*(u-0.72)+(v-0.68)*(v-0.68))/0.020)
			value := math.Exp(math.Log(0.05) + logE) // 5e-5 .. 50
			value *= math.Exp(blobA + blobB)
			if x < imgW/10 { // a bright vertical bar
				value *= 8.0
			}
			value *= 0.97 + 0.06*rng.Float64()
			e[y*imgW+x] = value
		}
	}
	return e
}

// TrueResponse is the known camera response used to fabricate exposures.
// It maps log exposure z = ln(E * t) to an 8-bit value.
func TrueResponse(logExposure float64) int {
	y := 255.0 * math.Pow(clamp01(0.5+0.18*logExposure), 1.0/2.2)
	return int(math.Round(y))
}

// InverseTrueResponse returns g_true(z) = ln(E*t) implied by TrueResponse.
func InverseTrueResponse(z int) float64 {
	y := float64(z) / 255.0
	return (math.Pow(y, 2.2) - 0.5) / 0.18
}

// ExposureSet describes a geometric sequence of exposure times.
func ExposureSet() []float64 {
	times := make([]float64, 9)
	for k := range times {
		times[k] = math.Exp(float64(k-4) * math.Ln2) // 1/16 .. 16 s
	}
	return times
}

// ShootExposures renders the grayscale exposure strip images with the true CRF.
// noiseSigma adds read noise; glitch injects a calibration defect that makes the
// recovered response locally non-monotonic.
func ShootExposures(irradiance, times []float64, noiseSigma float64, glitch bool, seed int64) []*Gray8 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]*Gray8, len(times))
	for j, t := range times {
		img := NewGray8(imgW, imgH)
		for i, e := range irradiance {
			z := TrueResponse(math.Log(e * t))
			if glitch {
				// A firmware defect remaps a midtone band downward.
				if z >= 148 && z <= 168 {
					z -= 18
				}
			}
			if noiseSigma > 0 {
				z += int(math.Round(rng.NormFloat64() * noiseSigma))
			}
			if z < 0 {
				z = 0
			} else if z > 255 {
				z = 255
			}
			img.Pix[i] = uint8(z)
		}
		out[j] = img
	}
	return out
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
