package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateRoundTripAndSpecialRays(t *testing.T) {
	out := t.TempDir()
	report, err := Generate(out, 160, 90, 160)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !report.OK {
		t.Fatalf("validation report not OK: %+v", report)
	}

	if !report.Structure.AddressesChanged || report.Structure.NodeAddressChanges != report.Structure.NodeCount {
		t.Fatalf("node addresses were not fully relocated: changed=%d total=%d",
			report.Structure.NodeAddressChanges, report.Structure.NodeCount)
	}
	if !report.Structure.TriangleOrderChanged {
		t.Fatal("triangle storage order did not change")
	}
	if len(report.Structure.BoundsMismatches) != 0 || len(report.Structure.TopologyMismatches) != 0 {
		t.Fatalf("unexpected structural mismatches: %+v", report.Structure)
	}
	if report.Structure.DegenerateTriangles == 0 {
		t.Fatal("expected degenerate triangles in fixture")
	}
	if report.Structure.DeepInstancePathLength < 8 {
		t.Fatalf("deep instance path too short: %d", report.Structure.DeepInstancePathLength)
	}
	if report.Rays.ProbeFailed != 0 {
		t.Fatalf("special probes failed: %+v", report.Rays.Comparisons)
	}
	for _, cmp := range report.Rays.Comparisons {
		if cmp.Mode == "random_world" {
			continue
		}
		if !cmp.OK || cmp.SurfaceID != SurfaceGold {
			t.Fatalf("probe %s/%s did not land on gold surface: %+v", cmp.Mode, cmp.Name, cmp)
		}
	}
	if report.Rays.RandomMismatches != 0 {
		t.Fatalf("random ray mismatches: %d", report.Rays.RandomMismatches)
	}
	if !report.Diff.Identical || report.Diff.DifferentPixels != 0 {
		t.Fatalf("rendered PPMs differ: %+v", report.Diff)
	}
	if report.MismatchVisual.RedPixels == 0 || report.MismatchVisual.GreenPixels == 0 || report.MismatchVisual.YellowPixels == 0 {
		t.Fatalf("mismatch diagnostic lacks color coding: %+v", report.MismatchVisual)
	}

	for _, name := range []string{"original.ppm", "reloaded.ppm", "diff.ppm", "bounds_mismatch.ppm", "accel.asb", "validation.json"} {
		path := filepath.Join(out, name)
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing or empty: %v", name, err)
		}
	}

	for _, name := range []string{"original.ppm", "reloaded.ppm", "diff.ppm", "bounds_mismatch.ppm"} {
		canvas, err := ReadPPM(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("ReadPPM(%s): %v", name, err)
		}
		if canvas.Width != 160 || canvas.Height != 90 {
			t.Fatalf("%s dimensions = %dx%d", name, canvas.Width, canvas.Height)
		}
	}
}

func TestCorruptAssetChecksumRejected(t *testing.T) {
	out := t.TempDir()
	if _, err := Generate(out, 64, 36, 8); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(out, "accel.asb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-8] ^= 0xFF
	corrupt := filepath.Join(out, "corrupt.asb")
	if err := os.WriteFile(corrupt, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadAsset(corrupt); err == nil {
		t.Fatal("expected CRC corruption to be rejected")
	}
}

func TestInjectedBoundsMismatchIsDetected(t *testing.T) {
	scene := NewDemoScene()
	path := filepath.Join(t.TempDir(), "accel.asb")
	if _, err := SaveAsset(path, scene); err != nil {
		t.Fatal(err)
	}
	rt, _, err := LoadAsset(path)
	if err != nil {
		t.Fatal(err)
	}
	rt.Nodes[0].Bounds.Min.X += 0.73
	report := ValidateStructure(scene, rt)
	if report.OK {
		t.Fatal("validation unexpectedly passed after bounds corruption")
	}
	if len(report.BoundsMismatches) == 0 {
		t.Fatal("expected bounds mismatch report")
	}
}
