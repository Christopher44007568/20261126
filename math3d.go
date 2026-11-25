package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 {
	return V(a.X*s, a.Y*s, a.Z*s)
}

func (a Vec3) MulVec(b Vec3) Vec3 {
	return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z)
}

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) LenSqr() float64 { return a.Dot(a) }
func (a Vec3) Len() float64    { return math.Sqrt(a.LenSqr()) }

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l == 0 || math.IsNaN(l) || math.IsInf(l, 0) {
		return V(0, 0, 0)
	}
	return a.Mul(1 / l)
}

func (a Vec3) Min(b Vec3) Vec3 {
	return V(math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z))
}

func (a Vec3) Max(b Vec3) Vec3 {
	return V(math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z))
}

func clampFloat(x, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, x))
}

func clamp01(x float64) float64 { return clampFloat(x, 0, 1) }

type Mat4 [16]float64

func IdentityMat4() Mat4 {
	return Mat4{
		0: 1, 5: 1, 10: 1, 15: 1,
	}
}

// TRS builds a column-vector affine transform using rotation around Z.
func TRS(pos Vec3, angleZ float64, scale Vec3) Mat4 {
	c, s := math.Cos(angleZ), math.Sin(angleZ)
	return Mat4{
		0: c * scale.X, 1: s * scale.X,
		4: -s * scale.Y, 5: c * scale.Y,
		10: scale.Z,
		12: pos.X, 13: pos.Y, 14: pos.Z, 15: 1,
	}
}

func matMul(a, b Mat4) Mat4 {
	var out Mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			sum := 0.0
			for k := 0; k < 4; k++ {
				sum += a[k*4+row] * b[col*4+k]
			}
			out[col*4+row] = sum
		}
	}
	return out
}

func matPoint(m Mat4, p Vec3) Vec3 {
	x := m[0]*p.X + m[4]*p.Y + m[8]*p.Z + m[12]
	y := m[1]*p.X + m[5]*p.Y + m[9]*p.Z + m[13]
	z := m[2]*p.X + m[6]*p.Y + m[10]*p.Z + m[14]
	w := m[3]*p.X + m[7]*p.Y + m[11]*p.Z + m[15]
	if w != 0 && w != 1 {
		x /= w
		y /= w
		z /= w
	}
	return V(x, y, z)
}

func matDir(m Mat4, d Vec3) Vec3 {
	return V(
		m[0]*d.X+m[4]*d.Y+m[8]*d.Z,
		m[1]*d.X+m[5]*d.Y+m[9]*d.Z,
		m[2]*d.X+m[6]*d.Y+m[10]*d.Z,
	)
}

func matInverse(m Mat4) Mat4 {
	// General 4x4 inversion via cofactors. The demo only uses affine TRS
	// matrices, but keeping the full path makes nested relocation robust.
	a := [16]float64(m)
	var inv [16]float64
	inv[0] = a[5]*a[10]*a[15] - a[5]*a[11]*a[14] - a[9]*a[6]*a[15] + a[9]*a[7]*a[14] + a[13]*a[6]*a[11] - a[13]*a[7]*a[10]
	inv[4] = -a[4]*a[10]*a[15] + a[4]*a[11]*a[14] + a[8]*a[6]*a[15] - a[8]*a[7]*a[14] - a[12]*a[6]*a[11] + a[12]*a[7]*a[10]
	inv[8] = a[4]*a[9]*a[15] - a[4]*a[11]*a[13] - a[8]*a[5]*a[15] + a[8]*a[7]*a[13] + a[12]*a[5]*a[11] - a[12]*a[7]*a[9]
	inv[12] = -a[4]*a[9]*a[14] + a[4]*a[10]*a[13] + a[8]*a[5]*a[14] - a[8]*a[6]*a[13] - a[12]*a[5]*a[10] + a[12]*a[6]*a[9]
	inv[1] = -a[1]*a[10]*a[15] + a[1]*a[11]*a[14] + a[9]*a[2]*a[15] - a[9]*a[3]*a[14] - a[13]*a[2]*a[11] + a[13]*a[3]*a[10]
	inv[5] = a[0]*a[10]*a[15] - a[0]*a[11]*a[14] - a[8]*a[2]*a[15] + a[8]*a[3]*a[14] + a[12]*a[2]*a[11] - a[12]*a[3]*a[10]
	inv[9] = -a[0]*a[9]*a[15] + a[0]*a[11]*a[13] + a[8]*a[1]*a[15] - a[8]*a[3]*a[13] - a[12]*a[1]*a[11] + a[12]*a[3]*a[9]
	inv[13] = a[0]*a[9]*a[14] - a[0]*a[10]*a[13] - a[8]*a[1]*a[14] + a[8]*a[2]*a[13] + a[12]*a[1]*a[10] - a[12]*a[2]*a[9]
	inv[2] = a[1]*a[6]*a[15] - a[1]*a[7]*a[14] - a[5]*a[2]*a[15] + a[5]*a[3]*a[14] + a[13]*a[2]*a[7] - a[13]*a[3]*a[6]
	inv[6] = -a[0]*a[6]*a[15] + a[0]*a[7]*a[14] + a[4]*a[2]*a[15] - a[4]*a[3]*a[14] - a[12]*a[2]*a[7] + a[12]*a[3]*a[6]
	inv[10] = a[0]*a[5]*a[15] - a[0]*a[7]*a[13] - a[4]*a[1]*a[15] + a[4]*a[3]*a[13] + a[12]*a[1]*a[7] - a[12]*a[3]*a[5]
	inv[14] = -a[0]*a[5]*a[14] + a[0]*a[6]*a[13] + a[4]*a[1]*a[14] - a[4]*a[2]*a[13] - a[12]*a[1]*a[6] + a[12]*a[2]*a[5]
	inv[3] = -a[1]*a[6]*a[11] + a[1]*a[7]*a[10] + a[5]*a[2]*a[11] - a[5]*a[3]*a[10] - a[9]*a[2]*a[7] + a[9]*a[3]*a[6]
	inv[7] = a[0]*a[6]*a[11] - a[0]*a[7]*a[10] - a[4]*a[2]*a[11] + a[4]*a[3]*a[10] + a[8]*a[2]*a[7] - a[8]*a[3]*a[6]
	inv[11] = -a[0]*a[5]*a[11] + a[0]*a[7]*a[9] + a[4]*a[1]*a[11] - a[4]*a[3]*a[9] - a[8]*a[1]*a[7] + a[8]*a[3]*a[5]
	inv[15] = a[0]*a[5]*a[10] - a[0]*a[6]*a[9] - a[4]*a[1]*a[10] + a[4]*a[2]*a[9] + a[8]*a[1]*a[6] - a[8]*a[2]*a[5]
	det := a[0]*inv[0] + a[1]*inv[4] + a[2]*inv[8] + a[3]*inv[12]
	if math.Abs(det) < 1e-18 {
		return IdentityMat4()
	}
	for i := range inv {
		inv[i] /= det
	}
	return Mat4(inv)
}
