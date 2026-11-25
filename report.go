package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writeReports(outputDir string, report RunReport) error {
	csvPath := filepath.Join(outputDir, "metrics.csv")
	file, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := strings.Join([]string{
		"frame", "light", "true_cct", "input_neutral_rmse",
		"gray_r_gain", "gray_b_gain", "gray_cct", "gray_rmse",
		"white_r_gain", "white_b_gain", "white_cct", "white_rmse",
		"smooth_r_gain", "smooth_b_gain", "smooth_cct", "smooth_rmse",
	}, ",")
	if _, err := fmt.Fprintln(file, header); err != nil {
		return err
	}
	for _, frame := range report.Frames {
		row := []string{
			strconv.Itoa(frame.Index), frame.LightName, formatFloat(frame.TrueCCT), formatFloat(frame.InputError),
			formatFloat(frame.GrayGains.R), formatFloat(frame.GrayGains.B), formatFloat(frame.GrayCCT), formatFloat(frame.GrayError),
			formatFloat(frame.WhiteGains.R), formatFloat(frame.WhiteGains.B), formatFloat(frame.WhiteCCT), formatFloat(frame.WhiteError),
			formatFloat(frame.SmoothGains.R), formatFloat(frame.SmoothGains.B), formatFloat(frame.SmoothCCT), formatFloat(frame.SmoothError),
		}
		if _, err := fmt.Fprintln(file, strings.Join(row, ",")); err != nil {
			return err
		}
	}

	summaryPath := filepath.Join(outputDir, "summary.txt")
	summary, err := os.Create(summaryPath)
	if err != nil {
		return err
	}
	defer summary.Close()

	fmt.Fprintln(summary, "Synthetic AWB comparison")
	fmt.Fprintln(summary, "Frames: 0-9 sunset, 10-13 transition, 14-23 fluorescent, 24-27 transition, 28-39 shadow")
	fmt.Fprintln(summary, "Neutral RMSE is measured on four fixed gray patches in linear RGB.")
	fmt.Fprintln(summary)
	writeStrategySummary(summary, "gray_world", report, func(frame FrameReport) (Gains, float64) {
		return frame.GrayGains, frame.GrayError
	})
	writeStrategySummary(summary, "white_point", report, func(frame FrameReport) (Gains, float64) {
		return frame.WhiteGains, frame.WhiteError
	})
	writeStrategySummary(summary, "temporal_smooth", report, func(frame FrameReport) (Gains, float64) {
		return frame.SmoothGains, frame.SmoothError
	})
	return nil
}

func writeStrategySummary(file *os.File, name string, report RunReport, values func(FrameReport) (Gains, float64)) {
	ranges := [][2]int{{0, 9}, {14, 14}, {15, 23}, {28, 28}, {29, 39}}
	fmt.Fprintf(file, "%s\n", name)
	for _, segment := range ranges {
		maxStep := 0.0
		previous, _ := values(report.Frames[segment[0]])
		for i := segment[0] + 1; i <= segment[1]; i++ {
			current, _ := values(report.Frames[i])
			maxStep = maxFloat(maxStep, absFloat(current.R-previous.R), absFloat(current.B-previous.B))
			previous = current
		}
		lastGains, lastError := values(report.Frames[segment[1]])
		fmt.Fprintf(file, "  frames %02d-%02d: last_rmse=%.5f max_gain_step=%.5f last_gains=(%.5f,1,%.5f)\n",
			segment[0], segment[1], lastError, maxStep, lastGains.R, lastGains.B)
	}
	fmt.Fprintln(file)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
