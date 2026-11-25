package main

import (
	"math"
	"testing"
)

func nearlyEqual(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("%s = %.12f, want %.12f", name, got, want)
	}
}

func TestEncodeDecodeOctahedralDirections(t *testing.T) {
	directions := []Vec3{
		{X: 1}, {Y: 1}, {Z: 1}, {Z: -1},
		{X: -1}, {Y: -1},
		{X: 1, Y: 0, Z: -1},
		{X: 0, Y: 1, Z: -1},
		{X: -1, Y: 1, Z: -1},
		{X: 0.7, Y: -0.4, Z: -0.59160797831},
	}

	for _, direction := range directions {
		want := Normalize(direction)
		got := DecodeOcta(EncodeOcta(want))
		if math.Abs(Dot(got, want)-1) > 1e-12 {
			t.Fatalf("direction %+v reconstructed as %+v", want, got)
		}
	}
}

func TestNegativeZIsNotReconstructedByAbsoluteValue(t *testing.T) {
	original := Normalize(Vec3{X: 0.32, Y: -0.47, Z: -0.82})
	u, v := EncodeOcta(original)
	decoded := DecodeOcta(u, v)

	if decoded.Z >= 0 {
		t.Fatalf("negative tangent-space z lost sign: got z=%.6f", decoded.Z)
	}
	if math.Abs(Dot(decoded, original)-1) > 1e-12 {
		t.Fatalf("folded decode = %+v, want %+v", decoded, original)
	}

	wrongSign := decoded
	wrongSign.Z = math.Abs(wrongSign.Z)
	wrongSign = Normalize(wrongSign)
	if Dot(wrongSign, original) > 0 {
		t.Fatalf("absolute-value reconstruction unexpectedly matched negative-z normal")
	}
}

func TestAbsoluteThirdComponentCreatesFalseLight(t *testing.T) {
	original := Normalize(Vec3{X: 0.2, Y: 0.3, Z: -1})
	u, v := EncodeOcta(original)

	correct := DecodeOcta(u, v)
	wrong := Normalize(Vec3{
		X: u,
		Y: v,
		Z: math.Abs(1 - math.Abs(u) - math.Abs(v)),
	})
	light := Normalize(Vec3{X: 0.2, Y: 0.3, Z: 0.9})

	if Dot(correct, light) >= 0 {
		t.Fatalf("correct back-facing normal appears lit: %.6f", Dot(correct, light))
	}
	if Dot(wrong, light) <= 0 {
		t.Fatalf("absolute-value reconstruction did not produce the expected false highlight")
	}
}

func TestUnsignedBoundaryUsesZeroFoldSign(t *testing.T) {
	u, v := EncodeOcta(Vec3{X: -1, Y: 0, Z: 0})
	nearlyEqual(t, "u", u, -1)
	nearlyEqual(t, "v", v, 0)

	decoded := DecodeOcta(u, v)
	nearlyEqual(t, "folded y sign", decoded.Y, 0)
	if decoded.X >= 0 || decoded.Z != 0 {
		t.Fatalf("zero-boundary fold has wrong sign: %+v", decoded)
	}
}

func TestQuantizeDequantizeEndpointsAndSymmetry(t *testing.T) {
	bits := 8
	if got := Quantize(-1, bits); got != 0 {
		t.Fatalf("Quantize(-1) = %d, want 0", got)
	}
	if got := Quantize(1, bits); got != 255 {
		t.Fatalf("Quantize(1) = %d, want 255", got)
	}
	zeroLevel := Quantize(0, bits)
	zero := Dequantize(zeroLevel, bits)
	oneStep := 2.0 / float64(uint(1)<<bits-1)
	if math.Abs(zero) > oneStep {
		t.Fatalf("zero quantization bias %.12f exceeds one step %.12f", zero, oneStep)
	}

	packed := PackNormal(Vec3{Z: 1}, 8)
	if packed.U != 128 || packed.V != 128 {
		t.Fatalf("front-facing normal packed to (%d,%d), want (128,128)", packed.U, packed.V)
	}
}

func TestQuantizedRoundTripError(t *testing.T) {
	original := Normalize(Vec3{X: -0.53, Y: 0.28, Z: -0.8})
	decoded := PackNormal(original, 8).Unpack(8)
	cosAngle := math.Max(-1, math.Min(1, Dot(original, decoded)))
	angleDeg := math.Acos(cosAngle) * 180 / math.Pi

	if angleDeg > 1.0 {
		t.Fatalf("8-bit angular error %.4f degrees exceeds budget", angleDeg)
	}
	if decoded.Z >= 0 {
		t.Fatalf("quantized decode lost negative z: %.6f", decoded.Z)
	}
}

func TestSphereCoversBothTangentSpaceHemispheres(t *testing.T) {
	samples, _ := BuildSphere(64, 8)
	positiveZ, negativeZ := false, false

	for _, row := range samples {
		for _, sample := range row {
			if !sample.Inside {
				continue
			}
			if sample.Original.Z > 0.99 {
				positiveZ = true
			}
			if sample.Original.Z < -0.99 {
				negativeZ = true
			}
		}
	}

	if !positiveZ || !negativeZ {
		t.Fatalf("sphere coverage positive=%v negative=%v", positiveZ, negativeZ)
	}
}
