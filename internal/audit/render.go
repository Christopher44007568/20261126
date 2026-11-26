package audit

import "math"

const (
	cameraDistance = 3.2
	verticalFOV    = 38 * math.Pi / 180
	sphereRadius   = 1.0
)

type RenderVariant struct {
	ForceSRGBAlbedo bool
}

// Render returns a display-encoded sRGB image. The variant only changes the
// albedo interpretation. Normal and roughness always feed the shader as raw
// linear data on both paths.
func Render(manifest Manifest, textures MaterialTextures, variant RenderVariant) *Image {
	picture := NewImage(manifest.Render.Width, manifest.Render.Height)
	aspect := float64(manifest.Render.Width) / float64(manifest.Render.Height)
	tanHalfFOV := math.Tan(verticalFOV / 2)
	sampleCount := manifest.Render.Samples
	camera := Vec3{0, 0, cameraDistance}
	toLight := lightDirection(manifest.Light)

	for row := 0; row < picture.Height; row++ {
		for column := 0; column < picture.Width; column++ {
			var accumulated Vec3
			for sampleY := 0; sampleY < sampleCount; sampleY++ {
				for sampleX := 0; sampleX < sampleCount; sampleX++ {
					fractionX := (float64(sampleX) + 0.5) / float64(sampleCount)
					fractionY := (float64(sampleY) + 0.5) / float64(sampleCount)
					ndcX := ((float64(column)+fractionX)/float64(picture.Width)*2 - 1) * aspect * tanHalfFOV
					ndcY := (1 - (float64(row)+fractionY)/float64(picture.Height)*2) * tanHalfFOV
					ray := Vec3{ndcX, ndcY, -1}.Normalize()

					hit, found := intersectSphere(camera, ray)
					if !found {
						accumulated = accumulated.Add(backgroundRadiance(ndcX, ndcY))
						continue
					}
					accumulated = accumulated.Add(shadeHit(
						manifest,
						textures,
						variant,
						hit,
						camera,
						toLight,
					))
				}
			}

			sampleArea := float64(sampleCount * sampleCount)
			linear := accumulated.Scale(1 / sampleArea)
			linear = linear.Scale(manifest.Render.Exposure)
			display := toneMapACES(linear)
			picture.Set(column, row, Vec3{
				LinearToSRGB(display.X),
				LinearToSRGB(display.Y),
				LinearToSRGB(display.Z),
			})
		}
	}
	return picture
}

func lightDirection(light LightConfig) Vec3 {
	azimuth := light.AzimuthDegrees * math.Pi / 180
	elevation := light.ElevationDegrees * math.Pi / 180
	return Vec3{
		math.Cos(elevation) * math.Sin(azimuth),
		math.Sin(elevation),
		math.Cos(elevation) * math.Cos(azimuth),
	}.Normalize()
}

func intersectSphere(origin, direction Vec3) (Vec3, bool) {
	halfB := origin.Dot(direction)
	c := origin.Dot(origin) - sphereRadius*sphereRadius
	discriminant := halfB*halfB - c
	if discriminant < 0 {
		return Vec3{}, false
	}
	root := math.Sqrt(discriminant)
	distance := -halfB - root
	if distance <= 0 {
		distance = -halfB + root
	}
	if distance <= 0 {
		return Vec3{}, false
	}
	return origin.Add(direction.Scale(distance)), true
}

func shadeHit(manifest Manifest, textures MaterialTextures, variant RenderVariant, hit, camera, toLight Vec3) Vec3 {
	objectNormal := hit.Normalize()
	viewDirection := camera.Sub(hit).Normalize()
	u, v := sphereUV(objectNormal)

	interpretation := textures.Albedo.DeclaredColorSpace
	if variant.ForceSRGBAlbedo {
		interpretation = ColorSpaceSRGB
	}
	albedo := textures.Albedo.SampleAlbedo(u, v, interpretation).Clamp01()
	shadingNormal := tangentSpaceNormal(
		textures.Normal.SampleNormal(u, v),
		objectNormal,
		manifest.Material.NormalStrength,
	)
	roughness := clamp(textures.Roughness.SampleRoughness(u, v), 0.045, 1)
	return evaluateBRDF(manifest, albedo, shadingNormal, viewDirection, toLight, roughness)
}

func sphereUV(normal Vec3) (float64, float64) {
	u := math.Atan2(normal.X, normal.Z)/(2*math.Pi) + 0.5
	v := math.Acos(clamp(normal.Y, -1, 1)) / math.Pi
	return u, v
}

func tangentSpaceNormal(encoded Vec3, objectNormal Vec3, strength float64) Vec3 {
	mapNormal := Vec3{
		(2*encoded.X - 1) * strength,
		(2*encoded.Y - 1) * strength,
		2*encoded.Z - 1,
	}
	if mapNormal.Length() <= 1e-8 {
		return objectNormal
	}
	tangent := Vec3{objectNormal.Z, 0, -objectNormal.X}
	if tangent.Length() < 1e-8 {
		tangent = Vec3{1, 0, 0}
	}
	tangent = tangent.Normalize()
	bitangent := objectNormal.Cross(tangent).Normalize()
	shadingNormal := tangent.Scale(mapNormal.X).
		Add(bitangent.Scale(mapNormal.Y)).
		Add(objectNormal.Scale(mapNormal.Z)).
		Normalize()
	return shadingNormal
}

func evaluateBRDF(manifest Manifest, albedo, normal, viewDirection, toLight Vec3, roughness float64) Vec3 {
	normalDotLight := math.Max(normal.Dot(toLight), 0)
	normalDotView := math.Max(normal.Dot(viewDirection), 1e-4)
	halfVector := toLight.Add(viewDirection).Normalize()
	normalDotHalf := math.Max(normal.Dot(halfVector), 0)

	dielectricF0 := 0.16 * manifest.Material.Specular * manifest.Material.Specular
	f0 := blendColor(colorOne(dielectricF0), albedo, manifest.Material.Metallic)
	fresnel := fresnelSchlick(normalDotView, f0)

	var direct Vec3
	if normalDotLight > 0 && normalDotHalf > 0 {
		distribution := distributionGGX(normalDotHalf, roughness)
		geometry := geometrySchlickGGX(normalDotLight, roughness) *
			geometrySchlickGGX(normalDotView, roughness)
		denominator := 4*normalDotLight*normalDotView + 1e-4
		specular := fresnel.Scale(distribution * geometry / denominator)

		diffuseWeight := Vec3{
			(1 - fresnel.X) * (1 - manifest.Material.Metallic),
			(1 - fresnel.Y) * (1 - manifest.Material.Metallic),
			(1 - fresnel.Z) * (1 - manifest.Material.Metallic),
		}
		diffuse := albedo.Scale(1 / math.Pi).Mul(diffuseWeight)
		direct = diffuse.Add(specular).Scale(normalDotLight * manifest.Light.Intensity)
	}

	ambient := albedo.Scale(manifest.Light.Ambient * (1 - manifest.Material.Metallic))
	return direct.Add(ambient)
}

func backgroundRadiance(ndcX, ndcY float64) Vec3 {
	radiusSquared := ndcX*ndcX + ndcY*ndcY
	falloff := math.Exp(-1.8 * radiusSquared)
	value := 0.003 + 0.016*falloff
	return Vec3{value * 0.80, value * 0.88, value}
}
