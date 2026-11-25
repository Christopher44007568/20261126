package main

import "math"

type Vec3 struct{ X, Y, Z float32 }

func vadd(a, b Vec3) Vec3  { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func vsub(a, b Vec3) Vec3  { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func vscale(a Vec3, s float32) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func vdot(a, b Vec3) float32 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func vcross(a, b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func vlen(a Vec3) float32 { return float32(math.Sqrt(float64(vdot(a, a)))) }
func vnorm(a Vec3) Vec3 {
	l := vlen(a)
	if l == 0 {
		return a
	}
	return vscale(a, 1/l)
}
func vmin(a, b Vec3) Vec3 {
	return Vec3{min(a.X, b.X), min(a.Y, b.Y), min(a.Z, b.Z)}
}
func vmax(a, b Vec3) Vec3 {
	return Vec3{max(a.X, b.X), max(a.Y, b.Y), max(a.Z, b.Z)}
}

type Ray struct {
	Orig, Dir, Inv Vec3
}

// NewRay nudges zero direction components so slab tests never see Inf*0.
func NewRay(o, d Vec3) Ray {
	nudge := func(v float32) float32 {
		if v == 0 {
			return 1e-9
		}
		return v
	}
	d.X, d.Y, d.Z = nudge(d.X), nudge(d.Y), nudge(d.Z)
	return Ray{Orig: o, Dir: d, Inv: Vec3{1 / d.X, 1 / d.Y, 1 / d.Z}}
}

// slabHit reports whether the ray hits box [mn,mx] before tBest.
func slabHit(r Ray, mn, mx Vec3, tBest float32) bool {
	t1 := (mn.X - r.Orig.X) * r.Inv.X
	t2 := (mx.X - r.Orig.X) * r.Inv.X
	tmin, tmax := min(t1, t2), max(t1, t2)
	t1 = (mn.Y - r.Orig.Y) * r.Inv.Y
	t2 = (mx.Y - r.Orig.Y) * r.Inv.Y
	tmin, tmax = max(tmin, min(t1, t2)), min(tmax, max(t1, t2))
	t1 = (mn.Z - r.Orig.Z) * r.Inv.Z
	t2 = (mx.Z - r.Orig.Z) * r.Inv.Z
	tmin, tmax = max(tmin, min(t1, t2)), min(tmax, max(t1, t2))
	return tmax >= tmin && tmax > 0 && tmin < tBest
}
