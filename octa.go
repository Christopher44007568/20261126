package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

type PackedNormal struct {
	U, V uint16
}

func Normalize(v Vec3) Vec3 {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 {
		return Vec3{Z: 1}
	}
	return Vec3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

func Dot(a, b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func signNotZero(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func EncodeOcta(normal Vec3) (u, v float64) {
	n := Normalize(normal)
	l1 := math.Abs(n.X) + math.Abs(n.Y) + math.Abs(n.Z)
	u = n.X / l1
	v = n.Y / l1

	if n.Z < 0 {
		u, v = (1-math.Abs(v))*signNotZero(u), (1-math.Abs(u))*signNotZero(v)
	}
	return u, v
}

func DecodeOcta(u, v float64) Vec3 {
	x, y := u, v
	z := 1 - math.Abs(x) - math.Abs(y)

	if z < 0 {
		foldedX := (1 - math.Abs(y)) * signNotZero(x)
		foldedY := (1 - math.Abs(x)) * signNotZero(y)
		x, y = foldedX, foldedY
	}

	return Normalize(Vec3{X: x, Y: y, Z: z})
}

func Quantize(value float64, bits int) uint16 {
	if bits < 1 || bits > 16 {
		panic("quantization bits must be between 1 and 16")
	}
	maxValue := float64(uint(1)<<bits - 1)
	scaled := (value + 1) * 0.5 * maxValue
	scaled = math.Max(0, math.Min(maxValue, math.Round(scaled)))
	return uint16(scaled)
}

func Dequantize(value uint16, bits int) float64 {
	if bits < 1 || bits > 16 {
		panic("quantization bits must be between 1 and 16")
	}
	maxValue := float64(uint(1)<<bits - 1)
	return float64(value)/maxValue*2 - 1
}

func PackNormal(normal Vec3, bits int) PackedNormal {
	u, v := EncodeOcta(normal)
	return PackedNormal{U: Quantize(u, bits), V: Quantize(v, bits)}
}

func (p PackedNormal) Unpack(bits int) Vec3 {
	return DecodeOcta(Dequantize(p.U, bits), Dequantize(p.V, bits))
}
