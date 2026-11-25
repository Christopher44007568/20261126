package main

import "math"

var (
	colorAmber   = hexColor(0xffb000)
	colorMagenta = hexColor(0xff2bd6)
)

func (s *Scene) Render(frame int) *Canvas {
	canvas := NewCanvas(s.Width, s.Height)
	samples := [4][2]float64{
		{0.25, 0.25}, {0.75, 0.25},
		{0.25, 0.75}, {0.75, 0.75},
	}

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			var color Vec3
			for _, sample := range samples {
				ray := s.Camera.RayAt(float64(x)+sample[0], float64(y)+sample[1])
				color = color.Add(s.Trace(ray))
			}
			canvas.Set(x, y, color.Mul(0.25))
		}
	}

	for _, in := range s.TLAS.Instances {
		if in.Status == StatusStaleBounds {
			DrawAABBOverlay(canvas, s.Camera, in.Bounds, colorAmber)
		}
	}
	DrawLegend(canvas, frame)
	return canvas
}

type worldHit struct {
	instance int
	triangle int
	u, v     float64
	t        float64
}

func (s *Scene) Trace(ray Ray) Vec3 {
	const maxDistance = 80.0
	best := worldHit{instance: -1, t: maxDistance}

	instanceID, triID, u, v, hitT, ok := s.TLAS.Intersect(ray, maxDistance)
	if ok {
		best = worldHit{instance: instanceID, triangle: triID, u: u, v: v, t: hitT}
	}

	// The diagnostic pass asks the stale-bound instance's immutable BLAS
	// directly. It reveals the current instance even when the stale TLAS box
	// prevented ordinary top-level traversal from visiting it.
	for _, in := range s.TLAS.Instances {
		if in.Status != StatusStaleBounds {
			continue
		}
		triID, hu, hv, ht, directHit := in.Intersect(ray, best.t)
		if directHit && ht < best.t {
			best = worldHit{instance: in.ID, triangle: triID, u: hu, v: hv, t: ht}
		}
	}

	if best.instance < 0 {
		return background(ray)
	}

	in := s.TLAS.Instances[best.instance]
	tri := in.BLAS.Tris[best.triangle]
	localNormal := triangleNormal(&tri, best.u, best.v)
	normal := in.World.Direction(localNormal).Norm()
	if normal.Dot(ray.Dir) > 0 {
		normal = normal.Mul(-1)
	}
	point := ray.Origin.Add(ray.Dir.Mul(best.t))
	base := s.materialColor(in, point)

	diffuse := math.Max(0, normal.Dot(s.Light))
	shadowOrigin := point.Add(normal.Mul(0.015))
	shadowRay := NewRay(shadowOrigin, s.Light)
	shadow := 1.0
	if _, _, _, _, _, blocked := s.TLAS.Intersect(shadowRay, 30); blocked {
		shadow = 0.28
	}

	viewDir := ray.Dir.Mul(-1)
	halfDir := s.Light.Add(viewDir).Norm()
	specular := math.Pow(math.Max(0, normal.Dot(halfDir)), 34) * 0.22
	if in.Status == StatusMissedUpdate || in.Status == StatusStaleBounds {
		lightAmount := 0.52 + 0.48*diffuse
		color := base.Mul(lightAmount).Add(base.Mul(0.20))
		specular *= 1.5
		return color.Add(V(specular, specular, specular))
	}
	lightAmount := 0.34 + 0.66*diffuse*shadow
	color := base.Mul(lightAmount)
	return color.Add(V(specular*shadow, specular*shadow, specular*shadow))
}

func (s *Scene) materialColor(in *Instance, point Vec3) Vec3 {
	if in.ID == IDFloor {
		cellX := int(math.Floor(point.X * 0.55))
		cellZ := int(math.Floor(point.Z * 0.55))
		if (cellX+cellZ)&1 == 0 {
			return hexColor(0xdce4ef)
		}
		return hexColor(0xb9c6d8)
	}
	switch in.Status {
	case StatusStaleBounds:
		return colorAmber
	case StatusMissedUpdate:
		return colorMagenta
	default:
		return in.Color
	}
}

func background(ray Ray) Vec3 {
	blend := 0.5 * (ray.Dir.Y + 1)
	top := hexColor(0x8fc3f2)
	horizon := hexColor(0xeaf5ff)
	return horizon.Mul(1 - blend).Add(top.Mul(blend))
}
