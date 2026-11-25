package geom

import "math"

// Mat3 is a row-major 3x3 homogeneous matrix.
type Mat3 struct {
	M [9]float64
}

func Identity() Mat3 {
	return Mat3{M: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}}
}

func (a Mat3) At(r, c int) float64 { return a.M[r*3+c] }

// Mul returns a*b (apply b first, then a).
func (a Mat3) Mul(b Mat3) Mat3 {
	var out [9]float64
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			s := 0.0
			for k := 0; k < 3; k++ {
				s += a.M[r*3+k] * b.M[k*3+c]
			}
			out[r*3+c] = s
		}
	}
	return Mat3{M: out}
}

// Apply maps the homogeneous point (x,y,1) and divides by w.
func (a Mat3) Apply(x, y float64) (float64, float64) {
	w := a.M[6]*x + a.M[7]*y + a.M[8]
	xh := (a.M[0]*x + a.M[1]*y + a.M[2]) / w
	yh := (a.M[3]*x + a.M[4]*y + a.M[5]) / w
	return xh, yh
}

// Invert returns the inverse of a.
func (a Mat3) Invert() Mat3 {
	m := a.M
	// Cofactor matrix C with the standard checkerboard signs.
	c := [9]float64{
		m[4]*m[8] - m[5]*m[7], m[5]*m[6] - m[3]*m[8], m[3]*m[7] - m[4]*m[6],
		m[2]*m[7] - m[1]*m[8], m[0]*m[8] - m[2]*m[6], m[1]*m[6] - m[0]*m[7],
		m[1]*m[5] - m[2]*m[4], m[2]*m[3] - m[0]*m[5], m[0]*m[4] - m[1]*m[3],
	}
	det := m[0]*c[0] + m[1]*c[1] + m[2]*c[2]
	// Inverse is the transposed cofactor (adjugate) divided by determinant.
	return Mat3{M: [9]float64{
		c[0] / det, c[3] / det, c[6] / det,
		c[1] / det, c[4] / det, c[7] / det,
		c[2] / det, c[5] / det, c[8] / det,
	}}
}

// RotXYZ builds a rotation for yaw (about Y), pitch (about X), roll (about Z),
// composed as Rz*Ry*Rx.
func RotXYZ(yaw, pitch, roll float64) Mat3 {
	cy, sy := math.Cos(yaw), math.Sin(yaw)
	cp, sp := math.Cos(pitch), math.Sin(pitch)
	cr, sr := math.Cos(roll), math.Sin(roll)
	ry := Mat3{M: [9]float64{
		cy, 0, sy,
		0, 1, 0,
		-sy, 0, cy,
	}}
	rx := Mat3{M: [9]float64{
		1, 0, 0,
		0, cp, -sp,
		0, sp, cp,
	}}
	rz := Mat3{M: [9]float64{
		cr, -sr, 0,
		sr, cr, 0,
		0, 0, 1,
	}}
	return rz.Mul(ry).Mul(rx)
}

// Clamp limits v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
