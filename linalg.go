package main

import (
	"errors"
	"math"
)

func SolveDense(a [][]float64, b []float64) ([]float64, error) {
	n := len(b)
	if n == 0 || len(a) != n {
		return nil, errors.New("solve: dimension mismatch")
	}
	m := make([][]float64, n)
	for i := range a {
		if len(a[i]) != n {
			return nil, errors.New("solve: matrix is not square")
		}
		m[i] = append(append([]float64{}, a[i]...), b[i])
	}
	for col := 0; col < n; col++ {
		pivot := col
		for r := col + 1; r < n; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot][col]) < 1e-12 {
			return nil, errors.New("solve: singular matrix")
		}
		m[col], m[pivot] = m[pivot], m[col]
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			factor := m[r][col] / m[col][col]
			for k := col; k <= n; k++ {
				m[r][k] -= factor * m[col][k]
			}
		}
	}
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = m[i][n] / m[i][i]
	}
	return x, nil
}

func FitLeastSquares(rows [][]float64, ys []float64) ([]float64, error) {
	if len(rows) == 0 || len(rows) != len(ys) {
		return nil, errors.New("least squares: empty input")
	}
	k := len(rows[0])
	ata := make([][]float64, k)
	atb := make([]float64, k)
	for i := range ata {
		ata[i] = make([]float64, k)
	}
	for r, row := range rows {
		for i := 0; i < k; i++ {
			atb[i] += row[i] * ys[r]
			for j := 0; j < k; j++ {
				ata[i][j] += row[i] * row[j]
			}
		}
	}
	return SolveDense(ata, atb)
}
