package main

import (
	"fmt"
	"math"
)

const maxTraceDepth = 4

type SurfaceHit struct {
	T      float64
	Point  Vec3
	Normal Vec3
	Object SceneObject
}

type PixelSample struct {
	ObjectID    ObjectID
	HitGroupID  HitGroupID
	MaterialID  MaterialID
	Color       Vec3
	PrimaryRay  Ray
	HasGeometry bool
}

type RenderResult struct {
	Width  int
	Height int
	Pixels []PixelSample
}

func (r *RenderResult) At(x, y int) PixelSample {
	return r.Pixels[y*r.Width+x]
}

type pointLight struct {
	ObjectID ObjectID
	Position Vec3
	Emission Vec3
}

type renderer struct {
	scene   *Scene
	mapping ObjectMapping
	lights  []pointLight
}

func Render(scene *Scene, mapping ObjectMapping, width, height int) (*RenderResult, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("image size must be positive, got %dx%d", width, height)
	}
	if err := mapping.Validate(scene); err != nil {
		return nil, err
	}
	r := &renderer{
		scene: scene, mapping: mapping,
		lights: gatherLights(scene, mapping),
	}
	result := &RenderResult{
		Width: width, Height: height,
		Pixels: make([]PixelSample, width*height),
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ray := cameraRay(scene.Camera, width, height, x, y)
			hit, ok := traceScene(scene, ray, rayEpsilon, math.Inf(1))
			sample := PixelSample{PrimaryRay: ray, HasGeometry: ok}
			if ok {
				groupID := mapping[hit.Object.ID]
				group, _ := scene.Catalog.Group(groupID)
				sample.ObjectID = hit.Object.ID
				sample.HitGroupID = groupID
				sample.MaterialID = group.MaterialID
				sample.Color = r.shade(hit, ray.Dir, 0)
			} else {
				sample.Color = scene.background(ray.Dir)
			}
			result.Pixels[y*width+x] = sample
		}
	}
	return result, nil
}

func cameraRay(camera Camera, width, height, x, y int) Ray {
	forward := camera.Target.Sub(camera.Eye).Normalize()
	right := forward.Cross(camera.Up).Normalize()
	up := right.Cross(forward).Normalize()

	aspect := float64(width) / float64(height)
	halfHeight := math.Tan(camera.FOVY * 0.5)
	u := (2*((float64(x)+0.5)/float64(width)) - 1) * halfHeight * aspect
	v := (1 - 2*((float64(y)+0.5)/float64(height))) * halfHeight
	direction := forward.Add(right.Mul(u)).Add(up.Mul(v)).Normalize()
	return Ray{Origin: camera.Eye, Dir: direction}
}

func traceScene(scene *Scene, ray Ray, tMin, tMax float64) (SurfaceHit, bool) {
	best := SurfaceHit{T: tMax}
	found := false
	for _, object := range scene.Objects {
		t, normal, ok := object.Shape.intersect(ray, tMin, best.T)
		if !ok || t >= best.T {
			continue
		}
		best = SurfaceHit{T: t, Point: ray.At(t), Normal: normal, Object: object}
		found = true
	}
	if !found {
		return SurfaceHit{}, false
	}
	if best.Normal.Dot(ray.Dir) > 0 {
		best.Normal = best.Normal.Mul(-1)
	}
	return best, true
}

func gatherLights(scene *Scene, mapping ObjectMapping) []pointLight {
	lights := make([]pointLight, 0, 1)
	for _, object := range scene.Objects {
		groupID, ok := mapping[object.ID]
		if !ok {
			continue
		}
		material, ok := scene.Catalog.MaterialForGroup(groupID)
		if !ok || material.Emission.LengthSquared() == 0 {
			continue
		}
		lights = append(lights, pointLight{
			ObjectID: object.ID,
			Position: object.Shape.samplePoint(),
			Emission: material.Emission,
		})
	}
	return lights
}

func (s *Scene) background(direction Vec3) Vec3 {
	t := 0.5 * (direction.Y + 1)
	sky := mix(s.SkyBelow, s.SkyAbove, t)
	glow := math.Pow(math.Max(0, direction.Dot(Vec3{-0.45, 0.55, -0.70})), 18)
	return sky.Add(Vec3{1.0, 0.68, 0.36}.Mul(glow * 0.55))
}
