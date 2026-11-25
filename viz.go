package main

import "math"

// Metrics holds quality statistics for a reconstruction.
type Metrics struct {
	PSNR         float64
	MeanBlockErr float64 // mean per-block RMSE (0..255)
	MaxBlockErr  float64 // worst per-block RMSE
	FlatErr      float64 // mean RMSE of the 25% flattest blocks
	EdgeErr      float64 // mean RMSE of the 25% most active blocks
}

func msePSNR(mse float64) float64 {
	if mse <= 0 {
		return 99
	}
	return 10 * math.Log10(255*255/mse)
}

// blockStats computes per-8x8-block RMSE between a and b, plus activity of a.
func blockStats(a, b *Image) (errs, act []float64, bw, bh int) {
	bw, bh = (a.W+7)/8, (a.H+7)/8
	errs = make([]float64, bw*bh)
	act = make([]float64, bw*bh)
	maxA := 1e-9
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			var se, sum, sum2 float64
			n := 0
			for j := 0; j < 8; j++ {
				for i := 0; i < 8; i++ {
					x, y := bx*8+i, by*8+j
					if x >= a.W || y >= a.H {
						continue
					}
					idx := (y*a.W + x) * 3
					dr := (a.Pix[idx] - b.Pix[idx]) * 255
					dg := (a.Pix[idx+1] - b.Pix[idx+1]) * 255
					db := (a.Pix[idx+2] - b.Pix[idx+2]) * 255
					se += (dr*dr + dg*dg + db*db) / 3
					lum := 0.299*a.Pix[idx] + 0.587*a.Pix[idx+1] + 0.114*a.Pix[idx+2]
					sum += lum
					sum2 += lum * lum
					n++
				}
			}
			errs[by*bw+bx] = math.Sqrt(se / float64(n))
			mean := sum / float64(n)
			v := sum2/float64(n) - mean*mean
			if v > maxA {
				maxA = v
			}
			act[by*bw+bx] = v
		}
	}
	for i := range act {
		act[i] /= maxA
	}
	return
}

// ComputeMetrics compares orig and recon.
func ComputeMetrics(orig, recon *Image) Metrics {
	var se float64
	for i := range orig.Pix {
		d := (orig.Pix[i] - recon.Pix[i]) * 255
		se += d * d
	}
	m := Metrics{PSNR: msePSNR(se / float64(len(orig.Pix)))}
	errs, act, bw, bh := blockStats(orig, recon)
	n := bw * bh
	var sum, max float64
	var flatSum, edgeSum float64
	var flatN, edgeN int
	for i := 0; i < n; i++ {
		sum += errs[i]
		if errs[i] > max {
			max = errs[i]
		}
		if act[i] <= 0.05 {
			flatSum += errs[i]
			flatN++
		}
		if act[i] >= 0.30 {
			edgeSum += errs[i]
			edgeN++
		}
	}
	m.MeanBlockErr = sum / float64(n)
	m.MaxBlockErr = max
	if flatN > 0 {
		m.FlatErr = flatSum / float64(flatN)
	}
	if edgeN > 0 {
		m.EdgeErr = edgeSum / float64(edgeN)
	}
	return m
}

// heatColor maps t in [0,1] to a black->purple->red->yellow->white ramp.
func heatColor(t float64) (float64, float64, float64) {
	t = clamp01(t)
	switch {
	case t < 0.25:
		s := t / 0.25
		return 0.35 * s, 0, 0.45 * s
	case t < 0.5:
		s := (t - 0.25) / 0.25
		return 0.35 + 0.65*s, 0.05 * s, 0.45 * (1 - s)
	case t < 0.75:
		s := (t - 0.5) / 0.25
		return 1, 0.05 + 0.75*s, 0
	default:
		s := (t - 0.75) / 0.25
		return 1, 0.8 + 0.2*s, 0.6 * s
	}
}

// Heatmap renders per-pixel squared-error magnitude as a color ramp.
// Errors are normalized so that `fullScale` (0..255) maps to white.
func Heatmap(orig, recon *Image, fullScale float64) *Image {
	out := NewImage(orig.W, orig.H)
	for i := 0; i < orig.W*orig.H; i++ {
		dr := (orig.Pix[i*3] - recon.Pix[i*3]) * 255
		dg := (orig.Pix[i*3+1] - recon.Pix[i*3+1]) * 255
		db := (orig.Pix[i*3+2] - recon.Pix[i*3+2]) * 255
		e := math.Sqrt((dr*dr + dg*dg + db*db) / 3)
		r, g, b := heatColor(e / fullScale)
		out.Pix[i*3] = r
		out.Pix[i*3+1] = g
		out.Pix[i*3+2] = b
	}
	return out
}

// BlockMap overlays the 8x8 block grid on the reconstruction; grid line
// color encodes each block's RMSE (green = low, red = high).
func BlockMap(orig, recon *Image) *Image {
	out := recon.Clone()
	errs, _, bw, bh := blockStats(orig, recon)
	maxE := 1e-9
	for _, e := range errs {
		if e > maxE {
			maxE = e
		}
	}
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			t := errs[by*bw+bx] / maxE
			r, g, b := 0.2+0.8*t, 0.85-0.65*t, 0.2
			for k := 0; k < 8; k++ {
				if x := bx*8 + k; x < out.W && by*8 < out.H {
					out.Set(x, by*8, r, g, b)
				}
				if y := by*8 + k; y < out.H && bx*8 < out.W {
					out.Set(bx*8, y, r, g, b)
				}
			}
		}
	}
	return out
}
