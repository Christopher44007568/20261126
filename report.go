package hdrlab

import (
	"math"
)

// EdgeEnergy returns the mean luma gradient magnitude inside r.
func EdgeEnergy(img *Img, r Rect) float64 {
	var sum float64
	var n int
	for y := r.MinY + 1; y < r.MaxY-1; y++ {
		for x := r.MinX + 1; x < r.MaxX-1; x++ {
			gx := (img.Luma(x+1, y) - img.Luma(x-1, y)) / 2
			gy := (img.Luma(x, y+1) - img.Luma(x, y-1)) / 2
			sum += math.Sqrt(gx*gx + gy*gy)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// CountEdges counts strong luma transitions along one row.
func CountEdges(img *Img, y int, thresh float64) int {
	count := 0
	for x := 1; x < img.W-1; x++ {
		if math.Abs(img.Luma(x+1, y)-img.Luma(x-1, y)) > thresh {
			count++
		}
	}
	return count
}

type point struct {
	X int `json:"X"`
	Y int `json:"Y"`
}

type boundsReport struct {
	Min point `json:"Min"`
	Max point `json:"Max"`
}

type cellReport struct {
	Bounds         boundsReport `json:"bounds"`
	MeanLuma       float64      `json:"mean_luma"`
	EVRelative     float64      `json:"ev_relative_18pct"`
	ClipFraction   float64      `json:"clipped_fraction"`
	ShadowFraction float64      `json:"shadow_fraction"`
}

type frameReport struct {
	Name        string  `json:"name"`
	EV          float64 `json:"ev"`
	Exposure    float64 `json:"exposure"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	TrueDX      *int    `json:"true_dx,omitempty"`
	TrueDY      *int    `json:"true_dy,omitempty"`
	EstimatedDX int     `json:"estimated_dx"`
	EstimatedDY int     `json:"estimated_dy"`
	Confidence  float64 `json:"confidence"`
}

type shiftReport struct {
	DX         int     `json:"dx"`
	DY         int     `json:"dy"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

type alignmentReport struct {
	Reference int           `json:"reference"`
	Shifts    []shiftReport `json:"shifts"`
}

type meteringReport struct {
	Columns               int          `json:"columns"`
	Rows                  int          `json:"rows"`
	Cells                 []cellReport `json:"cells"`
	CenterEV              float64      `json:"center_ev"`
	HighlightClipFraction float64      `json:"highlight_clip_fraction"`
	DeepShadowFraction    float64      `json:"deep_shadow_fraction"`
}

type metricsReport struct {
	HighlightEdgeEnergyHDR     float64 `json:"highlight_edge_energy_hdr"`
	HighlightEdgeEnergyDirect  float64 `json:"highlight_edge_energy_direct_average"`
	HighlightEdgeEnergyAligned float64 `json:"highlight_edge_energy_aligned_average"`
	HighlightEdgeRatio         float64 `json:"highlight_edge_ratio_hdr_over_average_of_baselines"`
	ObjectEdgeCountHDR         int     `json:"object_edge_count_hdr_row"`
	ObjectEdgeCountReference   int     `json:"object_edge_count_reference_row"`
	ObjectEdgeCountDirect      int     `json:"object_edge_count_direct_row"`
	MotionPixelFraction        float64 `json:"motion_pixel_fraction"`
	MeanDisagreement           float64 `json:"mean_disagreement"`
}

type acceptanceReport struct {
	GroundTruth            bool `json:"ground_truth"`
	AlignmentRecoversShift bool `json:"alignment_recovers_handheld_shift"`
	HighlightEdgePreserved bool `json:"highlight_edge_preserved"`
	NoMultipleMovingEdges  bool `json:"no_multiple_moving_edges"`
	MaxShiftErrorPixels    int  `json:"max_shift_error_pixels"`
}

// MetricsForReport packs the computed quality metrics.
func MetricsForReport(eHDR, eDirect, eAligned, ratio float64, nHDR, nRef, nDirect int, merged *MergeResult) metricsReport {
	return metricsReport{
		HighlightEdgeEnergyHDR:     eHDR,
		HighlightEdgeEnergyDirect:  eDirect,
		HighlightEdgeEnergyAligned: eAligned,
		HighlightEdgeRatio:         ratio,
		ObjectEdgeCountHDR:         nHDR,
		ObjectEdgeCountReference:   nRef,
		ObjectEdgeCountDirect:      nDirect,
		MotionPixelFraction:        merged.MotionFraction,
		MeanDisagreement:           merged.MeanDisagreement,
	}
}

// AcceptanceForReport evaluates the acceptance criteria.
func AcceptanceForReport(maxShiftErr int, edgeRatio float64, nHDR, nRef int) acceptanceReport {
	return acceptanceReport{
		GroundTruth:            true,
		AlignmentRecoversShift: maxShiftErr == 0,
		HighlightEdgePreserved: edgeRatio > 1.15,
		NoMultipleMovingEdges:  nHDR <= nRef+2,
		MaxShiftErrorPixels:    maxShiftErr,
	}
}

type report struct {
	Tool                string            `json:"tool"`
	StandardLibraryOnly bool              `json:"standard_library_only"`
	OutputDirectory     string            `json:"output_directory"`
	Frames              []frameReport     `json:"frames"`
	Alignment           alignmentReport   `json:"alignment"`
	Metering            meteringReport    `json:"metering"`
	Metrics             metricsReport     `json:"metrics"`
	Acceptance          acceptanceReport  `json:"acceptance"`
	Files               map[string]string `json:"files"`
}
