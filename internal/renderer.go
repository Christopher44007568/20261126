package internal

import "math"

// Renderer traces differential rays into a scene.
type Renderer struct {
	Scene  *Scene
	Camera Camera
	Width  int
	Height int
	Naive  bool // when true, secondary rays reuse the primary LOD
}

// Buffers are the three requested outputs plus the per-pixel selected LOD.
type Buffers struct {
	Beauty     []RGBA
	Reflection []RGBA
	LODMap     []RGBA
	LODLevel   []float64
}

func NewRenderer(w, h int, naive bool) *Renderer {
	sc := NewScene()
	cam := NewCamera(V(0, 3.2, 7.5), V(0, 1, 0), 55, w, h)
	return &Renderer{Scene: sc, Camera: cam, Width: w, Height: h, Naive: naive}
}

type hitInfo struct {
	t        float64
	p, n     Vec3
	dpx, dpy Vec3
	dnx, dny Vec3
	mat      Material
}

func (r *Renderer) trace(ray Ray) (hitInfo, bool) {
	best := hitInfo{t: math.Inf(1)}
	found := false

	// Ground plane.
	ground := Plane{Height: 0}
	if t, p, n, dpx, dpy, dnx, dny, ok := ground.Intersect(ray); ok && t < best.t {
		best = hitInfo{
			t: t, p: p, n: n, dpx: dpx, dpy: dpy, dnx: dnx, dny: dny,
			mat: Material{Metal: false, Tint: RGBA{1, 1, 1, 1}},
		}
		found = true
	}

	for _, obj := range r.Scene.Objects {
		geo := obj.Geo
		if t, p, n, dpx, dpy, dnx, dny, ok := geo.Intersect(ray); ok && t < best.t {
			best = hitInfo{
				t: t, p: p, n: n, dpx: dpx, dpy: dpy, dnx: dnx, dny: dny,
				mat: obj.Material,
			}
			found = true
		}
	}
	return best, found
}

// groundSample samples the checker using the hit's own footprint.
func (r *Renderer) groundSample(h hitInfo) (RGBA, float64) {
	u, v := GroundUV(h.p)
	dudx, dvdx, dudy, dvdy := GroundDerivs(h.dpx, h.dpy)
	return r.Scene.Ground.FootprintSample(u, v, dudx, dvdx, dudy, dvdy)
}

// skySample samples the environment from a ray direction footprint.
func (r *Renderer) skySample(ray Ray) (RGBA, float64) {
	u, v := SkyUV(ray.Dir)
	if !ray.HasDiff {
		return r.Scene.Sky.FootprintSample(u, v, 0, 0, 0, 0)
	}
	dudx, dvdx, dudy, dvdy := SkyDerivs(ray.Dir, ray.DDirX, ray.DDirY)
	return r.Scene.Sky.FootprintSample(u, v, dudx, dvdx, dudy, dvdy)
}

// traceEnvironment follows a ray through the scene, returning the radiance it
// carries and the texture LOD selected at the terminal sample. Reflection
// differentials (including roughness broadening) propagate on every bounce.
//
// In naive mode every bounce ignores its own footprint and reuses the LOD the
// primary ray would have selected, which aliases distant metal reflections.
func (r *Renderer) traceEnvironment(ray Ray, depth, primaryLOD int) (RGBA, float64) {
	hit, ok := r.trace(ray)
	if !ok {
		c, lod := r.skySample(ray)
		if r.Naive {
			return c, float64(primaryLOD)
		}
		return c, lod
	}

	if !hit.mat.Metal {
		c, lod := r.groundSample(hit)
		if r.Naive {
			return c, float64(primaryLOD)
		}
		return c, lod
	}

	// Conductor hit: reflect and recurse while bounce budget remains.
	refl := ReflectedRay(
		ray, hit.p, hit.n,
		hit.dpx, hit.dpy, hit.dnx, hit.dny,
		hit.mat.Roughness,
	)
	if depth <= 0 {
		c, lod := r.skySample(refl)
		if r.Naive {
			return c, float64(primaryLOD)
		}
		return c, lod
	}

	child, lod := r.traceEnvironment(refl, depth-1, primaryLOD)
	cosTheta := math.Abs(ray.Dir.Dot(hit.n))
	fresnel := schlick(cosTheta)
	tinted := multiply(hit.mat.Tint, child)
	out := RGBA{
		R: Lerp(tinted.R, child.R, fresnel),
		G: Lerp(tinted.G, child.G, fresnel),
		B: Lerp(tinted.B, child.B, fresnel),
		A: 1,
	}
	return out, lod
}

// Render produces all three buffers in a single pass.
func (r *Renderer) Render() Buffers {
	n := r.Width * r.Height
	buf := Buffers{
		Beauty:     make([]RGBA, n),
		Reflection: make([]RGBA, n),
		LODMap:     make([]RGBA, n),
		LODLevel:   make([]float64, n),
	}
	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			idx := y*r.Width + x
			beauty, reflCol, lod := r.renderPixel(x, y)
			buf.Beauty[idx] = beauty
			buf.Reflection[idx] = reflCol
			buf.LODMap[idx] = LODColor(lod, r.Scene.Ground.MaxLOD())
			buf.LODLevel[idx] = lod
		}
	}
	return buf
}

func (r *Renderer) renderPixel(px, py int) (beauty, reflection RGBA, lod float64) {
	primary := r.Camera.GenerateRay(px, py)
	hit, ok := r.trace(primary)
	if !ok {
		skyCol, skyLOD := r.skySample(primary)
		return skyCol, RGBA{}, skyLOD
	}

	if !hit.mat.Metal {
		// Ground: primary LOD comes directly from the primary footprint.
		base, primaryLOD := r.groundSample(hit)
		shaded := r.shadeGround(base, hit)

		// The reflection buffer shows what a mirror bounce from this point sees.
		refl := ReflectedRay(
			primary, hit.p, hit.n,
			hit.dpx, hit.dpy, hit.dnx, hit.dny, 0,
		)
		reflCol, _ := r.traceEnvironment(refl, r.Scene.MaxBounce, int(primaryLOD))
		return shaded, reflCol, primaryLOD
	}

	// Metal: the visible color is the reflected environment. Its terminal LOD
	// already reflects roughness broadening through the transferred differentials.
	_, proxyLOD := r.groundSample(hit)
	beauty, secondaryLOD := r.traceEnvironment(
		primary, r.Scene.MaxBounce, int(proxyLOD),
	)
	return beauty, beauty, secondaryLOD
}
