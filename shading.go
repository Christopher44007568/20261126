package main

import "math"

func (r *renderer) rayColor(ray Ray, depth int) Vec3 {
	hit, ok := traceScene(r.scene, ray, rayEpsilon, math.Inf(1))
	if !ok {
		return r.scene.background(ray.Dir)
	}
	return r.shade(hit, ray.Dir, depth)
}

func (r *renderer) shade(hit SurfaceHit, rayDir Vec3, depth int) Vec3 {
	groupID := r.mapping[hit.Object.ID]
	material, ok := r.scene.Catalog.MaterialForGroup(groupID)
	if !ok {
		return Vec3{1, 0, 1}
	}
	if material.Emission.LengthSquared() > 0 {
		return material.Emission
	}
	if material.Transparency > 0 {
		return r.shadeGlass(hit, rayDir, depth, material)
	}

	base := surfaceAlbedo(material, hit)
	normal := hit.Normal
	ambient := base.Mul(0.075 + 0.04*math.Max(0, normal.Y))
	color := ambient
	for _, light := range r.lights {
		toLight := light.Position.Sub(hit.Point)
		distance := toLight.Length()
		if distance < rayEpsilon {
			continue
		}
		lightDir := toLight.Mul(1 / distance)
		nDotL := normal.Dot(lightDir)
		if nDotL <= 0 {
			continue
		}
		shadowRay := Ray{
			Origin: hit.Point.Add(normal.Mul(rayEpsilon * 4)),
			Dir:    lightDir,
		}
		if r.shadowed(shadowRay, distance-rayEpsilon*8) {
			continue
		}
		attenuation := 1 / (1 + 0.045*distance*distance)
		radiance := light.Emission.Mul(attenuation)
		color = color.Add(base.MulVec(radiance).Mul(nDotL))

		viewDir := rayDir.Mul(-1)
		halfDir := lightDir.Add(viewDir).Normalize()
		spec := math.Pow(math.Max(0, normal.Dot(halfDir)), material.Shininess)
		color = color.Add(radiance.Mul(spec * material.Specular * 0.55))
	}

	if material.Reflectivity > 0 && depth < maxTraceDepth {
		reflected := r.scene.background(reflect(rayDir, normal))
		if depth+1 < maxTraceDepth {
			reflected = r.rayColor(Ray{
				Origin: hit.Point.Add(normal.Mul(rayEpsilon * 4)),
				Dir:    reflect(rayDir, normal),
			}, depth+1)
		}
		color = mix(color, reflected.MulVec(material.Albedo), material.Reflectivity)
	}
	return color
}

func (r *renderer) shadeGlass(hit SurfaceHit, rayDir Vec3, depth int, material Material) Vec3 {
	normal := hit.Normal
	cosine := math.Max(0, -rayDir.Dot(normal))
	fresnel := schlick(cosine, material.IOR)

	reflected := r.scene.background(reflect(rayDir, normal))
	if depth < maxTraceDepth {
		reflected = r.rayColor(Ray{
			Origin: hit.Point.Add(normal.Mul(rayEpsilon * 4)),
			Dir:    reflect(rayDir, normal),
		}, depth+1)
	}

	transmitted := r.scene.background(rayDir)
	if depth < maxTraceDepth {
		transmitted = r.rayColor(Ray{
			Origin: hit.Point.Add(rayDir.Mul(rayEpsilon * 4)),
			Dir:    rayDir,
		}, depth+1)
	}
	tint := mix(Vec3{1, 1, 1}, material.Albedo, 0.22)
	return transmitted.MulVec(tint).Mul(1 - fresnel).Add(reflected.Mul(fresnel))
}

func (r *renderer) shadowed(ray Ray, maxDistance float64) bool {
	tMin := rayEpsilon
	for bounce := 0; bounce < 6; bounce++ {
		hit, ok := traceScene(r.scene, ray, tMin, maxDistance)
		if !ok {
			return false
		}
		groupID := r.mapping[hit.Object.ID]
		material, valid := r.scene.Catalog.MaterialForGroup(groupID)
		if !valid || material.Transparency < 0.5 {
			return true
		}
		ray.Origin = hit.Point.Add(ray.Dir.Mul(rayEpsilon * 4))
		tMin = rayEpsilon
	}
	return true
}

func surfaceAlbedo(material Material, hit SurfaceHit) Vec3 {
	color := material.Albedo
	n := hit.Normal
	switch material.Pattern {
	case PatternBrick:
		u, v := hit.Point.X, hit.Point.Y
		if math.Abs(n.X) > 0.5 {
			u = hit.Point.Z
		} else if math.Abs(n.Y) > 0.5 {
			u, v = hit.Point.X, hit.Point.Z
		}
		const brickHeight = 0.27
		const brickWidth = 0.57
		row := math.Floor(v / brickHeight)
		if int(math.Abs(row))%2 == 1 {
			u += brickWidth * 0.5
		}
		column := math.Floor(u / brickWidth)
		fv := v/brickHeight - row
		fu := u/brickWidth - column
		mortar := fu < 0.045 || fu > 0.955 || fv < 0.10 || fv > 0.90
		if mortar {
			color = color.Mul(0.62)
		} else {
			color = color.Mul(0.86 + 0.18*hash2(row, column))
		}
	case PatternChecker:
		tile := int(math.Floor(hit.Point.X/0.75)) + int(math.Floor(hit.Point.Z/0.75))
		if tile&1 != 0 {
			color = color.Mul(0.78)
		}
		color = color.Mul(0.92 + 0.08*hash2(
			math.Floor(hit.Point.X/3), math.Floor(hit.Point.Z/3)))
	}
	return color
}

func hash2(a, b float64) float64 {
	value := math.Sin(a*12.9898+b*78.233) * 43758.5453
	return value - math.Floor(value)
}
