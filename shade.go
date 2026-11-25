package rayao

import (
	"math"
	"math/bits"
)

func ambientVisibility(scene *Scene, primary Hit, opts Options, px, py int) (visibility float64, hits, rays int) {
	tangent, bitangent := orthonormalBasis(primary.Normal)
	origin := primary.Point.Add(primary.Normal.Mul(opts.SurfaceOffset))
	rotation := 2.0 * math.Pi * pixelHash(px, py)
	cosRot, sinRot := math.Cos(rotation), math.Sin(rotation)

	occluded := 0
	for i := 0; i < opts.AOSamples; i++ {
		u1 := (float64(i) + 0.5) / float64(opts.AOSamples)
		u2 := hammersleyInverse(i + 1)
		sample := cosineHemisphere(u1, u2)
		sx := sample.X*cosRot - sample.Y*sinRot
		sy := sample.X*sinRot + sample.Y*cosRot
		direction := tangent.Mul(sx).
			Add(bitangent.Mul(sy)).
			Add(primary.Normal.Mul(sample.Z)).
			Normalize()

		ray := Ray{
			Origin:    origin,
			Direction: direction,
			MinT:      opts.SecondaryMinT,
			MaxT:      opts.AORadius,
		}
		query := RayQuery{Ray: ray, IgnorePrimitive: primary.Primitive}
		if _, blocked := scene.AnyHit(query); blocked {
			occluded++
		}
	}
	total := opts.AOSamples
	return 1.0 - float64(occluded)/float64(total), occluded, total
}

func shadeHit(scene *Scene, hit Hit, visibility float64, incident Vec3, opts Options) Vec3 {
	material := Material{Color: Vec3{X: 0.7, Y: 0.7, Z: 0.7}}
	if hit.Material >= 0 && hit.Material < len(scene.Materials) {
		material = scene.Materials[hit.Material]
	}

	normal := hit.Normal
	lightDirection := keyLightDirection()
	diffuse := math.Max(0, normal.Dot(lightDirection))
	hemisphere := 0.5 + 0.5*normal.Y
	ambient := 0.30 + 0.20*hemisphere
	aoFactor := 0.28 + 0.72*visibility

	lit := ambient + 0.72*diffuse
	color := material.Color.Mul(lit * aoFactor)

	viewDistance := hit.T
	fog := clamp01((viewDistance - 5.2) / 3.8)
	return color.Lerp(background(incident), fog*0.72)
}

func keyLightDirection() Vec3 {
	return Vec3{X: 0.34, Y: 0.88, Z: 0.32}.Normalize()
}

// keyLightVisibility casts one any-hit query toward the directional key light.
// Faces oriented away from the light already receive zero diffuse term, so they
// do not need a self-occlusion query here.
func keyLightVisibility(scene *Scene, hit Hit, opts Options) (visibility float64, hits, rays int) {
	lightDirection := keyLightDirection()
	if hit.Normal.Dot(lightDirection) <= 0 {
		return 1, 0, 0
	}
	origin := hit.Point.Add(hit.Normal.Mul(opts.SurfaceOffset))
	ray := Ray{
		Origin:    origin,
		Direction: lightDirection,
		MinT:      opts.SecondaryMinT,
		MaxT:      math.Inf(1),
	}
	if _, blocked := scene.AnyHit(RayQuery{Ray: ray, IgnorePrimitive: hit.Primitive}); blocked {
		return 0, 1, 1
	}
	return 1, 0, 1
}

func orthonormalBasis(n Vec3) (tangent, bitangent Vec3) {
	helper := Vec3{X: 1, Z: 0}
	if math.Abs(n.X) > 0.9 {
		helper = Vec3{Y: 1}
	}
	tangent = helper.Cross(n).Normalize()
	bitangent = n.Cross(tangent).Normalize()
	return tangent, bitangent
}

func cosineHemisphere(u1, u2 float64) Vec3 {
	radius := math.Sqrt(u1)
	angle := 2.0 * math.Pi * u2
	return Vec3{
		X: radius * math.Cos(angle),
		Y: radius * math.Sin(angle),
		Z: math.Sqrt(math.Max(0, 1.0-u1)),
	}
}

func hammersleyInverse(index int) float64 {
	const inverse2To64 = 5.42101086242752217e-20
	return float64(bits.Reverse64(uint64(index))) * inverse2To64
}

func pixelHash(x, y int) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xBF58476D1CE4E5B9
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return float64(h&((uint64(1)<<53)-1)) / float64(uint64(1)<<53)
}
