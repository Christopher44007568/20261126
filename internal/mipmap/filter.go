package mipmap

import "math"

type BoundaryMode int

const (
	BoundaryWrap BoundaryMode = iota
	BoundaryZero
	BoundaryClamp
)

// EnergyReport records the DC/flux correction applied after a mip pass.
type EnergyReport struct {
	Level        int
	Gain         [3]float64
	MeanBefore   [3]float64
	MeanAfter    [3]float64
	MaxGainDrift float64
}

func nextMipSize(size int) int {
	if size <= 1 {
		return 1
	}
	return (size + 1) / 2
}

// DownsampleWrap performs a normalized 4x4 periodic filter followed by 2x
// decimation. The taps at destination x are parent columns 2x-1 ... 2x+2,
// so the tile boundary is genuinely included in a wrapped footprint.
func DownsampleWrap(src *RGBImage, level int) (*RGBImage, EnergyReport) {
	dst := downsampleFourTap(src, BoundaryWrap)
	reference := src.Mean()
	report := NormalizeEnergy(dst, reference, level)
	return dst, report
}

// DownsampleNaiveZero uses the same footprint but treats out-of-image taps as
// zero while retaining the full kernel denominator. This reproduces the
// progressive black seam bug.
func DownsampleNaiveZero(src *RGBImage) *RGBImage {
	return downsampleFourTap(src, BoundaryZero)
}

func DownsampleClampRenormalized(src *RGBImage) *RGBImage {
	return downsampleFourTapRenormalized(src, BoundaryClamp)
}

func downsampleFourTap(src *RGBImage, mode BoundaryMode) *RGBImage {
	return downsampleFourTapInternal(src, mode, false)
}

func downsampleFourTapRenormalized(src *RGBImage, mode BoundaryMode) *RGBImage {
	return downsampleFourTapInternal(src, mode, true)
}

func downsampleFourTapInternal(src *RGBImage, mode BoundaryMode, renormalize bool) *RGBImage {
	dstWidth := nextMipSize(src.Width)
	dstHeight := nextMipSize(src.Height)
	dst := NewRGBImage(dstWidth, dstHeight)
	offsets := [4]int{-1, 0, 1, 2}
	weight := 1.0 / 4.0

	for y := 0; y < dstHeight; y++ {
		for x := 0; x < dstWidth; x++ {
			var sum [3]float64
			var mass float64
			for _, oy := range offsets {
				sy := 2*y + oy
				for _, ox := range offsets {
					sx := 2*x + ox
					r, g, b, ok := sampleBoundary(src, sx, sy, mode)
					w := weight * weight
					if ok || mode == BoundaryWrap || mode == BoundaryClamp {
						sum[0] += r * w
						sum[1] += g * w
						sum[2] += b * w
						mass += w
					} else if !renormalize {
						mass += w
					}
				}
			}
			denominator := 1.0
			if renormalize {
				denominator = mass
			}
			if denominator == 0 {
				denominator = 1
			}
			dst.Set(x, y, sum[0]/denominator, sum[1]/denominator, sum[2]/denominator)
		}
	}
	return dst
}

func sampleBoundary(img *RGBImage, x, y int, mode BoundaryMode) (r, g, b float64, ok bool) {
	inBounds := x >= 0 && x < img.Width && y >= 0 && y < img.Height
	switch mode {
	case BoundaryWrap:
		r, g, b = img.WrapRGB(x, y)
		return r, g, b, true
	case BoundaryClamp:
		x = clampInt(x, 0, img.Width-1)
		y = clampInt(y, 0, img.Height-1)
		r, g, b = img.RGB(x, y)
		return r, g, b, true
	default:
		if !inBounds {
			return 0, 0, 0, false
		}
		r, g, b = img.RGB(x, y)
		return r, g, b, true
	}
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// NormalizeEnergy restores unit DC/flux mass after resampling. The wrapped
// kernel already has unit coefficient mass; this is also the numerical and
// odd-size conservation step. It is a per-channel gain, not a local seam fix.
func NormalizeEnergy(img *RGBImage, target [3]float64, level int) EnergyReport {
	before := img.Mean()
	gain := [3]float64{1, 1, 1}
	maxDrift := 0.0
	for channel := 0; channel < 3; channel++ {
		if math.Abs(before[channel]) > 1e-12 {
			gain[channel] = target[channel] / before[channel]
		} else if math.Abs(target[channel]) > 1e-12 {
			gain[channel] = 1
		}
		drift := math.Abs(gain[channel] - 1)
		if drift > maxDrift {
			maxDrift = drift
		}
	}
	for i := 0; i < len(img.Pix); i += 3 {
		img.Pix[i+0] *= gain[0]
		img.Pix[i+1] *= gain[1]
		img.Pix[i+2] *= gain[2]
	}
	after := img.Mean()
	return EnergyReport{
		Level:        level,
		Gain:         gain,
		MeanBefore:   before,
		MeanAfter:    after,
		MaxGainDrift: maxDrift,
	}
}

// NormalizeFlux is the graphics/lighting synonym for NormalizeEnergy.
func NormalizeFlux(img *RGBImage, target [3]float64, level int) EnergyReport {
	return NormalizeEnergy(img, target, level)
}

func GenerateWrapMipChain(base *RGBImage) ([]*RGBImage, []EnergyReport) {
	levels := []*RGBImage{base.Clone()}
	reports := []EnergyReport{{Level: 0, Gain: [3]float64{1, 1, 1}, MeanAfter: base.Mean()}}
	for levels[len(levels)-1].Width != 1 || levels[len(levels)-1].Height != 1 {
		next, report := DownsampleWrap(levels[len(levels)-1], len(levels))
		levels = append(levels, next)
		reports = append(reports, report)
	}
	return levels, reports
}

func GenerateNaiveMipChain(base *RGBImage) []*RGBImage {
	levels := []*RGBImage{base.Clone()}
	for levels[len(levels)-1].Width != 1 || levels[len(levels)-1].Height != 1 {
		levels = append(levels, DownsampleNaiveZero(levels[len(levels)-1]))
	}
	return levels
}

func GenerateClampMipChain(base *RGBImage) []*RGBImage {
	levels := []*RGBImage{base.Clone()}
	for levels[len(levels)-1].Width != 1 || levels[len(levels)-1].Height != 1 {
		levels = append(levels, DownsampleClampRenormalized(levels[len(levels)-1]))
	}
	return levels
}

func DownsampleInsideRenormalized(src *RGBImage) *RGBImage {
	return downsampleFourTapRenormalized(src, BoundaryZero)
}

func GenerateInsideMipChain(base *RGBImage) []*RGBImage {
	levels := []*RGBImage{base.Clone()}
	for levels[len(levels)-1].Width != 1 || levels[len(levels)-1].Height != 1 {
		levels = append(levels, DownsampleInsideRenormalized(levels[len(levels)-1]))
	}
	return levels
}
