package lut

import (
	"fmt"
	"io"
)

// Identity returns an n^3 identity LUT over the unit domain.
func Identity(n int) *LUT {
	return Build(n, [3]float64{0, 0, 0}, [3]float64{1, 1, 1},
		func(r, g, b float64) (float64, float64, float64) { return r, g, b })
}

// SwapRB returns an identity-domain LUT that exchanges the red and
// blue channels: out = (b, g, r).
func SwapRB(n int) *LUT {
	return Build(n, [3]float64{0, 0, 0}, [3]float64{1, 1, 1},
		func(r, g, b float64) (float64, float64, float64) { return b, g, r })
}

// Build samples fn on the lattice, storing rows in .cube order
// (red fastest). Lattice coordinates are mapped through the domain.
func Build(n int, min, max [3]float64, fn func(r, g, b float64) (float64, float64, float64)) *LUT {
	l := &LUT{Size: n, Min: min, Max: max, HasDom: true}
	l.Data = make([]float64, 0, 3*n*n*n)
	for bi := 0; bi < n; bi++ {
		b := min[2] + (max[2]-min[2])*float64(bi)/float64(n-1)
		for gi := 0; gi < n; gi++ {
			g := min[1] + (max[1]-min[1])*float64(gi)/float64(n-1)
			for ri := 0; ri < n; ri++ {
				r := min[0] + (max[0]-min[0])*float64(ri)/float64(n-1)
				or, og, ob := fn(r, g, b)
				l.Data = append(l.Data, or, og, ob)
			}
		}
	}
	return l
}

// Write serializes the LUT in .cube format.
func (l *LUT) Write(w io.Writer) error {
	if l.Title != "" {
		if _, err := fmt.Fprintf(w, "TITLE %q\n", l.Title); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "LUT_3D_SIZE %d\n", l.Size); err != nil {
		return err
	}
	if l.HasDom {
		if _, err := fmt.Fprintf(w, "DOMAIN_MIN %g %g %g\nDOMAIN_MAX %g %g %g\n",
			l.Min[0], l.Min[1], l.Min[2], l.Max[0], l.Max[1], l.Max[2]); err != nil {
			return err
		}
	}
	for i := 0; i < len(l.Data); i += 3 {
		if _, err := fmt.Fprintf(w, "%.6f %.6f %.6f\n", l.Data[i], l.Data[i+1], l.Data[i+2]); err != nil {
			return err
		}
	}
	return nil
}
