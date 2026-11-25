package main

import (
	"math"
	"testing"
)

func TestTraceTransparentReturnsAllLayersInOrder(t *testing.T) {
	ray := Ray{Origin: Vec3{X: -2}, Direction: Vec3{X: 1}}
	result, err := TraceTransparent(ray, DefaultScene(), whiteEnergy(), 0, 4.5, 3)
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"glass", "water", "mist"}
	if len(result.Hits) != len(wantNames) {
		t.Fatalf("got %d hits, want %d", len(result.Hits), len(wantNames))
	}
	for index, want := range wantNames {
		if result.Hits[index].LayerName != want {
			t.Fatalf("hit %d = %q, want %q", index, result.Hits[index].LayerName, want)
		}
	}
	if !result.Complete || result.Unprocessed != nil {
		t.Fatal("full-budget trace must be complete without an unprocessed segment")
	}

	want := 0.92 * 0.78 * 0.86
	if math.Abs(result.Energy.R-want) > 1e-12 {
		t.Fatalf("red energy = %.4f, want %.4f", result.Energy.R, want)
	}
}

func TestTraceTransparentKeepsUnprocessedSegmentWhenBudgetIsExhausted(t *testing.T) {
	ray := Ray{Origin: Vec3{X: -2}, Direction: Vec3{X: 1}}
	result, err := TraceTransparent(ray, DefaultScene(), whiteEnergy(), 0, 4.5, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Hits) != 2 {
		t.Fatalf("got %d processed hits, want 2", len(result.Hits))
	}
	if result.Complete {
		t.Fatal("trace must report incomplete status when budget is smaller than layer count")
	}
	if result.Unprocessed == nil {
		t.Fatal("missing unprocessed segment")
	}
	if result.Unprocessed.T0 != result.Hits[1].T1 {
		t.Fatalf("unprocessed starts at %.3f, want %.3f", result.Unprocessed.T0, result.Hits[1].T1)
	}
	if result.Unprocessed.T1 != 4.5 {
		t.Fatalf("unprocessed ends at %.3f, want 4.5", result.Unprocessed.T1)
	}
	if len(result.RemainingLayers) != 1 || result.RemainingLayers[0] != "mist" {
		t.Fatalf("remaining layers = %v, want [mist]", result.RemainingLayers)
	}
}

func TestTraceTransparentMarksOutOfRangeEnergy(t *testing.T) {
	layers := DefaultScene()
	layers[1].Transmittance.R = 1.25
	ray := Ray{Origin: Vec3{X: -2}, Direction: Vec3{X: 1}}
	result, err := TraceTransparent(ray, layers, whiteEnergy(), 0, 4.5, 3)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Hits[1].OutOfRange {
		t.Fatal("water hit must be marked out of range")
	}
	if result.Hits[1].EnergyAfter.R <= 1 {
		t.Fatalf("red energy = %.3f, want value above one", result.Hits[1].EnergyAfter.R)
	}
}

func TestTraceTransparentMarksNaNAndInfinity(t *testing.T) {
	layers := DefaultScene()
	layers[0].Transmittance.R = math.NaN()
	layers[0].Transmittance.G = math.Inf(1)
	ray := Ray{Origin: Vec3{X: -2}, Direction: Vec3{X: 1}}
	result, err := TraceTransparent(ray, layers, whiteEnergy(), 0, 4.5, 2)
	if err != nil {
		t.Fatal(err)
	}

	for index, hit := range result.Hits {
		if !hit.OutOfRange {
			t.Fatalf("hit %d must be marked out of range", index)
		}
	}
}

func TestTraceTransparentZeroBudgetDoesNotReturnBackgroundEarly(t *testing.T) {
	ray := Ray{Origin: Vec3{X: -2}, Direction: Vec3{X: 1}}
	result, err := TraceTransparent(ray, DefaultScene(), whiteEnergy(), 0, 4.5, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Hits) != 0 {
		t.Fatalf("got %d hits, want zero", len(result.Hits))
	}
	if result.Complete || result.Unprocessed == nil {
		t.Fatal("zero budget must retain the first unprocessed layer instead of completing")
	}
	if len(result.RemainingLayers) != 3 {
		t.Fatalf("remaining layers = %v, want all three", result.RemainingLayers)
	}
}
