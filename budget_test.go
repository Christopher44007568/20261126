package main

import (
	"errors"
	"testing"
)

func testScene() (*Mesh, *Mesh, Camera, BudgetPlan) {
	detailed := NewCrest(28, "detailed-test")
	coarse := NewCrest(8, "coarse-test")
	camera := DefaultCamera(144, 108)
	plan, _ := NewBudgetPlan(detailed, coarse)
	return detailed, coarse, camera, *plan
}

func TestBudgetTiersFitAndDescend(t *testing.T) {
	detailed, coarse, _, plan := testScene()
	if !(plan.High.LimitBytes > plan.Medium.LimitBytes && plan.Medium.LimitBytes > plan.Low.LimitBytes) {
		t.Fatalf("budgets are not descending: %+v", plan)
	}
	if plan.High.RequiredBytes > plan.High.LimitBytes {
		t.Fatal("full package does not fit high budget")
	}
	if plan.Medium.RequiredBytes > plan.Medium.LimitBytes {
		t.Fatal("compressed package does not fit medium budget")
	}
	if plan.Low.RequiredBytes > plan.Low.LimitBytes {
		t.Fatal("fallback package does not fit low budget")
	}
	if plan.High.RequiredBytes <= plan.Medium.LimitBytes {
		t.Fatal("medium budget unexpectedly fits full acceleration structure")
	}
	if plan.Medium.RequiredBytes <= plan.Low.LimitBytes {
		t.Fatal("low budget unexpectedly fits compressed acceleration structure")
	}

	full := NewFullPackage(detailed, 1)
	compressed, err := NewCompressedPackage(detailed, 2)
	if err != nil {
		t.Fatal(err)
	}
	fallback := NewFallbackPackage(coarse, 3)
	if compressed.Bytes >= full.Bytes {
		t.Fatalf("compressed bytes %d not smaller than full %d", compressed.Bytes, full.Bytes)
	}
	if fallback.Bytes >= compressed.Bytes {
		t.Fatalf("fallback bytes %d not smaller than compressed %d", fallback.Bytes, compressed.Bytes)
	}
}

func TestBudgetDowngradeStaleHandlesAndRestore(t *testing.T) {
	detailed, coarse, camera, plan := testScene()
	manager := NewManager(detailed, coarse, plan)

	fullHandle, err := manager.SetBudget(plan.High.LimitBytes, "test high")
	if err != nil {
		t.Fatal(err)
	}
	fullResult, err := manager.Render(fullHandle, camera)
	if err != nil {
		t.Fatal(err)
	}
	if manager.ResidentBytes() > plan.High.LimitBytes {
		t.Fatal("full resident set exceeds budget")
	}

	compressedHandle, err := manager.SetBudget(plan.Medium.LimitBytes, "test medium")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Render(fullHandle, camera); !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("old full handle error = %v, want ErrStaleHandle", err)
	}
	if manager.ResidentBytes() > plan.Medium.LimitBytes {
		t.Fatal("compressed resident set exceeds budget")
	}

	fallbackHandle, err := manager.SetBudget(plan.Low.LimitBytes, "test low")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Render(compressedHandle, camera); !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("old compressed handle error = %v, want ErrStaleHandle", err)
	}
	fallbackResult, err := manager.Render(fallbackHandle, camera)
	if err != nil {
		t.Fatal(err)
	}
	if manager.ResidentBytes() > plan.Low.LimitBytes {
		t.Fatal("fallback resident set exceeds budget")
	}

	diff := AnnotateSilhouetteErrors(fallbackResult.Image, fullResult.Mask, fallbackResult.Mask, 2)
	if diff.DifferentPixels < 40 {
		t.Fatalf("fallback silhouette errors = %d, want visible degradation", diff.DifferentPixels)
	}
	if diff.BorderPixels == 0 {
		t.Fatal("expected error border pixels")
	}

	restoredHandle, err := manager.RestoreFull("test restore")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Render(fallbackHandle, camera); !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("old fallback handle error = %v, want ErrStaleHandle", err)
	}
	restoredResult, err := manager.Render(restoredHandle, camera)
	if err != nil {
		t.Fatal(err)
	}
	if !imagesIdentical(fullResult.Image, restoredResult.Image) {
		t.Fatal("restored image is not identical to original full render")
	}
	if manager.BuildCounts[TierFull] != 2 {
		t.Fatalf("full build count = %d, want rebuild on restore", manager.BuildCounts[TierFull])
	}
	if manager.EvictionCount != 3 {
		t.Fatalf("eviction count = %d, want 3", manager.EvictionCount)
	}
}

func TestEvictedPackageCannotRenderThroughManager(t *testing.T) {
	detailed, coarse, _, plan := testScene()
	manager := NewManager(detailed, coarse, plan)
	handle, err := manager.SetBudget(plan.High.LimitBytes, "high")
	if err != nil {
		t.Fatal(err)
	}
	handle.pkg.Evict()
	camera := DefaultCamera(32, 24)
	_, err = manager.Render(handle, camera)
	if !errors.Is(err, ErrStaleHandle) {
		t.Fatalf("evicted package render error = %v, want ErrStaleHandle", err)
	}
}
