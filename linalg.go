package main

import "fmt"

// Triplet is one sparse entry (col, value) within a single matrix row.
type Triplet struct {
	Col int
	Val float64
}

// NormalEquations accumulates A^T A x = A^T b from one sparse row and its rhs.
func NormalEquations(ata [][]float64, atb []float64, row []Triplet, b float64) {
	for _, a := range row {
		atb[a.Col] += a.Val * b
		for _, c := range row {
			ata[a.Col][c.Col] += a.Val * c.Val
		}
	}
}

// SolveGaussian solves a dense linear system with partial-pivot Gaussian
// elimination. The input slices are modified in place.
func SolveGaussian(a [][]float64, b []float64) ([]float64, error) {
	n := len(a)
	if n == 0 || len(b) != n {
		return nil, fmt.Errorf("invalid system dimensions")
	}
	for col := 0; col < n; col++ {
		pivot := col
		best := abs(a[col][col])
		for r := col + 1; r < n; r++ {
			if v := abs(a[r][col]); v > best {
				best, pivot = v, r
			}
		}
		if best < 1e-12 {
			return nil, fmt.Errorf("singular matrix at column %d", col)
		}
		if pivot != col {
			a[col], a[pivot] = a[pivot], a[col]
			b[col], b[pivot] = b[pivot], b[col]
		}
		piv := a[col][col]
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			factor := a[r][col] / piv
			if factor == 0 {
				continue
			}
			for k := col; k < n; k++ {
				a[r][k] -= factor * a[col][k]
			}
			b[r] -= factor * b[col]
		}
	}
	x := make([]float64, n)
	for i := range x {
		x[i] = b[i] / a[i][i]
	}
	return x, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
