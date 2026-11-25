package main

import "math"

type polyMap struct {
	CX, CY, SX, SY float64
	AX, AY         []float64
}

func fitPolyMap(coords []Vec2, points []Vec2) (*polyMap, error) {
	if len(coords) < 6 {
		return nil, errInsufficientPoints
	}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, q := range coords {
		minX, maxX = math.Min(minX, q.X), math.Max(maxX, q.X)
		minY, maxY = math.Min(minY, q.Y), math.Max(maxY, q.Y)
	}
	m := &polyMap{CX: (minX + maxX) / 2, CY: (minY + maxY) / 2, SX: math.Max((maxX-minX)/2, 1), SY: math.Max((maxY-minY)/2, 1)}
	rows := make([][]float64, len(coords))
	xs := make([]float64, len(coords))
	ys := make([]float64, len(coords))
	for i, q := range coords {
		u, v := (q.X-m.CX)/m.SX, (q.Y-m.CY)/m.SY
		rows[i] = []float64{1, u, v, u * u, u * v, v * v}
		xs[i], ys[i] = points[i].X, points[i].Y
	}
	var err error
	if m.AX, err = FitLeastSquares(rows, xs); err != nil {
		return nil, err
	}
	if m.AY, err = FitLeastSquares(rows, ys); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *polyMap) Map(q Vec2) Vec2 {
	u, v := (q.X-m.CX)/m.SX, (q.Y-m.CY)/m.SY
	base := []float64{1, u, v, u * u, u * v, v * v}
	var x, y float64
	for i := range m.AX {
		x += m.AX[i] * base[i]
		y += m.AY[i] * base[i]
	}
	return Vec2{X: x, Y: y}
}
