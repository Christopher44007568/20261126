package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testCamera(width, height int) Camera {
	camera := DefaultCamera()
	camera.Width = width
	camera.Height = height
	return camera
}

func TestBVHMatchesBruteAcrossDeformation(t *testing.T) {
	character := NewSoftCharacter()
	camera := testCamera(120, 80)
	for _, progress := range []float64{0, 0.35, 0.7, 1} {
		mesh := character.Pose(progress)
		tree := BuildBVH(mesh, 4)
		for y := 0; y < camera.Height; y++ {
			for x := 0; x < camera.Width; x++ {
				point := camera.PointAt(x, y)
				gotTri, gotHit := tree.IntersectPoint(mesh, point)
				wantTri, wantHit := BruteIntersect(mesh, point)
				if gotHit != wantHit || gotTri != wantTri {
					t.Fatalf("progress %.2f pixel (%d,%d): bvh=(%d,%v) brute=(%d,%v)",
						progress, x, y, gotTri, gotHit, wantTri, wantHit)
				}
			}
		}
	}
}

func TestFastRefitDegradesAfterLargeDeformation(t *testing.T) {
	character := NewSoftCharacter()
	bent := character.Pose(0)
	stretched := character.Pose(1)
	tree := BuildBVH(bent, 4)
	tree.RefitFast(stretched)

	metrics := tree.AssessBounds(stretched)
	if !DefaultBoundsBudget().Exceeded(metrics) {
		t.Fatalf("expected degradation over budget, got %+v", metrics)
	}

	camera := testCamera(120, 80)
	_, mode := RenderBVH(stretched, tree, camera)
	_, truth := RenderBrute(stretched, camera)
	stats := CompareCoverage(truth, mode)
	if stats.MissedPixels == 0 {
		t.Fatal("expected translated leaf boxes to miss pixels")
	}
	if stats.FalseHitPixels == 0 {
		t.Fatal("expected wrong-triangle pixels to be counted as false hits")
	}
}

func TestAdaptiveRebuildsWhenBudgetExceeded(t *testing.T) {
	config := DefaultSimulationConfig()
	config.Frames = 18
	config.Camera = testCamera(120, 80)
	result := config.RunAdaptive()

	if len(result.RebuildFrames) == 0 {
		t.Fatal("adaptive path did not schedule a rebuild")
	}
	if result.Stats.EmptyBackground {
		t.Fatal("adaptive path returned an empty background")
	}
	if result.Coverage.Count == 0 {
		t.Fatal("adaptive coverage is empty")
	}
	if result.Stats.MissedPixels != 0 || result.Stats.FalseHitPixels != 0 {
		t.Fatalf("adaptive final frame differs from brute force: %+v", result.Stats)
	}
}

func TestAdaptiveEmptyFrameForcesRebuild(t *testing.T) {
	config := DefaultSimulationConfig()
	config.Camera = testCamera(80, 60)
	mesh := NewSoftCharacter().Pose(1)
	_, truth := RenderBrute(mesh, config.Camera)

	badTree := BuildBVH(mesh, config.LeafSize)
	badTree.Root.Bounds = AABB{
		Min: Vec2{X: 100, Y: 100},
		Max: Vec2{X: 101, Y: 101},
	}
	image, coverage, rebuiltTree, fallback, elapsed := config.renderAdaptiveFrame(mesh, badTree, truth)
	if !fallback {
		t.Fatal("expected empty adaptive frame to trigger a rebuild")
	}
	if elapsed <= 0 || image == nil || coverage.Count == 0 {
		t.Fatalf("fallback did not produce a rebuilt frame: elapsed=%v hits=%d", elapsed, coverage.Count)
	}
	metrics := rebuiltTree.AssessBounds(mesh)
	if metrics.MaxLeafEscape != 0 || metrics.RootEscape != 0 {
		t.Fatalf("rebuilt tree is not exact: %+v", metrics)
	}
}

func TestWritePPM(t *testing.T) {
	image := NewImage(3, 2)
	image.Fill(RGB{R: 1, G: 2, B: 3})
	path := filepath.Join(t.TempDir(), "test.ppm")
	if err := WritePPM(path, image); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(len("P6\n3 2\n255\n") + 3*2*3)
	if info.Size() != want {
		t.Fatalf("ppm size=%d want=%d", info.Size(), want)
	}
}

func BenchmarkRefitVsRebuild(b *testing.B) {
	character := NewSoftCharacter()
	mesh := character.Pose(0.7)
	camera := testCamera(80, 60)
	b.Run("refit", func(b *testing.B) {
		tree := BuildBVH(character.Pose(0), 4)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tree.RefitFast(mesh)
			_, _ = RenderBVH(mesh, tree, camera)
		}
	})
	b.Run("rebuild", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tree := BuildBVH(mesh, 4)
			_, _ = RenderBVH(mesh, tree, camera)
		}
	})
}
