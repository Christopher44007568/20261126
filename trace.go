package main

import (
	"errors"
	"math"
	"sort"
)

type interval struct {
	layer Layer
	index int
	t0    float64
	t1    float64
}

func DefaultScene() []Layer {
	return []Layer{
		{
			Name:          "glass",
			Min:           Vec3{X: -1.50, Y: -2.0, Z: -1.5},
			Max:           Vec3{X: -1.05, Y: 2.0, Z: 1.5},
			Transmittance: RGB{R: 0.92, G: 0.95, B: 0.96},
			Tint:          RGB{R: 0.72, G: 0.90, B: 1.00},
		},
		{
			Name:          "water",
			Min:           Vec3{X: -0.45, Y: -2.0, Z: -1.5},
			Max:           Vec3{X: 0.30, Y: 2.0, Z: 1.5},
			Transmittance: RGB{R: 0.78, G: 0.62, B: 0.88},
			Tint:          RGB{R: 0.25, G: 0.62, B: 0.95},
		},
		{
			Name:          "mist",
			Min:           Vec3{X: 0.95, Y: -2.0, Z: -1.5},
			Max:           Vec3{X: 1.70, Y: 2.0, Z: 1.5},
			Transmittance: RGB{R: 0.86, G: 0.88, B: 0.90},
			Tint:          RGB{R: 0.82, G: 0.88, B: 0.92},
		},
	}
}

func TraceTransparent(ray Ray, layers []Layer, initial RGB, tMin, tMax float64, maxHits int) (TraceResult, error) {
	if math.Abs(ray.Direction.X) < 1e-12 {
		return TraceResult{}, errors.New("ray direction must have a non-zero X component")
	}
	if tMax <= tMin {
		return TraceResult{}, errors.New("tMax must be greater than tMin")
	}

	intervals, err := slabIntervals(ray, layers, tMin, tMax)
	if err != nil {
		return TraceResult{}, err
	}
	sort.SliceStable(intervals, func(i, j int) bool {
		if intervals[i].t0 == intervals[j].t0 {
			return intervals[i].index < intervals[j].index
		}
		return intervals[i].t0 < intervals[j].t0
	})

	result := TraceResult{
		Hits:   make([]Hit, 0, min(maxHits, len(intervals))),
		Energy: initial,
	}
	if maxHits < 0 {
		return TraceResult{}, errors.New("maxHits cannot be negative")
	}

	for index, item := range intervals {
		if index >= maxHits {
			segmentStart := tMin
			if len(result.Hits) > 0 {
				segmentStart = result.Hits[len(result.Hits)-1].T1
			}
			segment := &Segment{
				Start:  ray.At(segmentStart),
				End:    ray.At(tMax),
				T0:     segmentStart,
				T1:     tMax,
				Reason: "hit budget exhausted",
			}
			result.Complete = false
			result.Unprocessed = segment
			result.RemainingLayers = remainingLayerNames(intervals[index:])
			return result, nil
		}

		before := result.Energy
		after := RGB{
			R: before.R * item.layer.Transmittance.R,
			G: before.G * item.layer.Transmittance.G,
			B: before.B * item.layer.Transmittance.B,
		}
		result.Hits = append(result.Hits, Hit{
			LayerIndex:    item.index,
			LayerName:     item.layer.Name,
			Entry:         ray.At(item.t0),
			Exit:          ray.At(item.t1),
			T0:            item.t0,
			T1:            item.t1,
			Transmittance: item.layer.Transmittance,
			EnergyBefore:  before,
			EnergyAfter:   after,
			OutOfRange:    energyOutOfRange(before) || energyOutOfRange(after),
		})
		result.Energy = after
	}

	result.Complete = true
	return result, nil
}

func slabIntervals(ray Ray, layers []Layer, tMin, tMax float64) ([]interval, error) {
	intervals := make([]interval, 0, len(layers))
	for index, layer := range layers {
		t0, t1, hit, err := slabInterval(ray, layer, tMin, tMax)
		if err != nil {
			return nil, err
		}
		if hit {
			intervals = append(intervals, interval{
				layer: layer,
				index: index,
				t0:    t0,
				t1:    t1,
			})
		}
	}
	return intervals, nil
}

func slabInterval(ray Ray, layer Layer, tMin, tMax float64) (float64, float64, bool, error) {
	lo := tMin
	hi := tMax
	axes := []struct {
		origin    float64
		direction float64
		min       float64
		max       float64
	}{
		{ray.Origin.X, ray.Direction.X, layer.Min.X, layer.Max.X},
		{ray.Origin.Y, ray.Direction.Y, layer.Min.Y, layer.Max.Y},
		{ray.Origin.Z, ray.Direction.Z, layer.Min.Z, layer.Max.Z},
	}

	for _, axis := range axes {
		if math.Abs(axis.direction) < 1e-12 {
			if axis.origin < axis.min || axis.origin > axis.max {
				return 0, 0, false, nil
			}
			continue
		}

		near := (axis.min - axis.origin) / axis.direction
		far := (axis.max - axis.origin) / axis.direction
		if near > far {
			near, far = far, near
		}
		if near > lo {
			lo = near
		}
		if far < hi {
			hi = far
		}
		if hi < lo {
			return 0, 0, false, nil
		}
	}

	if lo >= tMax || hi <= tMin {
		return 0, 0, false, nil
	}
	if hi <= lo {
		return 0, 0, false, errors.New("degenerate slab intersection")
	}
	return lo, hi, true, nil
}

func remainingLayerNames(items []interval) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.layer.Name)
	}
	return names
}

func energyOutOfRange(energy RGB) bool {
	return math.IsNaN(energy.R) || math.IsNaN(energy.G) || math.IsNaN(energy.B) ||
		math.IsInf(energy.R, 0) || math.IsInf(energy.G, 0) || math.IsInf(energy.B, 0) ||
		energy.R < 0 || energy.R > 1 ||
		energy.G < 0 || energy.G > 1 ||
		energy.B < 0 || energy.B > 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
