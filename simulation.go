package main

import (
	"math"
	"time"
)

type SimulationConfig struct {
	Frames   int
	LeafSize int
	Camera   Camera
	Budget   BoundsBudget
}

func DefaultSimulationConfig() SimulationConfig {
	return SimulationConfig{
		Frames:   36,
		LeafSize: 4,
		Camera:   DefaultCamera(),
		Budget:   DefaultBoundsBudget(),
	}
}

type ModeResult struct {
	Name           string
	Image          *Image
	Coverage       *Coverage
	Truth          *Coverage
	Stats          CompareStats
	UpdateNanos    int64
	RebuildFrames  []int
	FinalMetrics   BoundsMetrics
	FinalRebuild   bool
	EmptyFallbacks int
	FinalFrame     int
	BuildCount     int
	RefitCount     int
}

type TimingSummary struct {
	RefitMillis   float64
	RebuildMillis float64
}

func (c SimulationConfig) poseAt(frame int) Mesh {
	char := NewSoftCharacter()
	denominator := math.Max(1, float64(c.Frames-1))
	return char.Pose(float64(frame) / denominator)
}

func (c SimulationConfig) RunRefit() ModeResult {
	bvh := BuildBVH(c.poseAt(0), c.LeafSize)
	result := ModeResult{Name: "refit", FinalFrame: c.Frames - 1, BuildCount: 1}
	var elapsed time.Duration
	var finalMesh Mesh
	for frame := 1; frame < c.Frames; frame++ {
		finalMesh = c.poseAt(frame)
		start := time.Now()
		bvh.RefitFast(finalMesh)
		elapsed += time.Since(start)
		result.RefitCount++
	}
	if finalMesh.Triangles == nil {
		finalMesh = c.poseAt(0)
	}
	result.UpdateNanos = elapsed.Nanoseconds()
	result.Image, result.Coverage = RenderBVH(finalMesh, bvh, c.Camera)
	result.FinalMetrics = bvh.AssessBounds(finalMesh)
	_, truth := RenderBrute(finalMesh, c.Camera)
	result.Truth = truth
	result.Stats = CompareCoverage(truth, result.Coverage)
	return result
}

func (c SimulationConfig) RunRebuild() ModeResult {
	bvh := BuildBVH(c.poseAt(0), c.LeafSize)
	result := ModeResult{Name: "rebuild", FinalFrame: c.Frames - 1, BuildCount: 1}
	var elapsed time.Duration
	var finalMesh Mesh
	for frame := 1; frame < c.Frames; frame++ {
		finalMesh = c.poseAt(frame)
		start := time.Now()
		bvh = BuildBVH(finalMesh, c.LeafSize)
		elapsed += time.Since(start)
		result.BuildCount++
		result.RebuildFrames = append(result.RebuildFrames, frame)
	}
	if finalMesh.Triangles == nil {
		finalMesh = c.poseAt(0)
	}
	result.UpdateNanos = elapsed.Nanoseconds()
	result.FinalRebuild = true
	result.Image, result.Coverage = RenderBVH(finalMesh, bvh, c.Camera)
	result.FinalMetrics = bvh.AssessBounds(finalMesh)
	_, truth := RenderBrute(finalMesh, c.Camera)
	result.Truth = truth
	result.Stats = CompareCoverage(truth, result.Coverage)
	return result
}

func (c SimulationConfig) RunBrute() ModeResult {
	finalMesh := c.poseAt(c.Frames - 1)
	start := time.Now()
	image, coverage := RenderBrute(finalMesh, c.Camera)
	result := ModeResult{
		Name:       "brute",
		Image:      image,
		Coverage:   coverage,
		Truth:      coverage,
		FinalFrame: c.Frames - 1,
	}
	result.UpdateNanos = time.Since(start).Nanoseconds()
	result.Stats = CompareCoverage(coverage, coverage)
	return result
}

func (c SimulationConfig) RunAdaptive() ModeResult {
	bvh := BuildBVH(c.poseAt(0), c.LeafSize)
	result := ModeResult{Name: "adaptive", FinalFrame: c.Frames - 1, BuildCount: 1}
	var elapsed time.Duration
	var finalMesh Mesh
	lastRebuild := -1
	for frame := 1; frame < c.Frames; frame++ {
		finalMesh = c.poseAt(frame)
		start := time.Now()
		bvh.RefitFast(finalMesh)
		elapsed += time.Since(start)
		result.RefitCount++

		start = time.Now()
		metrics := bvh.AssessBounds(finalMesh)
		elapsed += time.Since(start)
		if c.Budget.Exceeded(metrics) {
			start = time.Now()
			bvh = BuildBVH(finalMesh, c.LeafSize)
			elapsed += time.Since(start)
			result.BuildCount++
			result.RebuildFrames = append(result.RebuildFrames, frame)
			lastRebuild = frame
		}
	}
	if finalMesh.Triangles == nil {
		finalMesh = c.poseAt(0)
	}
	result.FinalMetrics = bvh.AssessBounds(finalMesh)
	result.FinalRebuild = lastRebuild == c.Frames-1

	_, truth := RenderBrute(finalMesh, c.Camera)
	result.Truth = truth
	image, coverage, finalTree, fallback, fallbackTime := c.renderAdaptiveFrame(finalMesh, bvh, truth)
	elapsed += fallbackTime
	result.Image, result.Coverage = image, coverage
	bvh = finalTree
	if fallback {
		result.BuildCount++
		result.RebuildFrames = append(result.RebuildFrames, c.Frames-1)
		result.EmptyFallbacks++
		result.FinalRebuild = true
		result.FinalMetrics = bvh.AssessBounds(finalMesh)
	}
	result.UpdateNanos = elapsed.Nanoseconds()
	result.Stats = CompareCoverage(truth, result.Coverage)
	return result
}

func (c SimulationConfig) renderAdaptiveFrame(mesh Mesh, bvh *BVH, truth *Coverage) (*Image, *Coverage, *BVH, bool, time.Duration) {
	image, coverage := RenderBVH(mesh, bvh, c.Camera)
	if coverage.Count != 0 || truth.Count == 0 {
		return image, coverage, bvh, false, 0
	}
	// A stale root can exclude the whole character. Never return that world
	// as a valid background-only image.
	start := time.Now()
	bvh = BuildBVH(mesh, c.LeafSize)
	image, coverage = RenderBVH(mesh, bvh, c.Camera)
	return image, coverage, bvh, true, time.Since(start)
}

func SummarizeTiming(refit, rebuild ModeResult) TimingSummary {
	return TimingSummary{
		RefitMillis:   float64(refit.UpdateNanos) / 1e6,
		RebuildMillis: float64(rebuild.UpdateNanos) / 1e6,
	}
}
