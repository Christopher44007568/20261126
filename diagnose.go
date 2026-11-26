package main

import (
	"image"
	"math"
)

type MismatchKind string

const (
	KindMatch           MismatchKind = "match"
	KindMissed          MismatchKind = "missed_triangle"
	KindFalseHit        MismatchKind = "false_hit"
	KindWrongTriangle   MismatchKind = "wrong_triangle"
	KindCountMismatch   MismatchKind = "intersection_count_mismatch"
	KindDepthMismatch   MismatchKind = "depth_mismatch"
	KindShadingMismatch MismatchKind = "shading_mismatch"
)

type Mismatch struct {
	X           int          `json:"x"`
	Y           int          `json:"y"`
	Kind        MismatchKind `json:"kind"`
	RefID       int          `json:"ref_id"`
	SimpleID    int          `json:"simple_id"`
	RefT        float64      `json:"ref_t"`
	SimpleT     float64      `json:"simple_t"`
	RefCount    int          `json:"ref_count"`
	SimpleCount int          `json:"simple_count"`
	SimpleTests int          `json:"simple_tests"`
	ColorError  float64      `json:"color_error"`
}

type ErrorRegion struct {
	Kind  MismatchKind `json:"kind"`
	Count int          `json:"count"`
	MinX  int          `json:"min_x"`
	MinY  int          `json:"min_y"`
	MaxX  int          `json:"max_x"`
	MaxY  int          `json:"max_y"`
}

type SuiteReport struct {
	Name                  string        `json:"name"`
	Width                 int           `json:"width"`
	Height                int           `json:"height"`
	Pixels                int           `json:"pixels"`
	Pass                  bool          `json:"pass"`
	MismatchPixels        int           `json:"mismatch_pixels"`
	MissedPixels          int           `json:"missed_pixels"`
	FalseHitPixels        int           `json:"false_hit_pixels"`
	WrongTrianglePixels   int           `json:"wrong_triangle_pixels"`
	CountMismatchPixels   int           `json:"count_mismatch_pixels"`
	DepthMismatchPixels   int           `json:"depth_mismatch_pixels"`
	MaxColorError         float64       `json:"max_color_error"`
	MeanColorError        float64       `json:"mean_color_error"`
	MaxIntersectionCount  int           `json:"max_intersection_count"`
	MeanIntersectionCount float64       `json:"mean_intersection_count"`
	MaxSimplifiedTests    int           `json:"max_simplified_tests"`
	MeanSimplifiedTests   float64       `json:"mean_simplified_tests"`
	ReferenceTriangles    int           `json:"reference_triangles"`
	Regions               []ErrorRegion `json:"regions"`
	FirstMismatches       []Mismatch    `json:"first_mismatches,omitempty"`
}

type SuiteResult struct {
	Report        SuiteReport
	Reference     *image.RGBA
	Simplified    *image.RGBA
	Intersections *image.RGBA
	TestCounts    *image.RGBA
	Heatmap       *image.RGBA
	Mask          *image.RGBA
	Mismatches    []Mismatch
}

func colorDistance(a, b Vec3) float64 {
	d := a.Sub(b)
	return math.Sqrt(d.X*d.X+d.Y*d.Y+d.Z*d.Z) / math.Sqrt(3)
}

func classifyTrace(ref, simple TraceInfo) MismatchKind {
	switch {
	case ref.OK && !simple.OK:
		return KindMissed
	case !ref.OK && simple.OK:
		return KindFalseHit
	case ref.OK && simple.OK && ref.Hit.ID != simple.Hit.ID:
		return KindWrongTriangle
	case len(ref.Hits) != len(simple.Hits):
		return KindCountMismatch
	}

	for i := range ref.Hits {
		rh := ref.Hits[i]
		sh := simple.Hits[i]
		if rh.ID != sh.ID {
			return KindCountMismatch
		}
		if rh.T != sh.T {
			return KindDepthMismatch
		}
	}

	if ref.OK && simple.OK && ref.Hit.T != simple.Hit.T {
		return KindDepthMismatch
	}
	return KindMatch
}

func newErrorRegion(kind MismatchKind, m Mismatch) ErrorRegion {
	return ErrorRegion{Kind: kind, Count: 1, MinX: m.X, MinY: m.Y, MaxX: m.X, MaxY: m.Y}
}

func (r *ErrorRegion) include(m Mismatch) {
	r.Count++
	r.MinX = min(r.MinX, m.X)
	r.MinY = min(r.MinY, m.Y)
	r.MaxX = max(r.MaxX, m.X)
	r.MaxY = max(r.MaxY, m.Y)
}
