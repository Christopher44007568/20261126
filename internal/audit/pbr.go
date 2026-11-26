package audit

import "math"

func fresnelSchlick(cosine float64, f0 Vec3) Vec3 {
	factor := math.Pow(clamp01(1-cosine), 5)
	return Vec3{
		f0.X + (1-f0.X)*factor,
		f0.Y + (1-f0.Y)*factor,
		f0.Z + (1-f0.Z)*factor,
	}
}

func distributionGGX(normalDotHalf, roughness float64) float64 {
	alpha := roughness * roughness
	alphaSquared := alpha * alpha
	denominator := normalDotHalf*normalDotHalf*(alphaSquared-1) + 1
	return alphaSquared / (math.Pi * denominator * denominator)
}

func geometrySchlickGGX(value, roughness float64) float64 {
	k := (roughness + 1) * (roughness + 1) / 8
	return value / (value*(1-k) + k)
}

func toneMapACES(value Vec3) Vec3 {
	return Vec3{
		acesChannel(value.X),
		acesChannel(value.Y),
		acesChannel(value.Z),
	}.Clamp01()
}

func acesChannel(value float64) float64 {
	value = math.Max(0, value)
	return (value*2.51 + 0.03) * value / (value*(2.43*value+0.59) + 0.14)
}

func blendColor(base, target Vec3, amount float64) Vec3 {
	return Vec3{
		mix(base.X, target.X, amount),
		mix(base.Y, target.Y, amount),
		mix(base.Z, target.Z, amount),
	}
}

func colorOne(value float64) Vec3 {
	return Vec3{value, value, value}
}
