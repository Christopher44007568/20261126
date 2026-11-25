// Package catalog writes the per-mip metadata table used to locate levels.
package catalog

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Row is one texture/level record in the metadata table.
type Row struct {
	Asset             string  `json:"asset"`
	Role              string  `json:"role"`
	ContainerFile     string  `json:"container_file"`
	ColorSpaceMarker  string  `json:"color_space_marker"`
	Level             int     `json:"level"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	PixelBytes        int     `json:"pixel_bytes"`
	ContainerOffset   uint64  `json:"container_offset"`
	ContainerSize     uint64  `json:"container_size"`
	CRC32             string  `json:"crc32"`
	PayloadSHA256     string  `json:"payload_sha256"`
	LinearSHA256      string  `json:"linear_material_sha256"`
	AtlasFile         string  `json:"atlas_file,omitempty"`
	AtlasX            int     `json:"atlas_x"`
	AtlasY            int     `json:"atlas_y"`
	AtlasWidth        int     `json:"atlas_width"`
	AtlasHeight       int     `json:"atlas_height"`
	DiagnosticMax     float64 `json:"normal_srgb_max_delta,omitempty"`
	DiagnosticMean    float64 `json:"normal_srgb_mean_delta,omitempty"`
	DiagnosticLocated bool    `json:"diagnostic_located,omitempty"`
}

var csvHeader = []string{
	"asset", "role", "container_file", "color_space_marker", "level",
	"width", "height", "pixel_bytes", "container_offset", "container_size",
	"crc32", "payload_sha256", "linear_material_sha256", "atlas_file",
	"atlas_x", "atlas_y", "atlas_width", "atlas_height",
	"normal_srgb_max_delta", "normal_srgb_mean_delta", "diagnostic_located",
}

// WriteJSON writes rows as indented JSON.
func WriteJSON(path string, rows []Row) error {
	return WriteJSONFile(path, rows)
}

// WriteJSONFile writes any value as indented JSON and creates its directory.
func WriteJSONFile(path string, value any) error {
	return writeJSON(path, value)
}

// WriteCSV writes a stable header plus one row per texture mip level.
func WriteCSV(path string, rows []Row) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(csvHeader); err != nil {
		return err
	}
	for _, row := range rows {
		record := []string{
			row.Asset, row.Role, row.ContainerFile, row.ColorSpaceMarker,
			fmt.Sprintf("%d", row.Level), fmt.Sprintf("%d", row.Width),
			fmt.Sprintf("%d", row.Height), fmt.Sprintf("%d", row.PixelBytes),
			fmt.Sprintf("%d", row.ContainerOffset), fmt.Sprintf("%d", row.ContainerSize),
			row.CRC32, row.PayloadSHA256, row.LinearSHA256, row.AtlasFile,
			fmt.Sprintf("%d", row.AtlasX), fmt.Sprintf("%d", row.AtlasY),
			fmt.Sprintf("%d", row.AtlasWidth), fmt.Sprintf("%d", row.AtlasHeight),
			fmt.Sprintf("%.9f", row.DiagnosticMax), fmt.Sprintf("%.9f", row.DiagnosticMean),
			fmt.Sprintf("%t", row.DiagnosticLocated),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return file.Close()
}

// WriteMarkdown writes a compact human-readable metadata table.
func WriteMarkdown(path string, rows []Row) error {
	var builder strings.Builder
	builder.WriteString("# Mip Metadata\n\n")
	builder.WriteString("| Asset | Marker | Level | Size | Offset | CRC32 | Atlas rect | Linear SHA256 (first 12) |\n")
	builder.WriteString("|---|---|---:|---:|---:|---|---|---|\n")
	for _, row := range rows {
		builder.WriteString(fmt.Sprintf(
			"| %s | `%s` | %d | %dx%d | %d | `%s` | `%s@%d,%d %dx%d` | `%s` |\n",
			escape(row.Asset), row.ColorSpaceMarker, row.Level, row.Width, row.Height,
			row.ContainerOffset, row.CRC32, escape(row.AtlasFile), row.AtlasX, row.AtlasY,
			row.AtlasWidth, row.AtlasHeight, safePrefix(row.LinearSHA256, 12)))
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func escape(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func safePrefix(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}
