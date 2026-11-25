package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"aelab/internal/pipeline"
)

func WriteFrameCSV(path string, result *pipeline.Result) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	header := []string{
		"profile_id", "profile_label", "frame", "time_s", "pan_x",
		"meter_geometric_lum", "meter_log_lum", "meter_p95_lum",
		"target_ev", "actual_ev", "gain_linear", "velocity_ev_s", "ev_step",
		"tracking_error_ev", "reference_final_ev", "overshoot_ev_positive_dark",
		"completion_pct", "output_mean_luma", "output_luma_delta",
		"window_mean", "window_std", "window_edge_energy", "window_clip_high",
		"shadow_mean", "shadow_std", "shadow_edge_energy", "shadow_clip_low",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, profile := range result.Profiles {
		for _, r := range profile.Records {
			row := []string{
				r.ProfileID, r.ProfileLabel, itoa(r.Frame), f3(r.TimeS), itoa(r.PanX),
				f6(r.MeterLum), f6(r.LogLum), f6(r.Percentile95),
				f6(r.TargetEV), f6(r.ActualEV), f6(r.Gain), f6(r.VelocityEVPerS), f6(r.EVStep),
				f6(r.TrackingErrorEV), f6(r.ReferenceEV), f6(r.OvershootEV),
				f6(r.CompletionPct), f6(r.MeanLuma), f6(r.LumaDelta),
				f6(r.WindowMean), f6(r.WindowStd), f6(r.WindowEdge), f6(r.WindowClipHigh),
				f6(r.ShadowMean), f6(r.ShadowStd), f6(r.ShadowEdge), f6(r.ShadowClipLow),
			}
			if err := w.Write(row); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}

func WriteOvershootCSV(path string, result *pipeline.Result) error {
	f, err := createCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	header := []string{"frame", "time_s", "reference_final_ev", "target_ev"}
	for _, p := range result.Profiles {
		header = append(header, p.Profile.ID+"_actual_ev", p.Profile.ID+"_overshoot_ev", p.Profile.ID+"_completion_pct")
	}
	if err := w.Write(header); err != nil {
		return err
	}
	frameCount := len(result.Profiles[0].Records)
	for i := 0; i < frameCount; i++ {
		first := result.Profiles[0].Records[i]
		row := []string{itoa(first.Frame), f3(first.TimeS), f6(first.ReferenceEV), f6(first.TargetEV)}
		for _, p := range result.Profiles {
			r := p.Records[i]
			row = append(row, f6(r.ActualEV), f6(r.OvershootEV), f6(r.CompletionPct))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func WriteMetricsCSV(path string, result *pipeline.Result) error {
	f, err := createCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	header := []string{
		"profile_id", "profile_label", "initial_ev", "final_target_ev", "final_actual_ev",
		"settle_time_s", "settle_after_turn_start_s", "max_overshoot_ev", "max_abs_deviation_ev",
		"zero_crossings", "max_ev_step", "max_luma_jump", "mean_abs_luma_jump", "luma_reversals",
		"window_mean", "window_edge", "window_clip_high", "shadow_mean", "shadow_std", "shadow_edge",
		"passes_no_flash", "passes_smooth_converge",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, p := range result.Profiles {
		m := p.Metrics
		settleAfter := m.SettleTimeS - pipeline.TransitionStart
		if m.SettleTimeS < 0 {
			settleAfter = -1
		}
		row := []string{
			p.Profile.ID, p.Profile.Label, f6(m.InitialEV), f6(m.FinalTargetEV), f6(m.FinalActualEV),
			f3(m.SettleTimeS), f3(settleAfter), f6(m.MaxOvershootEV), f6(m.MaxAbsDeviationEV),
			itoa(m.ZeroCrossings), f6(m.MaxEVStep), f6(m.MaxLumaJump), f6(m.MeanAbsLumaJump), itoa(m.LumaReversals),
			f6(m.WindowMean), f6(m.WindowEdge), f6(m.WindowClipHigh), f6(m.ShadowMean), f6(m.ShadowStd), f6(m.ShadowEdge),
			boolString(m.PassesNoFlash), boolString(m.PassesSmoothConverge),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func createCSV(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func itoa(v int) string { return fmt.Sprintf("%d", v) }
func f3(v float64) string {
	if v < 0 {
		return fmt.Sprintf("%.3f", v)
	}
	return fmt.Sprintf("%.3f", v)
}
func f6(v float64) string { return fmt.Sprintf("%.6f", v) }
func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
