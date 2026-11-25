package internal

import "math"

// RGBA is a linear-space texel value.
type RGBA struct {
	R, G, B, A float64
}

// Texture is a full mip pyramid. Levels[0] is the base (size x size) image and
// every subsequent level halves each dimension via a 2x2 box filter.
type Texture struct {
	Levels [][]RGBA
	Size   int
}

func NewTexture(base []RGBA, size int) *Texture {
	t := &Texture{Size: size}
	t.Levels = append(t.Levels, base)
	cur := base
	curSize := size
	for curSize > 1 {
		nextSize := curSize / 2
		if nextSize < 1 {
			break
		}
		next := make([]RGBA, nextSize*nextSize)
		for y := 0; y < nextSize; y++ {
			for x := 0; x < nextSize; x++ {
				sx, sy := 2*x, 2*y
				acc := RGBA{}
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						c := cur[(sy+dy)*curSize+(sx+dx)]
						acc.R += c.R
						acc.G += c.G
						acc.B += c.B
						acc.A += c.A
					}
				}
				next[y*nextSize+x] = RGBA{
					acc.R / 4, acc.G / 4, acc.B / 4, acc.A / 4,
				}
			}
		}
		t.Levels = append(t.Levels, next)
		cur = next
		curSize = nextSize
	}
	return t
}

// MaxLOD returns the index of the coarsest mip level.
func (t *Texture) MaxLOD() float64 {
	return float64(len(t.Levels) - 1)
}

func wrap(v, size int) int {
	v = v % size
	if v < 0 {
		v += size
	}
	return v
}

func sampleLevel(level []RGBA, size int, u, v float64) RGBA {
	fx := u * float64(size)
	fy := v * float64(size)
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := Clamp(fx-float64(x0), 0, 1)
	ty := Clamp(fy-float64(y0), 0, 1)

	at := func(x, y int) RGBA {
		return level[wrap(y, size)*size+wrap(x, size)]
	}
	c00 := at(x0, y0)
	c10 := at(x0+1, y0)
	c01 := at(x0, y0+1)
	c11 := at(x0+1, y0+1)

	blend := func(a, b RGBA, k float64) RGBA {
		return RGBA{
			Lerp(a.R, b.R, k), Lerp(a.G, b.G, k),
			Lerp(a.B, b.B, k), Lerp(a.A, b.A, k),
		}
	}
	top := blend(c00, c10, tx)
	bot := blend(c01, c11, tx)
	return blend(top, bot, ty)
}

// FootprintSample trilinearly filters the texture using the screen-space
// footprint expressed in UV derivatives.
//
// rho is the larger edge of the texel-sized footprint; selecting LOD from
// rho (rather than reusing a fixed primary LOD) is what lets secondary rough
// reflection rays choose coarser levels.
//
// The derivatives are clamped to MaxFootprintTexels before the log: a grazing
// ray at the screen border can otherwise diverge and alias onto an extreme
// level. Clamping keeps the chosen level inside [0, MaxLOD] and continuous.
func (t *Texture) FootprintSample(u, v, dudx, dvdx, dudy, dvdy float64) (RGBA, float64) {
	const maxFootprintTexels = 1 << 13

	clampDeriv := func(d float64) float64 {
		if math.IsNaN(d) || math.IsInf(d, 0) {
			return 0
		}
		return Clamp(math.Abs(d)*float64(t.Size), 0, maxFootprintTexels)
	}
	rhoX := math.Hypot(clampDeriv(dudx), clampDeriv(dvdx))
	rhoY := math.Hypot(clampDeriv(dudy), clampDeriv(dvdy))
	rho := math.Max(rhoX, rhoY)

	lod := 0.0
	if rho > 1e-8 {
		lod = math.Log2(rho)
	}
	lod = Clamp(lod, 0, t.MaxLOD())

	lo := int(math.Floor(lod))
	hi := lo + 1
	if hi > len(t.Levels)-1 {
		hi = lo
	}
	frac := lod - float64(lo)

	c0 := sampleLevel(t.Levels[lo], t.Size>>lo, u, v)
	c1 := sampleLevel(t.Levels[hi], t.Size>>hi, u, v)
	return RGBA{
		Lerp(c0.R, c1.R, frac),
		Lerp(c0.G, c1.G, frac),
		Lerp(c0.B, c1.B, frac),
		Lerp(c0.A, c1.A, frac),
	}, lod
}
