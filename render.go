package main

import "math"

type TraceFunc func(Ray) Hit

func RenderTrace(width, height int, camera Camera, trace TraceFunc) *Canvas {
	canvas := NewCanvas(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ray := camera.Ray(x, y)
			hit := trace(ray)
			if hit.OK {
				canvas.Set(x, y, shadeHit(ray, hit))
			} else {
				canvas.Set(x, y, background(ray))
			}
		}
	}
	return canvas
}

func background(ray Ray) RGB {
	t := 0.5 * (ray.Dir.Y + 1)
	low := V(0.018, 0.026, 0.060)
	high := V(0.070, 0.105, 0.180)
	return low.Add(high.Sub(low).Mul(t))
}

func shadeHit(ray Ray, hit Hit) RGB {
	normal := hit.Normal.Normalized()
	light := V(-0.38, 0.62, 0.72).Normalized()
	diffuse := math.Max(0, normal.Dot(light))
	fill := math.Max(0, normal.Dot(V(0.55, -0.25, 0.55).Normalized()))
	ambient := 0.30

	base := hit.Color
	color := base.Mul(ambient + 0.72*diffuse + 0.16*fill)

	view := ray.Dir.Mul(-1).Normalized()
	halfVector := light.Add(view).Normalized()
	specular := math.Pow(math.Max(0, normal.Dot(halfVector)), 42)
	if hit.SurfaceID == SurfaceGold {
		color = color.Add(V(1.0, 0.86, 0.48).Mul(0.55 * specular))
		color = color.Add(V(0.16, 0.10, 0.02))
	} else {
		color = color.Add(V(0.75, 0.86, 1.0).Mul(0.16 * specular))
	}

	// Distance fog keeps the deep instance legible without hiding geometry.
	fog := clamp01((hit.T - 7.0) / 18.0)
	return color.Mul(1 - 0.35*fog).Add(background(ray).Mul(0.35 * fog))
}
