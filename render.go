package main

import "math"

const MaxDepth = 5

type Camera struct {
	Origin, Forward, Right, Up Vec3
	HalfW, HalfH               float64
}

func NewCamera(eye, target Vec3, fovDeg, aspect float64) Camera {
	fwd := target.Sub(eye).Normalize()
	right := fwd.Cross(Vec3{0, 1, 0}).Normalize()
	up := right.Cross(fwd).Normalize()
	halfH := math.Tan(fovDeg * math.Pi / 360)
	return Camera{eye, fwd, right, up, halfH * aspect, halfH}
}

func (c Camera) Ray(u, v float64) Ray {
	dir := c.Forward.
		Add(c.Right.Scale(u * c.HalfW)).
		Add(c.Up.Scale(v * c.HalfH)).Normalize()
	return Ray{c.Origin, dir}
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func sky(dir Vec3) Vec3 {
	t := 0.5 * (dir.Y + 1)
	return Lerp(Vec3{0.85, 0.9, 1.0}, Vec3{0.25, 0.45, 0.75}, clamp(t))
}

// trace is the recursive Whitted camera path. The second result reports
// whether the first surface hit was a glass volume; the third returns the
// any-hit light occlusion for an opaque first hit so callers can reuse it for
// the shadow and transmission maps without a duplicate visibility query.
func (s *Scene) trace(r Ray, depth int, st *Stats) (Vec3, bool, *Occlusion) {
	if depth > MaxDepth {
		return Vec3{}, false, nil
	}
	h, ok := s.NearestHit(r)
	if !ok {
		return sky(r.Dir), false, nil
	}
	p := r.At(h.T)

	if h.Glass == nil {
		occ := s.occlude(p, h.Normal, st)
		return s.shadeOpaque(h, p, occ), false, &occ
	}

	// Dielectric surface: Fresnel split between reflection and refraction.
	g := h.Glass
	normal := h.Normal
	cosTheta := r.Dir.Dot(normal)
	eta := 1.0 / g.IOR // incident IOR over transmitted IOR
	if cosTheta > 0 {
		// Hitting the far face from inside: flip normal and invert eta.
		normal = normal.Scale(-1)
		cosTheta = -cosTheta
		eta = g.IOR
	}
	fres := Schlick(-cosTheta, g.IOR)

	reflectDir := Reflect(r.Dir, h.Normal)
	reflectRay := Ray{p.Add(h.Normal.Scale(Eps)), reflectDir}
	refl, _, _ := s.trace(reflectRay, depth+1, st)

	refrDir, canRefract := Refract(r.Dir, normal, eta)
	if !canRefract {
		return refl, true, nil // total internal reflection
	}

	tint := Vec3{1, 1, 1}
	if h.Entering {
		// Find where this ray leaves the same glass volume and tint by path.
		inside := Ray{p.Add(normal.Scale(-Eps)), refrDir}
		if exitT, found := s.exitDistance(inside, g); found {
			tint = BeerLambert(g.Absorption, exitT)
		}
	}
	refrRay := Ray{p.Add(normal.Scale(-Eps)), refrDir}
	transCol, _, _ := s.trace(refrRay, depth+1, st)
	trans := transCol.Comp(tint)

	return refl.Scale(fres).Add(trans.Scale(1 - fres)), true, nil
}

// exitDistance finds the first exit event of the same glass for a ray whose
// origin is inside that volume.
func (s *Scene) exitDistance(r Ray, g *Glass) (float64, bool) {
	for _, h := range s.AllHits(r) {
		if h.Glass == g && !h.Entering {
			return h.T, true
		}
	}
	return 0, false
}

func (s *Scene) occlude(p Vec3, n Vec3, st *Stats) Occlusion {
	shadowRay := Ray{p.Add(n.Scale(Eps)), s.LightDir}
	return s.QueryOcclusion(shadowRay, st)
}

func (s *Scene) shadeOpaque(h Hit, p Vec3, occ Occlusion) Vec3 {
	base := Albedo(h, p)
	ambient := base.Scale(0.22)
	if occ.Opaque {
		return ambient // hard shadow region
	}

	diffuse := math.Max(h.Normal.Dot(s.LightDir), 0)
	direct := base.Scale(diffuse).Comp(occ.Transmitted)
	return ambient.Add(direct)
}
