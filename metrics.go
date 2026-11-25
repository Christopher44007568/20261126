package main

import (
	"fmt"
	"math"
)

type Metrics struct {
	Pixels               int     `json:"pixels"`
	VisiblePixels        int     `json:"visible_pixels"`
	TransparentPixels    int     `json:"transparent_pixels"`
	VisibleRGBMAE        float64 `json:"visible_rgb_mae"`
	RGBPSNR              float64 `json:"rgb_psnr_db"`
	AlphaMAE             float64 `json:"alpha_mae"`
	PremultipliedRGBAMAE float64 `json:"premultiplied_rgba_mae"`
	AlphaClassErrors     int     `json:"alpha_class_errors"`
	HoleFillErrors       int     `json:"hole_fill_errors"`
	HoleErosionErrors    int     `json:"hole_erosion_errors"`
	BoundaryPixels       int     `json:"source_alpha_boundary_pixels"`
	BoundaryVisible      int     `json:"visible_boundary_pixels"`
	BoundaryAlphaMAE     float64 `json:"boundary_alpha_mae"`
	BoundaryRGBMAE       float64 `json:"boundary_rgb_mae"`
	WorstBlockRGBMAE     float64 `json:"worst_block_rgb_mae"`
	WorstBlockX          int     `json:"worst_block_x"`
	WorstBlockY          int     `json:"worst_block_y"`
	RawBytes             int     `json:"raw_bytes"`
	CompressedBytes      int     `json:"compressed_bytes"`
	CompressionRatio     float64 `json:"compression_ratio"`
}

func CompareMetrics(orig, decoded *Bitmap) (Metrics, error) {
	if orig.W != decoded.W || orig.H != decoded.H {
		return Metrics{}, fmt.Errorf("size mismatch: %dx%d vs %dx%d",
			orig.W, orig.H, decoded.W, decoded.H)
	}
	var m Metrics
	m.Pixels = orig.W * orig.H
	m.RawBytes = m.Pixels * 4
	var rgbAbs, alphaAbs, premulAbs, rgbSquared float64
	for y := 0; y < orig.H; y++ {
		for x := 0; x < orig.W; x++ {
			a, b := orig.At(x, y), decoded.At(x, y)
			if a.A > 0 {
				m.VisiblePixels++
				dr := float64(absU8(a.R, b.R))
				dg := float64(absU8(a.G, b.G))
				db := float64(absU8(a.B, b.B))
				rgbAbs += dr + dg + db
				rgbSquared += dr*dr + dg*dg + db*db
			} else {
				m.TransparentPixels++
			}
			alphaAbs += float64(absU8(a.A, b.A))
			premulAbs += premulDiff(a, b)
			if (a.A == 0) != (b.A == 0) {
				m.AlphaClassErrors++
			}
			if a.A == 0 && b.A > 0 {
				m.HoleFillErrors++
			}
			if a.A > 0 && b.A == 0 {
				m.HoleErosionErrors++
			}
		}
	}
	if m.VisiblePixels > 0 {
		m.VisibleRGBMAE = rgbAbs / float64(m.VisiblePixels*3)
		mse := rgbSquared / float64(m.VisiblePixels*3)
		if mse == 0 {
			m.RGBPSNR = 99
		} else {
			m.RGBPSNR = 10 * math.Log10(255*255/mse)
		}
	}
	m.AlphaMAE = alphaAbs / float64(m.Pixels)
	m.PremultipliedRGBAMAE = premulAbs / float64(m.Pixels*4)

	var boundaryAlpha, boundaryRGB float64
	var boundaryVisible int
	for y := 0; y < orig.H; y++ {
		for x := 0; x < orig.W; x++ {
			if !isAlphaBoundary(orig, x, y) {
				continue
			}
			a, b := orig.At(x, y), decoded.At(x, y)
			m.BoundaryPixels++
			boundaryAlpha += float64(absU8(a.A, b.A))
			if a.A > 0 {
				boundaryVisible++
				boundaryRGB += float64(absU8(a.R, b.R)+absU8(a.G, b.G)+absU8(a.B, b.B)) / 3
			}
		}
	}
	if m.BoundaryPixels > 0 {
		m.BoundaryAlphaMAE = boundaryAlpha / float64(m.BoundaryPixels)
		m.BoundaryVisible = boundaryVisible
	}
	if boundaryVisible > 0 {
		m.BoundaryRGBMAE = boundaryRGB / float64(boundaryVisible)
	}
	m.WorstBlockRGBMAE, m.WorstBlockX, m.WorstBlockY = worstBlock(orig, decoded)
	m.CompressionRatio = float64(m.RawBytes) / float64(maxInt(1, m.CompressedBytes))
	return m, nil
}

func SetCompressedSize(m *Metrics, bytes int) {
	m.CompressedBytes = bytes
	m.CompressionRatio = float64(m.RawBytes) / float64(maxInt(1, bytes))
}

func isAlphaBoundary(im *Bitmap, x, y int) bool {
	center := im.At(x, y).A == 0
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		if x+d[0] < 0 || y+d[1] < 0 || x+d[0] >= im.W || y+d[1] >= im.H {
			continue
		}
		if (im.At(x+d[0], y+d[1]).A == 0) != center {
			return true
		}
	}
	return false
}

func worstBlock(orig, decoded *Bitmap) (float64, int, int) {
	worst, bx, by := 0.0, 0, 0
	for y := 0; y < orig.H; y += BlockSide {
		for x := 0; x < orig.W; x += BlockSide {
			var sum float64
			var count int
			for yy := y; yy < y+BlockSide; yy++ {
				for xx := x; xx < x+BlockSide; xx++ {
					a := orig.At(xx, yy)
					if a.A == 0 {
						continue
					}
					b := decoded.At(xx, yy)
					sum += float64(absU8(a.R, b.R) + absU8(a.G, b.G) + absU8(a.B, b.B))
					count += 3
				}
			}
			mae := 0.0
			if count > 0 {
				mae = sum / float64(count)
			}
			if mae > worst {
				worst, bx, by = mae, x, y
			}
		}
	}
	return worst, bx, by
}

func premulDiff(a, b Pixel) float64 {
	pa := []float64{
		float64(a.R) * float64(a.A) / 255,
		float64(a.G) * float64(a.A) / 255,
		float64(a.B) * float64(a.A) / 255,
		float64(a.A),
	}
	pb := []float64{
		float64(b.R) * float64(b.A) / 255,
		float64(b.G) * float64(b.A) / 255,
		float64(b.B) * float64(b.A) / 255,
		float64(b.A),
	}
	var sum float64
	for i := range pa {
		sum += math.Abs(pa[i] - pb[i])
	}
	return sum
}

func absU8(a, b uint8) int {
	return absInt(int(a) - int(b))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
