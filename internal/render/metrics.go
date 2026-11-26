package render

import (
	"math"
)

type PipelineMetrics struct {
	Bits                  int     `json:"bits"`
	UniqueHardColors      int     `json:"unique_hard_colors"`
	HardQuantMAE          float64 `json:"hard_quant_mae_code"`
	HardQuantRMSE         float64 `json:"hard_quant_rmse_code"`
	HardQuantMaxAbs       float64 `json:"hard_quant_max_abs_code"`
	HardClipLoss          float64 `json:"hard_clip_loss_code"`
	ClippedPixelFraction  float64 `json:"clipped_pixel_fraction"`
	DitherBlockRMSE       float64 `json:"dither_block_rmse_code"`
	DitherPixelRMSE       float64 `json:"dither_pixel_rmse_code"`
	HueMAE                float64 `json:"hue_mae_deg"`
	HueMax                float64 `json:"hue_max_deg"`
	HueApplicable         bool    `json:"hue_applicable"`
	HardBandRuns          int     `json:"hard_band_runs"`
	HardBandMeanWidth     float64 `json:"hard_band_mean_width_px"`
	FirstSaturatedRColumn int     `json:"first_saturated_r_column"`
	FirstSaturatedGColumn int     `json:"first_saturated_g_column"`
	FirstSaturatedBColumn int     `json:"first_saturated_b_column"`
	SubcodeStep8Bit       float64 `json:"subcode_step_8bit_code"`
}

type SceneMetrics struct {
	SceneID     string            `json:"scene_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ClipU       [3]float64        `json:"clip_u_rgb"`
	HueWrapU    float64           `json:"hue_wrap_u"`
	Pipelines   []PipelineMetrics `json:"pipelines"`
}

type HueSwitch struct {
	FromBits int     `json:"from_bits"`
	ToBits   int     `json:"to_bits"`
	Samples  int     `json:"samples"`
	MeanDeg  float64 `json:"mean_deg"`
	MaxDeg   float64 `json:"max_deg"`
}

type MetricsReport struct {
	DisplayBits     int            `json:"display_bits"`
	Dither          string         `json:"dither"`
	ColorPolicy     string         `json:"color_policy"`
	HardLimitNote   string         `json:"hard_limit_note"`
	HueSwitches     []HueSwitch    `json:"hue_switches"`
	Scenes          []SceneMetrics `json:"scenes"`
	LadderDensities map[string]int `json:"ladder_distinct_densities"`
	OutputFiles     []string       `json:"output_files"`
}

const metricW = 576
const metricH = 120

func buildMetrics() MetricsReport {
	report := MetricsReport{
		DisplayBits: 8,
		Dither:      "4x4 ordered Bayer after retaining the fractional 8-bit coordinate",
		ColorPolicy: "one unwrapped source per scene; identical geometry and hue path at every depth",
		HardLimitNote: "an 8-bit display has at most 256 codes per channel; " +
			"10/12-bit depth becomes visible through more accurate rounding and denser spatial averaging",
		LadderDensities: map[string]int{},
	}
	for i := range Scenes {
		scene := Scenes[i]
		sm := SceneMetrics{
			SceneID:     scene.ID,
			Name:        scene.Title,
			Description: scene.Note,
			ClipU:       scene.ClipU,
			HueWrapU:    scene.HueWrapU,
		}
		for _, p := range Pipelines {
			sm.Pipelines = append(sm.Pipelines, measure(scene, p))
		}
		report.Scenes = append(report.Scenes, sm)
	}
	report.HueSwitches = []HueSwitch{
		compareHues(Scenes[1], Pipelines[0], Pipelines[1]),
		compareHues(Scenes[1], Pipelines[1], Pipelines[2]),
		compareHues(Scenes[1], Pipelines[0], Pipelines[2]),
	}
	report.LadderDensities = ladderDensities()
	return report
}

func measure(scene Scene, p Pipeline) PipelineMetrics {
	m := PipelineMetrics{
		Bits:            p.Bits,
		HardQuantMaxAbs: 0,
		SubcodeStep8Bit: 255 / float64((int(1)<<p.Bits)-1),
		HueApplicable:   scene.HueScene,
	}
	unique := map[uint32]struct{}{}
	colors := make([]uint32, metricW)
	errByBlock := map[int][3]float64{}
	blockCount := map[int]int{}
	var absSum, sqSum float64
	var samples int
	var clipLoss float64
	var clipped int
	var hueSum, hueMax float64
	var hueSamples int

	for y := 0; y < metricH; y++ {
		for x := 0; x < metricW; x++ {
			src := sourceAt(scene, x, y, metricW, metricH)
			disp := DisplayRGB(src, p, x, y)
			target := RGB{Clamp01(src.R) * 255, Clamp01(src.G) * 255, Clamp01(src.B) * 255}
			errs := [3]float64{disp.R - target.R, disp.G - target.G, disp.B - target.B}

			for _, e := range errs {
				ae := math.Abs(e)
				absSum += ae
				sqSum += e * e
				if ae > m.HardQuantMaxAbs {
					m.HardQuantMaxAbs = ae
				}
				samples++
			}
			key := (y/4)*(metricW/4+1) + x/4
			acc := errByBlock[key]
			acc[0] += errs[0]
			acc[1] += errs[1]
			acc[2] += errs[2]
			errByBlock[key] = acc
			blockCount[key]++

			hard := uint32(HardCode(src.R, p.Bits))<<16 |
				uint32(HardCode(src.G, p.Bits))<<8 |
				uint32(HardCode(src.B, p.Bits))
			unique[hard] = struct{}{}
			colors[x] = hard

			if isClipped(src) {
				clipped++
			}
			clipLoss += (math.Max(0, src.R-1) + math.Max(0, src.G-1) + math.Max(0, src.B-1)) * 255 / 3

			if scene.HueScene {
				sh, ss := RGBHue(src)
				dh, ds := RGBHue(RGB{disp.R / 255, disp.G / 255, disp.B / 255})
				if ss > 0.12 && ds > 0.08 {
					d := HueDistance(sh, dh)
					hueSum += d
					hueSamples++
					if d > hueMax {
						hueMax = d
					}
				}
			}
		}
	}

	m.UniqueHardColors = len(unique)
	m.HardQuantMAE = absSum / float64(max(1, samples))
	m.HardQuantRMSE = math.Sqrt(sqSum / float64(max(1, samples)))
	m.HardClipLoss = clipLoss / float64(metricW*metricH)
	m.ClippedPixelFraction = float64(clipped) / float64(metricW*metricH)
	m.DitherPixelRMSE = m.HardQuantRMSE
	m.DitherBlockRMSE = blockRMSE(errByBlock, blockCount)
	if hueSamples > 0 {
		m.HueMAE = hueSum / float64(hueSamples)
		m.HueMax = hueMax
	}
	m.HardBandRuns, m.HardBandMeanWidth = bandRuns(colors)
	m.FirstSaturatedRColumn = firstSaturated(scene, p, 0, metricW, metricH)
	m.FirstSaturatedGColumn = firstSaturated(scene, p, 1, metricW, metricH)
	m.FirstSaturatedBColumn = firstSaturated(scene, p, 2, metricW, metricH)
	return m
}

func blockRMSE(errs map[int][3]float64, counts map[int]int) float64 {
	var sum float64
	for key, acc := range errs {
		n := float64(counts[key])
		for i := 0; i < 3; i++ {
			mean := acc[i] / n
			sum += mean * mean
		}
	}
	if len(errs) == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(len(errs)*3))
}

func bandRuns(colors []uint32) (int, float64) {
	if len(colors) == 0 {
		return 0, 0
	}
	runs := 1
	for i := 1; i < len(colors); i++ {
		if colors[i] != colors[i-1] {
			runs++
		}
	}
	return runs, float64(len(colors)) / float64(runs)
}

func compareHues(scene Scene, a, b Pipeline) HueSwitch {
	hs := HueSwitch{FromBits: a.Bits, ToBits: b.Bits}
	var sum, maxd float64
	for y := 0; y < metricH; y++ {
		for x := 0; x < metricW; x++ {
			src := sourceAt(scene, x, y, metricW, metricH)
			ha, sa := RGBHue(RGB{
				R: float64(DisplayCode(src.R, a, x, y)) / 255,
				G: float64(DisplayCode(src.G, a, x, y)) / 255,
				B: float64(DisplayCode(src.B, a, x, y)) / 255,
			})
			hb, sb := RGBHue(RGB{
				R: float64(DisplayCode(src.R, b, x, y)) / 255,
				G: float64(DisplayCode(src.G, b, x, y)) / 255,
				B: float64(DisplayCode(src.B, b, x, y)) / 255,
			})
			if sa < 0.02 || sb < 0.02 {
				continue
			}
			d := HueDistance(ha, hb)
			sum += d
			hs.Samples++
			if d > maxd {
				maxd = d
			}
		}
	}
	if hs.Samples > 0 {
		hs.MeanDeg = sum / float64(hs.Samples)
	}
	hs.MaxDeg = maxd
	return hs
}

func ladderDensities() map[string]int {
	out := map[string]int{}
	for _, p := range Pipelines {
		seen := map[int]struct{}{}
		for i := 0; i < 16; i++ {
			target := 100.0 + float64(i)/16
			on := 0
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					if DisplayCode(target/255, p, x, y) > 100 {
						on++
					}
				}
			}
			seen[on] = struct{}{}
		}
		out[p.Short] = len(seen)
	}
	return out
}
