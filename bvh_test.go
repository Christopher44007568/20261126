package main

import (
	"math"
	"testing"
)

func TestBVHMatchesBruteForce(t *testing.T) {
	mesh := NewCrest(28, "test-crest")
	bvh := BuildBVH(mesh.Tris)
	camera := DefaultCamera(120, 90)

	for y := 0; y < camera.Height; y++ {
		for x := 0; x < camera.Width; x++ {
			ray := camera.Ray(x, y)
			got, ok := bvh.Intersect(ray)
			want, wantOK := BruteIntersect(mesh.Tris, ray)
			if ok != wantOK {
				t.Fatalf("pixel %d,%d hit mismatch: bvh=%v brute=%v", x, y, ok, wantOK)
			}
			if ok && math.Abs(got.T-want.T) > 1e-9 {
				t.Fatalf("pixel %d,%d t=%v want %v", x, y, got.T, want.T)
			}
		}
	}
}

func TestQuantizedBVHMatchesQuantizedMeshBruteForce(t *testing.T) {
	mesh := NewCrest(24, "quantized-test")
	qm := QuantizeMesh(mesh, QuantizedLevels)
	qbvh, err := BuildQBVH(qm)
	if err != nil {
		t.Fatal(err)
	}
	tris := qm.Dequantize()
	camera := DefaultCamera(96, 72)

	for y := 0; y < camera.Height; y += 2 {
		for x := 0; x < camera.Width; x += 2 {
			ray := camera.Ray(x, y)
			got, ok := qbvh.Intersect(ray)
			want, wantOK := BruteIntersect(tris, ray)
			if ok != wantOK {
				t.Fatalf("pixel %d,%d compressed hit mismatch: qbvh=%v brute=%v", x, y, ok, wantOK)
			}
			if ok && math.Abs(got.T-want.T) > 1e-8 {
				t.Fatalf("pixel %d,%d compressed t=%v want %v", x, y, got.T, want.T)
			}
		}
	}
}
