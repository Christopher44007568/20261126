package mipmap

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LevelMetric struct {
	Method               string       `json:"method"`
	Level                int          `json:"level"`
	Width                int          `json:"width"`
	Height               int          `json:"height"`
	Summary              ColorSummary `json:"summary"`
	ColorErrorWorkingRMS float64      `json:"average_color_error_working_rms"`
	ColorErrorOutput8RMS float64      `json:"average_color_error_output8_rms"`
	PixelMAEVsWrap       *float64     `json:"pixel_mae_vs_wrap,omitempty"`
	SeamBandMAEVsWrap    *float64     `json:"seam_band_mae_vs_wrap,omitempty"`
	BorderLuminance      float64      `json:"border_luminance_working"`
	EnergyGainDrift      *float64     `json:"energy_gain_drift,omitempty"`
}

type ProbeMetric struct {
	Method          string  `json:"method"`
	Level           int     `json:"level"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Luminance       float64 `json:"mean_luminance"`
	BorderLuminance float64 `json:"border_luminance"`
	BrightnessRatio float64 `json:"brightness_ratio_vs_input"`
}

type Report struct {
	Width                    int            `json:"width"`
	Height                   int            `json:"height"`
	ColorSpace               string         `json:"color_space"`
	NormalStrength           float64        `json:"normal_strength"`
	MaxAllowedMeanError      float64        `json:"max_allowed_mean_error_working"`
	Pass                     bool           `json:"pass"`
	MaxWrapColorErrorWorking float64        `json:"max_wrap_color_error_working"`
	MaxProbeEdgeDeviation    float64        `json:"max_probe_edge_deviation"`
	ColorMetrics             []LevelMetric  `json:"color_metrics"`
	ConstantProbe            []ProbeMetric  `json:"constant_probe"`
	EnergyReports            []EnergyReport `json:"energy_reports"`
}

func BuildLevelMetric(method string, level int, img, base, wrapLevel *RGBImage, energy *EnergyReport) LevelMetric {
	baseOutput := SummarizeColor(base).MeanOutput8
	metric := LevelMetric{
		Method:               method,
		Level:                level,
		Width:                img.Width,
		Height:               img.Height,
		Summary:              SummarizeColor(img),
		ColorErrorWorkingRMS: AverageColorError(img, base.Mean()),
		ColorErrorOutput8RMS: AverageOutputColorError(img, baseOutput),
		BorderLuminance:      BorderMeanLuminance(img, 1),
	}
	if wrapLevel != nil && img != wrapLevel {
		pixelMAE := MeanAbsoluteError(img, wrapLevel)
		seamMAE := BorderBandMAE(img, wrapLevel, 1)
		metric.PixelMAEVsWrap = &pixelMAE
		metric.SeamBandMAEVsWrap = &seamMAE
	}
	if energy != nil {
		drift := energy.MaxGainDrift
		metric.EnergyGainDrift = &drift
	}
	return metric
}

func WriteJSON(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func WriteColorCSV(path string, metrics []LevelMetric) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	header := []string{
		"color_space", "method", "level", "width", "height",
		"mean_r_working", "mean_g_working", "mean_b_working", "luminance_working",
		"mean_r_output8", "mean_g_output8", "mean_b_output8",
		"mean_luminance_linear_light",
		"avg_color_error_working_rms", "avg_color_error_output8_rms",
		"pixel_mae_vs_wrap", "seam_band_mae_vs_wrap", "border_luminance",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, metric := range metrics {
		row := []string{
			metric.Summary.ColorSpace,
			metric.Method,
			fmt.Sprint(metric.Level),
			fmt.Sprint(metric.Width),
			fmt.Sprint(metric.Height),
			fmt.Sprintf("%.12f", metric.Summary.MeanWorkingRGB[0]),
			fmt.Sprintf("%.12f", metric.Summary.MeanWorkingRGB[1]),
			fmt.Sprintf("%.12f", metric.Summary.MeanWorkingRGB[2]),
			fmt.Sprintf("%.12f", metric.Summary.LuminanceWorking),
			fmt.Sprintf("%.6f", metric.Summary.MeanOutput8[0]),
			fmt.Sprintf("%.6f", metric.Summary.MeanOutput8[1]),
			fmt.Sprintf("%.6f", metric.Summary.MeanOutput8[2]),
			fmt.Sprintf("%.12f", metric.Summary.LuminanceLinear),
			fmt.Sprintf("%.12e", metric.ColorErrorWorkingRMS),
			fmt.Sprintf("%.6f", metric.ColorErrorOutput8RMS),
			floatPtrString(metric.PixelMAEVsWrap),
			floatPtrString(metric.SeamBandMAEVsWrap),
			fmt.Sprintf("%.12f", metric.BorderLuminance),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func floatPtrString(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.12e", *value)
}

func WriteMarkdownReport(path string, report Report) error {
	var builder strings.Builder
	builder.WriteString("# Wrap-safe mipmap report\n\n")
	builder.WriteString(fmt.Sprintf("Base size: `%dx%d`; color space: `%s`; normal strength: `%.2f`.\n\n", report.Width, report.Height, report.ColorSpace, report.NormalStrength))
	status := "PASS"
	if !report.Pass {
		status = "FAIL"
	}
	builder.WriteString(fmt.Sprintf("Validation: **%s** (max wrap mean error `%.3e`, max constant-edge deviation `%.3e`).\n\n", status, report.MaxWrapColorErrorWorking, report.MaxProbeEdgeDeviation))
	builder.WriteString("Average color error compares each mip's mean working RGB with the base level; output-8 error is measured after PNG quantization. `seam_band_mae_vs_wrap` isolates the outer tile boundary.\n\n")
	builder.WriteString("| Method | Lvl | Size | Working luminance | Color error | Output-8 error | Seam MAE |\n")
	builder.WriteString("|---|---:|---:|---:|---:|---:|---:|\n")
	for _, metric := range report.ColorMetrics {
		seam := ""
		if metric.SeamBandMAEVsWrap != nil {
			seam = fmt.Sprintf("%.3e", *metric.SeamBandMAEVsWrap)
		}
		builder.WriteString(fmt.Sprintf("| `%s` | %d | %dx%d | %.6f | %.3e | %.3f | %s |\n",
			metric.Method, metric.Level, metric.Width, metric.Height,
			metric.Summary.LuminanceWorking, metric.ColorErrorWorkingRMS,
			metric.ColorErrorOutput8RMS, seam))
	}

	builder.WriteString("\n## Constant-color edge probe\n\n")
	builder.WriteString("A wrapped normalized chain stays at ratio `1` on every edge. The zero-padded inside-only chain darkens progressively.\n\n")
	builder.WriteString("| Method | Lvl | Size | Border luminance | Ratio vs input |\n")
	builder.WriteString("|---|---:|---:|---:|---:|\n")
	for _, probe := range report.ConstantProbe {
		builder.WriteString(fmt.Sprintf("| `%s` | %d | %dx%d | %.6f | %.6f |\n",
			probe.Method, probe.Level, probe.Width, probe.Height,
			probe.BorderLuminance, probe.BrightnessRatio))
	}

	return os.WriteFile(path, []byte(builder.String()), 0o644)
}
