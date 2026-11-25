package main

import "math"

// Point is a 2D point.
type Point struct {
	X, Y float64
}

// Matrix is a 2D affine transform:
//
//	| A C Tx |
//	| B D Ty |
//	| 0 0 1  |
type Matrix struct {
	A, B, C, D, Tx, Ty float64
}

// Identity returns the identity matrix.
func Identity() Matrix { return Matrix{A: 1, D: 1} }

// orIdentity maps the zero Matrix to Identity, so nodes that never set
// a transform behave as if they had none.
func (m Matrix) orIdentity() Matrix {
	if m == (Matrix{}) {
		return Identity()
	}
	return m
}

// Translate returns a translation matrix.
func Translate(x, y float64) Matrix { return Matrix{A: 1, D: 1, Tx: x, Ty: y} }

// Scale returns a scaling matrix.
func Scale(sx, sy float64) Matrix { return Matrix{A: sx, D: sy} }

// Rotate returns a rotation matrix (radians, counter-clockwise).
func Rotate(rad float64) Matrix {
	s, c := math.Sin(rad), math.Cos(rad)
	return Matrix{A: c, B: s, C: -s, D: c}
}

// Mul returns m * n (apply n first, then m).
func (m Matrix) Mul(n Matrix) Matrix {
	return Matrix{
		A:  m.A*n.A + m.C*n.B,
		B:  m.B*n.A + m.D*n.B,
		C:  m.A*n.C + m.C*n.D,
		D:  m.B*n.C + m.D*n.D,
		Tx: m.A*n.Tx + m.C*n.Ty + m.Tx,
		Ty: m.B*n.Tx + m.D*n.Ty + m.Ty,
	}
}

// Apply transforms a point.
func (m Matrix) Apply(p Point) Point {
	return Point{
		X: m.A*p.X + m.C*p.Y + m.Tx,
		Y: m.B*p.X + m.D*p.Y + m.Ty,
	}
}

// ApproxLen estimates how a unit vector along +X is scaled, used to
// approximate stroke-width scaling under the baked transform.
func (m Matrix) ApproxLen() float64 {
	sx := math.Hypot(m.A, m.B)
	sy := math.Hypot(m.C, m.D)
	return (sx + sy) / 2
}
