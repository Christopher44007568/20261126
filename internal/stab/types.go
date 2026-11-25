package stab

import "math"

// Rect is an axis-aligned integer crop rectangle.
type Rect struct {
	X0, Y0, X1, Y1 int
}

func (r Rect) W() int { return r.X1 - r.X0 }
func (r Rect) H() int { return r.Y1 - r.Y0 }

// Margin is the number of valid pixels between a frame edge and the crop box.
type Margin struct {
	Top, Bottom, Left, Right float64
}

// FrameResult is one stabilized output. Valid is false where the warp exposed
// area not covered by the source frame; those pixels are rendered as holes and
// are never filled from earlier frames.
type FrameResult struct {
	W, H  int
	Pix   []float64 // RGB, packed
	Valid []bool    // per-pixel coverage mask
	Hole  []bool    // explicit hole marker (== !Valid), kept for clarity
	Box   Rect      // per-frame maximal valid crop
	Edge  Margin    // per-edge margin implied by Box
}

func newResult(w, h int) *FrameResult {
	n := w * h
	return &FrameResult{
		W:     w,
		H:     h,
		Pix:   make([]float64, n*3),
		Valid: make([]bool, n),
		Hole:  make([]bool, n),
	}
}

// sampleBilinear reads the current source frame only using pixel-center
// coordinates (pixel (0,0) is centered at (0.5,0.5)). Positions whose 2x2
// neighborhood leaves the frame are reported invalid so callers mark a hole
// instead of clamping or stretching edge pixels across uncovered area.
func sampleBilinear(pix []float64, sw, sh int, cx, cy float64) (r, g, b float64, ok bool) {
	x, y := cx-0.5, cy-0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1, y1 := x0+1, y0+1
	if x0 < 0 || y0 < 0 || x1 >= sw || y1 >= sh {
		return 0, 0, 0, false
	}
	fx, fy := x-float64(x0), y-float64(y0)
	w00 := (1 - fx) * (1 - fy)
	w10 := fx * (1 - fy)
	w01 := (1 - fx) * fy
	w11 := fx * fy
	i00 := (y0*sw + x0) * 3
	i10 := (y0*sw + x1) * 3
	i01 := (y1*sw + x0) * 3
	i11 := (y1*sw + x1) * 3
	r = w00*pix[i00] + w10*pix[i10] + w01*pix[i01] + w11*pix[i11]
	g = w00*pix[i00+1] + w10*pix[i10+1] + w01*pix[i01+1] + w11*pix[i11+1]
	b = w00*pix[i00+2] + w10*pix[i10+2] + w01*pix[i01+2] + w11*pix[i11+2]
	return r, g, b, true
}

// markHole records an uncovered pixel. The renderer draws hatched magenta.
func (fr *FrameResult) markHole(x, y int) {
	i := y*fr.W + x
	fr.Valid[i] = false
	fr.Hole[i] = true
}
