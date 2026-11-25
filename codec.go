package main

import "math"

// Standard JPEG quantization tables.
var qLuma = [64]int{
	16, 11, 10, 16, 24, 40, 51, 61,
	12, 12, 14, 19, 26, 58, 60, 55,
	14, 13, 16, 24, 40, 57, 69, 56,
	14, 17, 22, 29, 51, 87, 80, 62,
	18, 22, 37, 56, 68, 109, 103, 77,
	24, 35, 55, 64, 81, 104, 113, 92,
	49, 64, 78, 87, 103, 121, 120, 101,
	72, 92, 95, 98, 112, 100, 103, 99,
}

var qChroma = [64]int{
	17, 18, 24, 47, 99, 99, 99, 99,
	18, 21, 26, 66, 99, 99, 99, 99,
	24, 26, 56, 99, 99, 99, 99, 99,
	47, 66, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
}

var cosTab [8][8]float64

func init() {
	for u := 0; u < 8; u++ {
		for x := 0; x < 8; x++ {
			cosTab[u][x] = math.Cos((2*float64(x) + 1) * float64(u) * math.Pi / 16)
		}
	}
}

func dct1d(in, out []float64) {
	for u := 0; u < 8; u++ {
		s := 0.0
		for x := 0; x < 8; x++ {
			s += in[x] * cosTab[u][x]
		}
		cu := 1.0
		if u == 0 {
			cu = 1.0 / math.Sqrt2
		}
		out[u] = 0.5 * cu * s
	}
}

func idct1d(in, out []float64) {
	for x := 0; x < 8; x++ {
		s := 0.0
		for u := 0; u < 8; u++ {
			cu := 1.0
			if u == 0 {
				cu = 1.0 / math.Sqrt2
			}
			s += cu * in[u] * cosTab[u][x]
		}
		out[x] = 0.5 * s
	}
}

func dct2(blk *[64]float64) {
	var tmp, rowIn, rowOut [64]float64
	for y := 0; y < 8; y++ {
		dct1d(blk[y*8:y*8+8], rowOut[y*8:y*8+8])
	}
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			rowIn[y] = rowOut[y*8+x]
		}
		var col [8]float64
		dct1d(rowIn[:], col[:])
		for y := 0; y < 8; y++ {
			tmp[y*8+x] = col[y]
		}
	}
	*blk = tmp
}

func idct2(blk *[64]float64) {
	var tmp, colIn, colOut [64]float64
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			colIn[y] = blk[y*8+x]
		}
		var col [8]float64
		idct1d(colIn[:], col[:])
		for y := 0; y < 8; y++ {
			tmp[y*8+x] = col[y]
		}
	}
	for y := 0; y < 8; y++ {
		idct1d(tmp[y*8:y*8+8], colOut[y*8:y*8+8])
	}
	*blk = colOut
}

func qualityScale(q int) float64 {
	if q < 1 {
		q = 1
	}
	if q > 100 {
		q = 100
	}
	if q < 50 {
		return 5000.0 / float64(q)
	}
	return 200.0 - 2.0*float64(q)
}

// Codec compresses an image with an 8x8 DCT block codec (YCbCr 4:2:0).
type Codec struct {
	Quality  int
	Adaptive bool // perceptual bit allocation: protect active/highlight blocks
	EstBPP   float64
}

type plane struct {
	w, h int
	data []float64
}

func (p *plane) at(x, y int) float64 {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= p.w {
		x = p.w - 1
	}
	if y >= p.h {
		y = p.h - 1
	}
	return p.data[y*p.w+x]
}

func rgbToPlanes(src *Image) (y, cb, cr *plane) {
	y = &plane{w: src.W, h: src.H, data: make([]float64, src.W*src.H)}
	cb = &plane{w: src.W, h: src.H, data: make([]float64, src.W*src.H)}
	cr = &plane{w: src.W, h: src.H, data: make([]float64, src.W*src.H)}
	for i := 0; i < src.W*src.H; i++ {
		r := src.Pix[i*3] * 255
		g := src.Pix[i*3+1] * 255
		b := src.Pix[i*3+2] * 255
		y.data[i] = 0.299*r + 0.587*g + 0.114*b
		cb.data[i] = -0.168736*r - 0.331264*g + 0.5*b + 128
		cr.data[i] = 0.5*r - 0.418688*g - 0.081312*b + 128
	}
	return
}

func downsample2x(p *plane) *plane {
	w, h := (p.w+1)/2, (p.h+1)/2
	out := &plane{w: w, h: h, data: make([]float64, w*h)}
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			s := p.at(2*xx, 2*yy) + p.at(2*xx+1, 2*yy) +
				p.at(2*xx, 2*yy+1) + p.at(2*xx+1, 2*yy+1)
			out.data[yy*w+xx] = s / 4
		}
	}
	return out
}

// blockActivity returns per-8x8-block luma activity (stddev + gradient energy),
// normalized to [0,1]. Highlights and textured regions score high.
func blockActivity(y *plane) []float64 {
	bw, bh := (y.w+7)/8, (y.h+7)/8
	act := make([]float64, bw*bh)
	maxA := 1e-9
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			var sum, sum2, grad float64
			for j := 0; j < 8; j++ {
				for i := 0; i < 8; i++ {
					v := y.at(bx*8+i, by*8+j)
					sum += v
					sum2 += v * v
					gx := y.at(bx*8+i+1, by*8+j) - v
					gy := y.at(bx*8+i, by*8+j+1) - v
					grad += gx*gx + gy*gy
				}
			}
			mean := sum / 64
			variance := sum2/64 - mean*mean
			a := math.Sqrt(math.Max(variance, 0)) + 0.05*math.Sqrt(grad/64)
			act[by*bw+bx] = a
			if a > maxA {
				maxA = a
			}
		}
	}
	for i := range act {
		act[i] /= maxA
	}
	return act
}

type entropyCounter struct {
	hist  map[int]int
	total int
}

func (e *entropyCounter) add(v int) {
	if e.hist == nil {
		e.hist = map[int]int{}
	}
	e.hist[v]++
	e.total++
}

func (e *entropyCounter) bits() float64 {
	b := 0.0
	for _, n := range e.hist {
		p := float64(n) / float64(e.total)
		b -= float64(n) * math.Log2(p)
	}
	return b
}

// quantizeBlock runs DCT -> quantize -> dequantize -> IDCT on one 8x8 block
// sampled from p at (bx*8, by*8). factor scales the quant step (<1 = finer).
// Returns quantized coefficients for entropy estimation.
func quantizeBlock(p *plane, bx, by int, table *[64]int, scale, factor float64, out []float64, stride int) [64]int {
	var blk [64]float64
	for j := 0; j < 8; j++ {
		for i := 0; i < 8; i++ {
			blk[j*8+i] = p.at(bx*8+i, by*8+j) - 128
		}
	}
	dct2(&blk)
	var q [64]int
	for k := 0; k < 64; k++ {
		step := float64(table[k]) * scale / 100 * factor
		if step < 1 {
			step = 1
		}
		qv := int(math.Round(blk[k] / step))
		q[k] = qv
		blk[k] = float64(qv) * step
	}
	idct2(&blk)
	for j := 0; j < 8; j++ {
		for i := 0; i < 8; i++ {
			x, y := bx*8+i, by*8+j
			out[y*stride+x] = blk[j*8+i] + 128
		}
	}
	return q
}

// Compress runs the block codec and returns the reconstructed image.
func (c *Codec) Compress(src *Image) *Image {
	yp, cbp, crp := rgbToPlanes(src)
	cbh, crh := downsample2x(cbp), downsample2x(crp)
	scale := qualityScale(c.Quality)

	var act []float64
	if c.Adaptive {
		act = blockActivity(yp)
	}

	var dcL, acL, dcC, acC entropyCounter
	totalBits := 0.0

	process := func(p *plane, table *[64]int, adaptive bool, dc, ac *entropyCounter) *plane {
		bw, bh := (p.w+7)/8, (p.h+7)/8
		stride := bw * 8
		out := make([]float64, stride*bh*8)
		prevDC := 0
		for by := 0; by < bh; by++ {
			for bx := 0; bx < bw; bx++ {
				factor := 1.0
				if adaptive {
					a := act[by*bw+bx]
					// Shift bits toward active blocks: flat blocks get up to
					// 1.45x coarser steps, highlight/texture blocks up to 5x finer.
					factor = 1.25 - 1.05*a
					if factor < 0.2 {
						factor = 0.2
					}
					if factor > 1.45 {
						factor = 1.45
					}
				}
				q := quantizeBlock(p, bx, by, table, scale, factor, out, stride)
				dc.add(q[0] - prevDC)
				prevDC = q[0]
				for k := 1; k < 64; k++ {
					ac.add(q[k])
				}
			}
		}
		res := &plane{w: p.w, h: p.h, data: make([]float64, p.w*p.h)}
		for yy := 0; yy < p.h; yy++ {
			copy(res.data[yy*p.w:yy*p.w+p.w], out[yy*stride:yy*stride+p.w])
		}
		return res
	}

	yr := process(yp, &qLuma, c.Adaptive, &dcL, &acL)
	cbr := process(cbh, &qChroma, false, &dcC, &acC)
	crr := process(crh, &qChroma, false, &dcC, &acC)

	totalBits = dcL.bits() + acL.bits() + dcC.bits() + acC.bits()
	c.EstBPP = totalBits / float64(src.W*src.H)

	dst := NewImage(src.W, src.H)
	for yy := 0; yy < src.H; yy++ {
		for xx := 0; xx < src.W; xx++ {
			yv := yr.at(xx, yy)
			cbv := cbr.at(xx/2, yy/2) - 128
			crv := crr.at(xx/2, yy/2) - 128
			r := yv + 1.402*crv
			g := yv - 0.344136*cbv - 0.714136*crv
			b := yv + 1.772*cbv
			dst.Set(xx, yy, r/255, g/255, b/255)
		}
	}
	return dst
}
