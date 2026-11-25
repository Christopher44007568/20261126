package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func testBoxTLAS(t *testing.T) (*TLAS, *Instance) {
	t.Helper()
	blas := NewBLAS("test-box", BoxMesh(V(0, 0, 0), V(0.5, 0.5, 0.5)))
	instance := NewInstance(0, "box", blas, Identity(), V(1, 1, 1), StatusNormal)
	return BuildTLAS([]*Instance{instance}), instance
}

func TestTLASUpdateAndRefitMovesHit(t *testing.T) {
	tlas, _ := testBoxTLAS(t)

	initialRay := NewRay(V(0, 0, 5), V(0, 0, -1))
	if _, _, _, _, _, ok := tlas.Intersect(initialRay, 10); !ok {
		t.Fatal("expected initial TLAS hit")
	}

	tlas.UpdateTransform(0, Translate(V(3, 0, 0)), 1)
	tlas.Refit()

	oldRay := NewRay(V(0, 0, 5), V(0, 0, -1))
	if _, _, _, _, _, ok := tlas.Intersect(oldRay, 10); ok {
		t.Fatal("refitted TLAS should not hit moved instance at old location")
	}

	newRay := NewRay(V(3, 0, 5), V(0, 0, -1))
	if _, _, _, _, _, ok := tlas.Intersect(newRay, 10); !ok {
		t.Fatal("refitted TLAS should hit instance at new location")
	}
}

func TestStaleBoundsCanHideMovedInstance(t *testing.T) {
	tlas, instance := testBoxTLAS(t)

	tlas.UpdateTransformWithStaleBounds(0, Translate(V(3, 0, 0)), 1)
	tlas.Refit()

	newRay := NewRay(V(3, 0, 5), V(0, 0, -1))
	if _, _, _, _, _, ok := tlas.Intersect(newRay, 10); ok {
		t.Fatal("stale TLAS bounds should prevent traversal at new location")
	}

	if _, _, _, _, directHit := instance.Intersect(newRay, 10); !directHit {
		t.Fatal("immutable BLAS should still contain the moved instance when asked directly")
	}
}

func TestGenerateFourPPMFiles(t *testing.T) {
	out := t.TempDir()
	if err := Generate(out, 160, 100); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	for frame := 0; frame < 4; frame++ {
		path := filepath.Join(out, fmt.Sprintf("pose_%d.ppm", frame))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if len(data) < 16 || string(data[:2]) != "P6" {
			t.Fatalf("%s is not a binary PPM file", path)
		}
	}

	if _, err := os.Stat(filepath.Join(out, "report.json")); err != nil {
		t.Fatalf("report.json was not generated: %v", err)
	}
}
