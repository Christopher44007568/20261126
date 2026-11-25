package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(o Vec3) Vec3 { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3 { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Mul(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}
func (v Vec3) MulV(o Vec3) Vec3 { return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z} }
func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}
func (v Vec3) Length() float64 { return math.Sqrt(v.Dot(v)) }
func (v Vec3) Norm() Vec3 {
	n := v.Length()
	if n == 0 {
		return Vec3{}
	}
	return v.Mul(1 / n)
}
func (v Vec3) Reflect(n Vec3) Vec3 {
	return v.Sub(n.Mul(2 * v.Dot(n)))
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func mix(a, b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

type Sphere struct {
	Center Vec3
	Radius float64
}

type SceneObject struct {
	ID      string
	Sphere  Sphere
	Albedo  Vec3
	Visible bool
}

type Scene struct {
	Name    string
	Objects []SceneObject
}

func (s *Sphere) Hit(r Ray, tMin, tMax float64) (float64, Vec3, bool) {
	oc := r.Origin.Sub(s.Center)
	a := r.Dir.Dot(r.Dir)
	b := 2 * oc.Dot(r.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, Vec3{}, false
	}
	root := math.Sqrt(disc)
	t := (-b - root) / (2 * a)
	if t < tMin || t > tMax {
		t = (-b + root) / (2 * a)
		if t < tMin || t > tMax {
			return 0, Vec3{}, false
		}
	}
	hit := r.Origin.Add(r.Dir.Mul(t))
	return t, hit.Sub(s.Center).Norm(), true
}

type hitRecord struct {
	T      float64
	Normal Vec3
	Object SceneObject
}

func nearestHit(objects []SceneObject, r Ray, tMin, tMax float64) (hitRecord, bool) {
	best := hitRecord{T: tMax}
	found := false
	for _, obj := range objects {
		if !obj.Visible || obj.Sphere.Radius <= 0 {
			continue
		}
		if t, n, ok := obj.Sphere.Hit(r, tMin, best.T); ok {
			best = hitRecord{T: t, Normal: n, Object: obj}
			found = true
		}
	}
	return best, found
}

var (
	keyLightDir = Vec3{-0.55, 0.78, 0.42}.Norm()
	keyLightCol = Vec3{1.0, 0.93, 0.80}
	fillLight   = Vec3{0.18, 0.32, 0.62}
)

func background(dir Vec3) Vec3 {
	t := clamp01(0.5*dir.Y + 0.5)
	return mix(Vec3{0.008, 0.014, 0.038}, Vec3{0.045, 0.105, 0.20}, t).
		Add(Vec3{0.018, 0.055, 0.11}.Mul(math.Pow(1-math.Abs(dir.X), 8)))
}

func shadeRay(objects []SceneObject, r Ray, depth int) Vec3 {
	hit, ok := nearestHit(objects, r, 0.001, 1e30)
	if !ok {
		return background(r.Dir)
	}

	n := hit.Normal
	view := r.Dir.Mul(-1).Norm()
	ambient := hit.Object.Albedo.Mul(0.14).Add(fillLight.Mul(0.08))

	shadowOrigin := r.Origin.Add(r.Dir.Mul(hit.T)).Add(n.Mul(0.002))
	toLight := keyLightDir
	shadowRay := Ray{Origin: shadowOrigin, Dir: toLight}
	_, blocked := nearestHit(objects, shadowRay, 0.001, 1e30)
	visibility := 1.0
	if blocked {
		visibility = 0.14
	}

	diffuse := math.Max(0, n.Dot(toLight))
	specular := 0.0
	if diffuse > 0 {
		half := toLight.Add(view).Norm()
		specular = math.Pow(math.Max(0, n.Dot(half)), 56)
	}

	col := ambient.Add(
		hit.Object.Albedo.MulV(keyLightCol).Mul(diffuse * visibility * 0.86),
	).Add(keyLightCol.Mul(specular * visibility * 0.55))

	if depth > 0 {
		reflected := r.Dir.Reflect(n)
		bounce := shadeRay(objects, Ray{
			Origin: shadowOrigin,
			Dir:    reflected,
		}, depth-1)
		col = col.Add(bounce.MulV(hit.Object.Albedo).Mul(0.13))
	}

	rim := math.Pow(1-math.Max(0, n.Dot(view)), 3)
	col = col.Add(Vec3{0.10, 0.22, 0.38}.Mul(rim * 0.22))
	return col
}

func toRGB(v Vec3) [3]byte {
	gamma := func(x float64) byte {
		x = clamp01(x)
		x = math.Pow(x, 1/2.2)
		return byte(math.Round(x * 255))
	}
	return [3]byte{gamma(v.X), gamma(v.Y), gamma(v.Z)}
}

func sampleScene(objects []SceneObject, px, py, width, height int) [3]byte {
	const samples = 2
	var sum Vec3
	aspect := float64(width) / float64(height)
	for sy := 0; sy < samples; sy++ {
		for sx := 0; sx < samples; sx++ {
			u := (float64(px) + (float64(sx)+0.5)/samples) / float64(width)
			v := (float64(py) + (float64(sy)+0.5)/samples) / float64(height)
			x := (2*u - 1) * aspect * 0.56
			y := (1 - 2*v) * 0.56
			dir := Vec3{x, y, -1}.Norm()
			sum = sum.Add(shadeRay(objects, Ray{
				Origin: Vec3{0, 0.12, 4.4},
				Dir:    dir,
			}, 1))
		}
	}
	return toRGB(sum.Mul(1.0 / (samples * samples)))
}

func renderOracle(scene Scene, width, height int) []byte {
	pix := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgb := sampleScene(scene.Objects, x, y, width, height)
			i := (y*width + x) * 3
			copy(pix[i:i+3], rgb[:])
		}
	}
	return pix
}

func frames() []Scene {
	return []Scene{
		{
			Name: "FRAME 1",
			Objects: []SceneObject{
				{ID: "amber", Sphere: Sphere{Center: Vec3{-1.00, 0.22, 0.0}, Radius: 0.52}, Albedo: Vec3{0.92, 0.31, 0.14}, Visible: true},
				{ID: "cyan", Sphere: Sphere{Center: Vec3{0.72, 0.34, -0.2}, Radius: 0.40}, Albedo: Vec3{0.12, 0.62, 0.80}, Visible: true},
				{ID: "violet", Sphere: Sphere{Center: Vec3{0.04, -0.62, 0.30}, Radius: 0.34}, Albedo: Vec3{0.48, 0.22, 0.72}, Visible: true},
				{ID: "lime", Sphere: Sphere{Center: Vec3{0.94, -0.48, 0.55}, Radius: 0.22}, Albedo: Vec3{0.55, 0.82, 0.20}, Visible: true},
			},
		},
		{
			Name: "FRAME 2",
			Objects: []SceneObject{
				{ID: "amber", Sphere: Sphere{Center: Vec3{-0.34, 0.30, 0.12}, Radius: 0.48}, Albedo: Vec3{0.92, 0.31, 0.14}, Visible: true},
				{ID: "cyan", Sphere: Sphere{Center: Vec3{-0.86, -0.30, 0.18}, Radius: 0.38}, Albedo: Vec3{0.12, 0.62, 0.80}, Visible: true},
				{ID: "violet", Sphere: Sphere{Center: Vec3{0.62, 0.18, -0.10}, Radius: 0.36}, Albedo: Vec3{0.48, 0.22, 0.72}, Visible: true},
				{ID: "lime", Sphere: Sphere{Center: Vec3{0.92, -0.50, 0.55}, Radius: 0.22}, Albedo: Vec3{0.55, 0.82, 0.20}, Visible: false},
			},
		},
		{
			Name: "FRAME 3",
			Objects: []SceneObject{
				{ID: "amber", Sphere: Sphere{Center: Vec3{-1.00, 0.22, 0.0}, Radius: 0.52}, Albedo: Vec3{0.92, 0.31, 0.14}, Visible: false},
				{ID: "cyan", Sphere: Sphere{Center: Vec3{-0.22, 0.20, 0.24}, Radius: 0.42}, Albedo: Vec3{0.12, 0.62, 0.80}, Visible: true},
				{ID: "violet", Sphere: Sphere{Center: Vec3{0.58, -0.42, 0.12}, Radius: 0.36}, Albedo: Vec3{0.48, 0.22, 0.72}, Visible: true},
				{ID: "lime", Sphere: Sphere{Center: Vec3{1.08, 0.42, 0.58}, Radius: 0.26}, Albedo: Vec3{0.55, 0.82, 0.20}, Visible: true},
			},
		},
	}
}
