package main

import "math"

type Light struct {
	Position  Vec3
	Color     Vec3
	Intensity float64
}

type Scene struct {
	Name       string
	Triangles  []Triangle
	Lights     []Light
	Ambient    Vec3
	Background Vec3
	Camera     Camera
}

func (s *Scene) AddTriangle(name string, a, b, c, albedo Vec3) int {
	id := len(s.Triangles)
	s.Triangles = append(s.Triangles, Triangle{
		ID:     id,
		Name:   name,
		A:      a,
		B:      b,
		C:      c,
		Albedo: albedo,
	})
	return id
}

type TraceInfo struct {
	Hits  []Hit
	Hit   Hit
	OK    bool
	Tests int
}

type Tracer interface {
	Trace(ray Ray) TraceInfo
}

func TraceReference(scene *Scene, ray Ray) TraceInfo {
	info := TraceInfo{Tests: len(scene.Triangles)}
	for i := range scene.Triangles {
		if hit, ok := scene.Triangles[i].Intersect(ray); ok {
			info.Hits = append(info.Hits, hit)
		}
	}
	sortedHits(info.Hits)
	if hit, ok := nearestHit(info.Hits); ok {
		info.Hit = hit
		info.OK = true
	}
	return info
}

func Shade(scene *Scene, ray Ray, info TraceInfo) Vec3 {
	if !info.OK {
		t := 0.5 * (ray.Dir.Y + 1)
		horizon := V(0.055, 0.075, 0.12)
		zenith := scene.Background
		return horizon.Lerp(zenith, t)
	}

	hit := info.Hit
	base := scene.Triangles[hit.ID].Albedo
	color := base.Mul(scene.Ambient)
	view := ray.Dir.Scale(-1)

	for _, light := range scene.Lights {
		toLight := light.Position.Sub(hit.Point)
		distance := toLight.Len()
		lightDir := toLight.Normalize()
		diffuse := math.Abs(hit.Normal.Dot(lightDir))
		attenuation := light.Intensity / (1 + 0.08*distance + 0.015*distance*distance)
		color = color.Add(base.Mul(light.Color).Scale(diffuse * attenuation))

		halfDir := lightDir.Add(view).Normalize()
		specular := math.Pow(math.Max(0, math.Abs(hit.Normal.Dot(halfDir))), 48)
		color = color.Add(light.Color.Scale(specular * attenuation * 0.35))
	}

	edgeTint := 0.08 * (1 - math.Abs(ray.Dir.Dot(hit.Normal)))
	return color.Add(V(edgeTint, edgeTint*0.7, edgeTint*0.3))
}
