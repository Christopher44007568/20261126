package atlas

import (
	"image/color"
	"math"
	"sort"
)

type ValidateOptions struct {
	// Grid is the number of samples per local U and V axis. The grid is
	// inclusive, so Grid=9 samples both UV boundaries.
	Grid int

	// ColorTolerance is Euclidean RGBA distance accepted as normal mip blur.
	ColorTolerance float64

	// MaxViolations caps the detailed list; counters remain complete.
	MaxViolations int
}

type Violation struct {
	SpriteID string   `json:"sprite_id"`
	MipLevel int      `json:"mip_level"`
	LocalU   float64  `json:"local_u"`
	LocalV   float64  `json:"local_v"`
	AtlasU   float64  `json:"atlas_u"`
	AtlasV   float64  `json:"atlas_v"`
	AtlasPX  float64  `json:"atlas_px"`
	AtlasPY  float64  `json:"atlas_py"`
	Actual   [4]uint8 `json:"actual"`
	Expected [4]uint8 `json:"expected"`
	Distance float64  `json:"distance"`
}

type LevelStat struct {
	Level           int     `json:"level"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Samples         int     `json:"samples"`
	Violations      int     `json:"violations"`
	MaxDistance     float64 `json:"max_distance"`
	RequiredPadding int     `json:"required_base_padding"`
	PackPadding     int     `json:"pack_padding"`
}

type ValidationReport struct {
	OK             bool        `json:"ok"`
	AtlasName      string      `json:"atlas_name,omitempty"`
	MipCount       int         `json:"mip_count"`
	Grid           int         `json:"grid"`
	Tolerance      float64     `json:"tolerance"`
	SampleCount    int         `json:"sample_count"`
	ViolationCount int         `json:"violation_count"`
	MaxDistance    float64     `json:"max_distance"`
	Levels         []LevelStat `json:"levels"`
	Violations     []Violation `json:"violations"`
}

// ValidateSampling emulates runtime bilinear sampling at every requested mip
// and compares atlas results against each sprite's independent mip chain.
func ValidateSampling(b *BuiltAtlas, name string, opts ValidateOptions) ValidationReport {
	if opts.Grid < 2 {
		opts.Grid = 9
	}
	if opts.ColorTolerance <= 0 {
		opts.ColorTolerance = 18
	}
	if opts.MaxViolations <= 0 {
		opts.MaxViolations = 250
	}

	rt := NewRuntimeAtlas(b)
	report := ValidationReport{
		OK: true, AtlasName: name, MipCount: len(b.Mips),
		Grid: opts.Grid, Tolerance: opts.ColorTolerance,
		Levels: make([]LevelStat, 0, len(b.Mips)),
	}

	coords := inclusiveGrid(opts.Grid)
	for level, mip := range b.Mips {
		stat := LevelStat{
			Level: level, Width: mip.Bounds().Dx(), Height: mip.Bounds().Dy(),
			RequiredPadding: b.Layout.Policy.BasePixelsForMip(level),
			PackPadding:     b.Layout.PackPadding,
		}

		for _, p := range b.Layout.Placements {
			expectedChain := b.FrameMips[p.ID]
			if expectedChain == nil {
				expectedChain = b.SourceMips[p.ID]
			}
			expectedImg := expectedChain[minInt(level, len(expectedChain)-1)]
			lodScale := float64(int(1) << uint(level))
			spanW := float64(p.Frame.W) / lodScale
			spanH := float64(p.Frame.H) / lodScale

			for _, lu := range coords {
				for _, lv := range coords {
					actual, _, err := rt.SampleMip(p.ID, lu, lv, level)
					if err != nil {
						continue
					}
					eu, ev := lu, lv
					if p.Rotated {
						eu, ev = 1.0-lv, lu
					}
					expected := SampleBilinearSpan(expectedImg, eu, ev, spanW, spanH)
					dist := colorDistance(actual, expected)
					stat.Samples++
					report.SampleCount++
					if dist > stat.MaxDistance {
						stat.MaxDistance = dist
					}
					if dist > report.MaxDistance {
						report.MaxDistance = dist
					}
					if dist <= opts.ColorTolerance {
						continue
					}

					stat.Violations++
					report.ViolationCount++
					report.OK = false
					au, av := SpriteToAtlasUV(p, b.Layout.Width, b.Layout.Height, lu, lv)
					v := Violation{
						SpriteID: p.ID, MipLevel: level,
						LocalU: lu, LocalV: lv, AtlasU: au, AtlasV: av,
						AtlasPX: au * float64(b.Layout.Width),
						AtlasPY: av * float64(b.Layout.Height),
						Actual:  rgbaArray(actual), Expected: rgbaArray(expected),
						Distance: math.Round(dist*100) / 100,
					}
					if len(report.Violations) < opts.MaxViolations {
						report.Violations = append(report.Violations, v)
					}
				}
			}
		}
		report.Levels = append(report.Levels, stat)
	}

	sort.SliceStable(report.Violations, func(i, j int) bool {
		if report.Violations[i].Distance != report.Violations[j].Distance {
			return report.Violations[i].Distance > report.Violations[j].Distance
		}
		if report.Violations[i].MipLevel != report.Violations[j].MipLevel {
			return report.Violations[i].MipLevel < report.Violations[j].MipLevel
		}
		return report.Violations[i].SpriteID < report.Violations[j].SpriteID
	})
	return report
}

func inclusiveGrid(n int) []float64 {
	if n <= 1 {
		return []float64{0, 1}
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = float64(i) / float64(n-1)
	}
	return out
}

func rgbaArray(c color.RGBA) [4]uint8 {
	return [4]uint8{c.R, c.G, c.B, c.A}
}
