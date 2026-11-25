package main

import (
	"errors"
	"fmt"
	"math"
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Camera struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Fx     float64 `json:"fx"`
	Fy     float64 `json:"fy"`
	Cx     float64 `json:"cx"`
	Cy     float64 `json:"cy"`
	K1     float64 `json:"k1"`
	K2     float64 `json:"k2"`
	P1     float64 `json:"p1"`
	P2     float64 `json:"p2"`
	K3     float64 `json:"k3"`
}

type Pose struct {
	R [3][3]float64 `json:"r"`
	T Vec3          `json:"t"`
}

func (c Camera) Valid() error {
	if c.Width < 16 || c.Height < 16 {
		return errors.New("camera dimensions are too small")
	}
	for name, v := range map[string]float64{
		"fx": c.Fx, "fy": c.Fy, "cx": c.Cx, "cy": c.Cy,
		"k1": c.K1, "k2": c.K2, "p1": c.P1, "p2": c.P2, "k3": c.K3,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("parameter %s is not finite", name)
		}
	}
	if c.Fx <= 0 || c.Fy <= 0 {
		return errors.New("focal lengths must be positive")
	}
	if c.Cx < -float64(c.Width) || c.Cx > 2*float64(c.Width) ||
		c.Cy < -float64(c.Height) || c.Cy > 2*float64(c.Height) {
		return errors.New("principal point is implausibly far outside image")
	}
	return nil
}

func RotationMatrix(r Vec3) [3][3]float64 {
	theta := math.Hypot(math.Hypot(r.X, r.Y), r.Z)
	if theta < 1e-12 {
		return [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	}
	kx, ky, kz := r.X/theta, r.Y/theta, r.Z/theta
	s, q := math.Sin(theta), 1-math.Cos(theta)
	return [3][3]float64{
		{q*kx*kx + math.Cos(theta), q*kx*ky - s*kz, q*kx*kz + s*ky},
		{q*kx*ky + s*kz, q*ky*ky + math.Cos(theta), q*ky*kz - s*kx},
		{q*kx*kz - s*ky, q*ky*kz + s*kx, q*kz*kz + math.Cos(theta)},
	}
}

func PoseFromRV(rvec, tvec Vec3) Pose {
	return Pose{R: RotationMatrix(rvec), T: tvec}
}

func (p Pose) Transform(x Vec3) Vec3 {
	return Vec3{
		X: p.R[0][0]*x.X + p.R[0][1]*x.Y + p.R[0][2]*x.Z + p.T.X,
		Y: p.R[1][0]*x.X + p.R[1][1]*x.Y + p.R[1][2]*x.Z + p.T.Y,
		Z: p.R[2][0]*x.X + p.R[2][1]*x.Y + p.R[2][2]*x.Z + p.T.Z,
	}
}

func (c Camera) distort(x, y float64) (float64, float64) {
	r2 := x*x + y*y
	r4 := r2 * r2
	r6 := r4 * r2
	radial := 1 + c.K1*r2 + c.K2*r4 + c.K3*r6
	dx := 2*c.P1*x*y + c.P2*(r2+2*x*x)
	dy := c.P1*(r2+2*y*y) + 2*c.P2*x*y
	return x*radial + dx, y*radial + dy
}

func (c Camera) ProjectLocal(p Vec3) Vec2 {
	x, y := p.X/p.Z, p.Y/p.Z
	xd, yd := c.distort(x, y)
	return Vec2{X: c.Fx*xd + c.Cx, Y: c.Fy*yd + c.Cy}
}

func (c Camera) Project(object Vec3, pose Pose) Vec2 {
	return c.ProjectLocal(pose.Transform(object))
}

func (c Camera) UndistortNormalized(xd, yd float64) (float64, float64) {
	x, y := xd, yd
	for i := 0; i < 12; i++ {
		r2 := x*x + y*y
		r4 := r2 * r2
		r6 := r4 * r2
		radial := 1 + c.K1*r2 + c.K2*r4 + c.K3*r6
		dx := 2*c.P1*x*y + c.P2*(r2+2*x*x)
		dy := c.P1*(r2+2*y*y) + 2*c.P2*x*y
		x = (xd - dx) / radial
		y = (yd - dy) / radial
	}
	return x, y
}

func (c Camera) PixelToUndistortedRay(p Vec2) Vec3 {
	xd := (p.X - c.Cx) / c.Fx
	yd := (p.Y - c.Cy) / c.Fy
	x, y := c.UndistortNormalized(xd, yd)
	return Vec3{X: x, Y: y, Z: 1}
}

type PointResidual struct {
	Frame int     `json:"frame"`
	ID    int     `json:"id"`
	Got   Vec2    `json:"got"`
	Want  Vec2    `json:"want"`
	Dx    float64 `json:"dx"`
	Dy    float64 `json:"dy"`
	Error float64 `json:"error"`
}

func ReprojectionResiduals(cam Camera, poses []Pose, objects [][]Vec3, observed [][]Vec2) []PointResidual {
	var out []PointResidual
	for f := range observed {
		for i, got := range observed[f] {
			want := cam.Project(objects[f][i], poses[f])
			dx, dy := got.X-want.X, got.Y-want.Y
			out = append(out, PointResidual{
				Frame: f, ID: i, Got: got, Want: want,
				Dx: dx, Dy: dy, Error: math.Hypot(dx, dy),
			})
		}
	}
	return out
}

func RMSEResidual(rs []PointResidual) float64 {
	if len(rs) == 0 {
		return math.Inf(1)
	}
	sum := 0.0
	for _, r := range rs {
		sum += r.Dx*r.Dx + r.Dy*r.Dy
	}
	return math.Sqrt(sum / float64(len(rs)))
}
