package main

import (
	"image"
	"math"
	"sort"
)

type Corner struct {
	Pos   Vec2
	Score float64
}

func gaussianBlurGray(src []float64, w, h int, sigma float64) []float64 {
	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float64, radius*2+1)
	sum := 0.0
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		kernel[i+radius] = v
		sum += v
	}
	for i := range kernel {
		kernel[i] /= sum
	}
	tmp := make([]float64, w*h)
	dst := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := 0.0
			for k := -radius; k <= radius; k++ {
				xx := clampInt(x+k, 0, w-1)
				v += src[y*w+xx] * kernel[k+radius]
			}
			tmp[y*w+x] = v
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := 0.0
			for k := -radius; k <= radius; k++ {
				yy := clampInt(y+k, 0, h-1)
				v += tmp[yy*w+x] * kernel[k+radius]
			}
			dst[y*w+x] = v
		}
	}
	return dst
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

func sampleGray(g []float64, w, h int, x, y float64) float64 {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx, ty := x-float64(x0), y-float64(y0)
	x0 = clampInt(x0, 0, w-1)
	y0 = clampInt(y0, 0, h-1)
	x1 := clampInt(x0+1, 0, w-1)
	y1 := clampInt(y0+1, 0, h-1)
	a := g[y0*w+x0]*(1-tx) + g[y0*w+x1]*tx
	b := g[y1*w+x0]*(1-tx) + g[y1*w+x1]*tx
	return a*(1-ty) + b*ty
}

func detectHarrisCorners(img *image.RGBA) ([]Corner, []float64) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	gray := gaussianBlurGray(ToGray(img), w, h, 1.15)
	ix2 := make([]float64, w*h)
	iy2 := make([]float64, w*h)
	ixy := make([]float64, w*h)
	maxResp := 0.0
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := (sampleGray(gray, w, h, float64(x)+1, float64(y)) -
				sampleGray(gray, w, h, float64(x)-1, float64(y))) * 0.5
			gy := (sampleGray(gray, w, h, float64(x), float64(y)+1) -
				sampleGray(gray, w, h, float64(x), float64(y)-1)) * 0.5
			i := y*w + x
			ix2[i], iy2[i], ixy[i] = gx*gx, gy*gy, gx*gy
		}
	}
	a := boxBlur(ix2, w, h, 2)
	cc := boxBlur(iy2, w, h, 2)
	bb := boxBlur(ixy, w, h, 2)
	resp := make([]float64, w*h)
	for i := range resp {
		r := a[i]*cc[i] - bb[i]*bb[i] - 0.06*math.Pow(a[i]+cc[i], 2)
		if r > 0 {
			resp[i] = r
			maxResp = math.Max(maxResp, r)
		}
	}
	if maxResp <= 0 {
		return nil, gray
	}
	var raw []Corner
	nms := 6
	threshold := maxResp * 0.035
	for y := nms + 1; y < h-nms-1; y++ {
		for x := nms + 1; x < w-nms-1; x++ {
			i := y*w + x
			if resp[i] < threshold {
				continue
			}
			localMax := true
			for dy := -nms; dy <= nms && localMax; dy++ {
				for dx := -nms; dx <= nms; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if resp[(y+dy)*w+x+dx] > resp[i] {
						localMax = false
						break
					}
				}
			}
			if localMax {
				p := refineCorner(gray, w, h, Vec2{X: float64(x), Y: float64(y)})
				raw = append(raw, Corner{Pos: p, Score: resp[i]})
			}
		}
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].Score > raw[j].Score })
	var kept []Corner
	for _, c := range raw {
		ok := true
		for _, k := range kept {
			if math.Hypot(c.Pos.X-k.Pos.X, c.Pos.Y-k.Pos.Y) < 16 {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, c)
		}
		if len(kept) > 500 {
			break
		}
	}
	return kept, gray
}

func filterInnerCheckerCorners(corners []Corner, gray []float64, w, h int, spacing float64) []Corner {
	radius := math.Max(5, spacing*0.22)
	var out []Corner
	for _, c := range corners {
		if checkerQuadrantScore(gray, w, h, c.Pos, radius) >= 42 {
			out = append(out, c)
		}
	}
	if len(out) < 8 {
		return corners
	}
	return out
}

func checkerQuadrantScore(gray []float64, w, h int, p Vec2, radius float64) float64 {
	best := 0.0
	for step := 0; step < 18; step++ {
		angle := float64(step) * math.Pi / 36
		var v [4]float64
		for k := 0; k < 4; k++ {
			a := angle + float64(k)*math.Pi/2
			v[k] = sampleGray(gray, w, h, p.X+radius*math.Cos(a), p.Y+radius*math.Sin(a))
		}
		contrast := math.Abs(v[0]-v[1]) + math.Abs(v[1]-v[2]) +
			math.Abs(v[2]-v[3]) + math.Abs(v[3]-v[0])
		opposite := math.Abs(v[0]-v[2]) + math.Abs(v[1]-v[3])
		score := contrast/4 - opposite*0.35
		best = math.Max(best, score)
	}
	return best
}

func boxBlur(src []float64, w, h, radius int) []float64 {
	tmp := make([]float64, w*h)
	dst := make([]float64, w*h)
	win := float64(2*radius + 1)
	for y := 0; y < h; y++ {
		sum := 0.0
		for x := -radius; x <= radius; x++ {
			sum += src[y*w+clampInt(x, 0, w-1)]
		}
		for x := 0; x < w; x++ {
			tmp[y*w+x] = sum / win
			xOut := clampInt(x-radius, 0, w-1)
			xIn := clampInt(x+radius+1, 0, w-1)
			sum += src[y*w+xIn] - src[y*w+xOut]
		}
	}
	for x := 0; x < w; x++ {
		sum := 0.0
		for y := -radius; y <= radius; y++ {
			sum += tmp[clampInt(y, 0, h-1)*w+x]
		}
		for y := 0; y < h; y++ {
			dst[y*w+x] = sum / win
			yOut := clampInt(y-radius, 0, h-1)
			yIn := clampInt(y+radius+1, 0, h-1)
			sum += tmp[yIn*w+x] - tmp[yOut*w+x]
		}
	}
	return dst
}

func refineCorner(gray []float64, w, h int, start Vec2) Vec2 {
	center := start
	radius := 6
	for iter := 0; iter < 4; iter++ {
		var a00, a01, a11, b0, b1 float64
		cx := int(math.Round(center.X))
		cy := int(math.Round(center.Y))
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				qx := float64(clampInt(cx+dx, 1, w-2))
				qy := float64(clampInt(cy+dy, 1, h-2))
				gx := (sampleGray(gray, w, h, qx+1, qy) - sampleGray(gray, w, h, qx-1, qy)) * 0.5
				gy := (sampleGray(gray, w, h, qx, qy+1) - sampleGray(gray, w, h, qx, qy-1)) * 0.5
				weight := math.Exp(-float64(dx*dx+dy*dy) / 18)
				gxx, gxy, gyy := gx*gx*weight, gx*gy*weight, gy*gy*weight
				a00 += gxx
				a01 += gxy
				a11 += gyy
				b0 += gxx*qx + gxy*qy
				b1 += gxy*qx + gyy*qy
			}
		}
		det := a00*a11 - a01*a01
		if math.Abs(det) < 1e-8 {
			break
		}
		next := Vec2{
			X: (a11*b0 - a01*b1) / det,
			Y: (a00*b1 - a01*b0) / det,
		}
		if math.Hypot(next.X-center.X, next.Y-center.Y) > 2.5 {
			break
		}
		if math.Hypot(next.X-center.X, next.Y-center.Y) < 0.01 {
			center = next
			break
		}
		center = next
	}
	center.X = math.Max(0, math.Min(float64(w-1), center.X))
	center.Y = math.Max(0, math.Min(float64(h-1), center.Y))
	return center
}
