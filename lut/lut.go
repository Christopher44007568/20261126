// Package lut parses Adobe .cube 3D LUT files and applies them with
// trilinear interpolation, honoring DOMAIN_MIN/DOMAIN_MAX and the
// cube channel/storage order (red fastest).
package lut

import "fmt"

// Error is a parse error tied to a 1-based line in the source file.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
	}
	return e.Msg
}

func errf(line int, format string, args ...any) *Error {
	return &Error{Line: line, Msg: fmt.Sprintf(format, args...)}
}

// LUT is a 3D lookup table with per-channel input domains.
type LUT struct {
	Title  string
	Size   int
	Min    [3]float64 // DOMAIN_MIN, default 0 0 0
	Max    [3]float64 // DOMAIN_MAX, default 1 1 1
	HasDom bool
	// Data stored in .cube order: index = r + size*(g + size*b).
	Data []float64
}

// Apply maps an input color (r,g,b in [0,1]) through the table using
// trilinear interpolation. Inputs are first normalized through the
// per-channel domain, then clamped to the grid. Storage is assumed to
// follow the .cube spec: red varies fastest.
func (l *LUT) Apply(r, g, b float64) (float64, float64, float64) {
	return l.apply(r, g, b, true, [3]int{0, 1, 2})
}

// ApplyNoDomain applies the table but ignores DOMAIN_MIN/MAX, assuming
// a 0..1 input range on every channel. This is the classic "unit scale"
// mistake and is used to render the misread comparison.
func (l *LUT) ApplyNoDomain(r, g, b float64) (float64, float64, float64) {
	return l.apply(r, g, b, false, [3]int{0, 1, 2})
}

// ApplyWrongOrder applies the table assuming blue varies fastest in
// storage, the classic channel-order misread of a .cube file.
func (l *LUT) ApplyWrongOrder(r, g, b float64) (float64, float64, float64) {
	return l.apply(r, g, b, true, [3]int{2, 1, 0})
}

// apply is the shared trilinear core. perm maps storage axis (0 =
// fastest) to color channel; useDomain toggles domain normalization.
func (l *LUT) apply(r, g, b float64, useDomain bool, perm [3]int) (float64, float64, float64) {
	n := l.Size
	in := [3]float64{r, g, b}
	var t [3]float64
	for c := 0; c < 3; c++ {
		v := in[c]
		if useDomain {
			v = l.norm(c, v)
		} else {
			v = clamp01(v)
		}
		t[c] = v * float64(n-1)
	}

	var lo [3]int
	var fr [3]float64
	for c := 0; c < 3; c++ {
		lo[c], fr[c] = split(t[c], n)
	}

	// node fetches the grid value for output channel oc at lattice
	// coordinates given per color channel, honoring storage order.
	node := func(oc int, i, j, k int) float64 {
		var idx [3]int
		idx[perm[0]] = i
		idx[perm[1]] = j
		idx[perm[2]] = k
		return l.Data[3*(idx[0]+n*(idx[1]+n*idx[2]))+oc]
	}

	var out [3]float64
	for c := 0; c < 3; c++ {
		ir, jr, kr := lo[0], lo[1], lo[2]
		c000 := node(c, ir, jr, kr)
		c100 := node(c, ir+1, jr, kr)
		c010 := node(c, ir, jr+1, kr)
		c110 := node(c, ir+1, jr+1, kr)
		c001 := node(c, ir, jr, kr+1)
		c101 := node(c, ir+1, jr, kr+1)
		c011 := node(c, ir, jr+1, kr+1)
		c111 := node(c, ir+1, jr+1, kr+1)

		c00 := c000 + (c100-c000)*fr[0]
		c10 := c010 + (c110-c010)*fr[0]
		c01 := c001 + (c101-c001)*fr[0]
		c11 := c011 + (c111-c011)*fr[0]
		c0 := c00 + (c10-c00)*fr[1]
		c1 := c01 + (c11-c01)*fr[1]
		out[c] = c0 + (c1-c0)*fr[2]
	}
	return out[0], out[1], out[2]
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

// norm maps a sample value for channel c into [0,1] grid space using
// the channel's domain, clamping out-of-domain inputs.
func (l *LUT) norm(c int, v float64) float64 {
	lo, hi := l.Min[c], l.Max[c]
	if hi <= lo {
		return 0
	}
	t := (v - lo) / (hi - lo)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// split decomposes a clamped grid coordinate into a lower cell index
// and fractional offset, keeping the upper index inside the grid.
func split(t float64, n int) (int, float64) {
	if t < 0 {
		t = 0
	}
	max := float64(n - 1)
	if t > max {
		t = max
	}
	i := int(t)
	if i >= n-1 {
		return n - 2, 1
	}
	return i, t - float64(i)
}
