package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Mul(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Norm() Vec3 {
	length := a.Len()
	if length < 1e-12 {
		return V(0, 0, 0)
	}
	return a.Mul(1 / length)
}

func (a Vec3) Min(b Vec3) Vec3 {
	return V(math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z))
}

func (a Vec3) Max(b Vec3) Vec3 {
	return V(math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

type Mat4 struct {
	E [4][4]float64
}

func Identity() Mat4 {
	var m Mat4
	for i := 0; i < 4; i++ {
		m.E[i][i] = 1
	}
	return m
}

func Translate(v Vec3) Mat4 {
	m := Identity()
	m.E[0][3] = v.X
	m.E[1][3] = v.Y
	m.E[2][3] = v.Z
	return m
}

func RotX(angle float64) Mat4 {
	c, s := math.Cos(angle), math.Sin(angle)
	m := Identity()
	m.E[1][1] = c
	m.E[1][2] = -s
	m.E[2][1] = s
	m.E[2][2] = c
	return m
}

func Mul(a, b Mat4) Mat4 {
	var out Mat4
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			out.E[row][col] = a.E[row][0]*b.E[0][col] +
				a.E[row][1]*b.E[1][col] +
				a.E[row][2]*b.E[2][col] +
				a.E[row][3]*b.E[3][col]
		}
	}
	return out
}

func (m Mat4) Point(p Vec3) Vec3 {
	return V(
		m.E[0][0]*p.X+m.E[0][1]*p.Y+m.E[0][2]*p.Z+m.E[0][3],
		m.E[1][0]*p.X+m.E[1][1]*p.Y+m.E[1][2]*p.Z+m.E[1][3],
		m.E[2][0]*p.X+m.E[2][1]*p.Y+m.E[2][2]*p.Z+m.E[2][3],
	)
}

func (m Mat4) Direction(d Vec3) Vec3 {
	return V(
		m.E[0][0]*d.X+m.E[0][1]*d.Y+m.E[0][2]*d.Z,
		m.E[1][0]*d.X+m.E[1][1]*d.Y+m.E[1][2]*d.Z,
		m.E[2][0]*d.X+m.E[2][1]*d.Y+m.E[2][2]*d.Z,
	)
}

// RigidInverse inverts a rotation-plus-translation matrix. Character
// instances intentionally use no scale, keeping ray distances unchanged.
func RigidInverse(m Mat4) Mat4 {
	var inv Mat4
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			inv.E[row][col] = m.E[col][row]
		}
	}
	for row := 0; row < 3; row++ {
		inv.E[row][3] = -(inv.E[row][0]*m.E[0][3] +
			inv.E[row][1]*m.E[1][3] +
			inv.E[row][2]*m.E[2][3])
	}
	inv.E[3][3] = 1
	return inv
}
