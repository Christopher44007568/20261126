package main

import (
	"os"
	"path/filepath"
	"testing"
)

func assertRayReportPass(t *testing.T, report RayProbeReport) {
	t.Helper()
	if report.Pass {
		return
	}
	for i, mismatch := range report.Mismatches {
		if i >= 12 {
			break
		}
		t.Errorf("ray mismatch %s/%s: %+v", report.Group, mismatch.Name, mismatch)
	}
	t.Fatalf("%s: %d/%d probe rays differ", report.Group, len(report.Mismatches), report.Rays)
}

func assertSuitePass(t *testing.T, result SuiteResult) {
	t.Helper()
	report := result.Report
	if report.Pass {
		return
	}
	for i, mismatch := range result.Mismatches {
		if i >= 12 {
			break
		}
		t.Errorf("pixel mismatch: %+v", mismatch)
	}
	t.Fatalf("%s: %d mismatched pixels; regions=%+v", report.Name, report.MismatchPixels, report.Regions)
}

func TestGrazingEdgeProbesMatchReference(t *testing.T) {
	scene := GrazingScene(240, 150)
	accel := NewSimplifiedAccel(scene.Triangles)
	probes := EdgeProbeRays("grazing", scene)
	assertRayReportPass(t, ValidateRays(scene, accel, probes))

	result := RenderSuite(scene, accel)
	assertSuitePass(t, result)
}

func TestParallelProbesMatchReference(t *testing.T) {
	scene := ParallelScene(240, 150)
	accel := NewSimplifiedAccel(scene.Triangles)
	probes := ParallelProbeRays("parallel", scene)
	assertRayReportPass(t, ValidateRays(scene, accel, probes))

	result := RenderSuite(scene, accel)
	assertSuitePass(t, result)
}

func TestLargeTriangleProbesMatchReference(t *testing.T) {
	scene := LargeTriangleScene(240, 150)
	accel := NewSimplifiedAccel(scene.Triangles)
	probes := append(EdgeProbeRays("large_triangles", scene), RandomProbeRays("large_triangles", scene, 1024, 99173)...)
	assertRayReportPass(t, ValidateRays(scene, accel, probes))

	result := RenderSuite(scene, accel)
	assertSuitePass(t, result)
}

func TestAllSuitesPixelPerfectAndFullHitLists(t *testing.T) {
	for _, scene := range AllScenes(192, 120) {
		t.Run(scene.Name, func(t *testing.T) {
			accel := NewSimplifiedAccel(scene.Triangles)
			result := RenderSuite(scene, accel)
			assertSuitePass(t, result)
			if result.Report.MaxIntersectionCount == 0 {
				t.Fatalf("%s produced no intersections", scene.Name)
			}
			if result.Report.MeanSimplifiedTests >= float64(result.Report.ReferenceTriangles) {
				t.Fatalf("%s BVH did not reduce triangle tests: mean=%.3f total=%d",
					scene.Name, result.Report.MeanSimplifiedTests, result.Report.ReferenceTriangles)
			}

			if result.Report.MaxColorError != 0 {
				t.Fatalf("expected bitwise-shared shading, max color error=%g", result.Report.MaxColorError)
			}

			rayReport := ValidateRays(scene, accel, DefaultProbeSuite(scene, 256))
			assertRayReportPass(t, rayReport)
		})
	}
}

func TestLocalOmittedTriangleFailsDespiteSmallAverage(t *testing.T) {
	scene := standardScene("tiny_local_fault", 192, 120)
	id := scene.AddTriangle("tiny-target",
		V(-0.16, 1.28, -0.2), V(0.16, 1.28, -0.2), V(0, 1.62, -0.2),
		V(1, 0.2, 0.2))
	faulty := OmitTracer{Scene: &scene, Omitted: map[int]bool{id: true}}

	result := RenderSuite(&scene, faulty)
	if result.Report.Pass {
		t.Fatal("omitted local triangle must fail strict comparison")
	}
	if result.Report.MissedPixels == 0 {
		t.Fatalf("expected direct missed pixels, got report %+v", result.Report)
	}
	if len(result.Report.Regions) == 0 {
		t.Fatal("expected a localized error region")
	}
	if result.Report.MeanColorError > 0.02 {
		t.Fatalf("test intended a small average error, got %g", result.Report.MeanColorError)
	}

	probes := EdgeProbeRays("fault", &scene)
	rayReport := ValidateRays(&scene, faulty, probes)
	if rayReport.Pass {
		t.Fatal("omitted triangle must also fail probe validation")
	}
}

func TestRunPipelineWritesRequiredImages(t *testing.T) {
	out := t.TempDir()
	report, err := RunPipeline(out, 96, 60, 48)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass {
		t.Fatalf("pipeline report should pass: %+v", report)
	}
	if !report.FaultControlDetected {
		t.Fatalf("fault injection control was not detected: %+v", report.FaultControl)
	}

	required := []string{
		"reference.png", "simplified.png", "intersection_count.png",
		"error_heatmap.png", "side_by_side.png", "report.json", "index.html",
		filepath.Join("fault_injection", "side_by_side.png"),
		filepath.Join("suites", "grazing", "reference.png"),
		filepath.Join("suites", "parallel", "simplified.png"),
		filepath.Join("suites", "large_triangles", "error_heatmap.png"),
	}
	for _, name := range required {
		path := filepath.Join(out, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing artifact %s: %v", name, err)
		}
	}

	if len(report.PixelSuites) != 3 || len(report.RaySuites) != 3 {
		t.Fatalf("expected three suites, got %d pixel and %d ray", len(report.PixelSuites), len(report.RaySuites))
	}
	for i, suite := range report.PixelSuites {
		if !suite.Pass || !report.RaySuites[i].Pass {
			t.Fatalf("suite %s did not pass", suite.Name)
		}
	}
}
