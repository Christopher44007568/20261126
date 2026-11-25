package geom

import "math"

// Camera describes a pinhole camera with a principal-point offset.
// All quantities are in pixel units; the world is the z=0 plane.
type Camera struct {
	W, H   int
	Fx, Fy float64
	Cx, Cy float64
	Depth  float64 // world distance of the z=const scene plane
}

// Pose is a 3-DOF orientation plus an in-plane translation (world units).
type Pose struct {
	Yaw   float64
	Pitch float64
	Roll  float64
	Tx    float64
	Ty    float64
}

func (c Camera) K() Mat3 {
	return Mat3{M: [9]float64{
		c.Fx, 0, c.Cx,
		0, c.Fy, c.Cy,
		0, 0, 1,
	}}
}

// WorldToPixel maps scene plane points to sensor pixels for pose p.
// The scene lies at z=Depth with the camera looking toward +z, so for
// X_w=[X,Y,Depth]: x_c = r0*X + r1*Y + r2*Depth + t, giving the planar
// homography H = K * [r0 r1 (r2*Depth + t)].
func (c Camera) WorldToPixel(p Pose) Mat3 {
	r := RotXYZ(p.Yaw, p.Pitch, p.Roll).M
	h20 := r[2]*c.Depth + p.Tx
	h21 := r[5]*c.Depth + p.Ty
	h22 := r[8] * c.Depth
	extr := Mat3{M: [9]float64{
		r[0], r[1], h20,
		r[3], r[4], h21,
		r[6], r[7], h22,
	}}
	return c.K().Mul(extr)
}

// PixelToWorld is the inverse mapping, used by renderers to sample the scene.
func (c Camera) PixelToWorld(p Pose) Mat3 {
	return c.WorldToPixel(p).Invert()
}

// RowReadTime converts a sensor row to normalized readout time in [0,1],
// with row 0 exposed first and the last row last (rolling shutter).
func RowReadTime(row float64, h int) float64 {
	if h <= 1 {
		return 0
	}
	return row / float64(h-1)
}

// Deg converts degrees to radians.
func Deg(d float64) float64 { return d * math.Pi / 180 }
