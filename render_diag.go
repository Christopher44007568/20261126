package main

import (
	"image"
	"sort"
)

func RenderSuite(scene *Scene, tracer Tracer) SuiteResult {
	width := scene.Camera.Width
	height := scene.Camera.Height
	result := SuiteResult{
		Reference:     image.NewRGBA(image.Rect(0, 0, width, height)),
		Simplified:    image.NewRGBA(image.Rect(0, 0, width, height)),
		Intersections: image.NewRGBA(image.Rect(0, 0, width, height)),
		TestCounts:    image.NewRGBA(image.Rect(0, 0, width, height)),
		Heatmap:       image.NewRGBA(image.Rect(0, 0, width, height)),
		Mask:          image.NewRGBA(image.Rect(0, 0, width, height)),
	}

	report := SuiteReport{
		Name:               scene.Name,
		Width:              width,
		Height:             height,
		Pixels:             width * height,
		ReferenceTriangles: len(scene.Triangles),
	}
	regionMap := map[MismatchKind]*ErrorRegion{}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ray := scene.Camera.PixelRay(x, y)
			ref := TraceReference(scene, ray)
			simple := tracer.Trace(ray)
			refColor := Shade(scene, ray, ref)
			simpleColor := Shade(scene, ray, simple)
			colorError := colorDistance(refColor, simpleColor)

			kind := classifyTrace(ref, simple)
			if kind == KindMatch && colorError > 1e-12 {
				kind = KindShadingMismatch
			}

			refID, refT := -1, 0.0
			if ref.OK {
				refID, refT = ref.Hit.ID, ref.Hit.T
			}
			simpleID, simpleT := -1, 0.0
			if simple.OK {
				simpleID, simpleT = simple.Hit.ID, simple.Hit.T
			}
			mismatch := Mismatch{
				X: x, Y: y, Kind: kind,
				RefID: refID, SimpleID: simpleID,
				RefT: refT, SimpleT: simpleT,
				RefCount: len(ref.Hits), SimpleCount: len(simple.Hits),
				SimpleTests: simple.Tests, ColorError: colorError,
			}

			result.Reference.SetRGBA(x, y, ColorToRGBA(refColor))
			result.Simplified.SetRGBA(x, y, ColorToRGBA(simpleColor))
			result.Intersections.SetRGBA(x, y, CountColor(len(ref.Hits)))
			result.TestCounts.SetRGBA(x, y, TestDensityColor(simple.Tests, len(scene.Triangles)))

			heat := colorError
			if kind != KindMatch {
				heat = 1
				result.Mismatches = append(result.Mismatches, mismatch)
				report.MismatchPixels++
				switch kind {
				case KindMissed:
					report.MissedPixels++
				case KindFalseHit:
					report.FalseHitPixels++
				case KindWrongTriangle:
					report.WrongTrianglePixels++
				case KindCountMismatch:
					report.CountMismatchPixels++
				case KindDepthMismatch:
					report.DepthMismatchPixels++
				}
				if region, ok := regionMap[kind]; ok {
					region.include(mismatch)
				} else {
					r := newErrorRegion(kind, mismatch)
					regionMap[kind] = &r
				}
				result.Heatmap.SetRGBA(x, y, HeatColor(1))
				result.Mask.SetRGBA(x, y, mismatchMaskColor(kind))
			} else {
				result.Heatmap.SetRGBA(x, y, HeatColor(heat))
				result.Mask.SetRGBA(x, y, matchMaskColor())
			}

			report.MaxColorError = maxFloat(report.MaxColorError, colorError)
			report.MeanColorError += colorError
			report.MaxIntersectionCount = maxInt(report.MaxIntersectionCount, len(ref.Hits))
			report.MeanIntersectionCount += float64(len(ref.Hits))
			report.MaxSimplifiedTests = maxInt(report.MaxSimplifiedTests, simple.Tests)
			report.MeanSimplifiedTests += float64(simple.Tests)
		}
	}

	pixels := float64(report.Pixels)
	report.MeanColorError /= pixels
	report.MeanIntersectionCount /= pixels
	report.MeanSimplifiedTests /= pixels
	report.Pass = report.MismatchPixels == 0

	kinds := make([]MismatchKind, 0, len(regionMap))
	for kind := range regionMap {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	for _, kind := range kinds {
		report.Regions = append(report.Regions, *regionMap[kind])
	}

	if len(result.Mismatches) > 0 {
		limit := minInt(64, len(result.Mismatches))
		report.FirstMismatches = append([]Mismatch(nil), result.Mismatches[:limit]...)
	}
	result.Report = report
	return result
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
