package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ScenarioResult struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Metrics      Metrics `json:"metrics"`
	StreamPath   string  `json:"stream_path"`
	StreamBytes  int     `json:"stream_bytes"`
	BlockCount   int     `json:"block_count"`
	OpaqueBlocks int     `json:"opaque_blocks"`
	MaskBlocks   int     `json:"mask_blocks"`
	Alpha4Blocks int     `json:"alpha4_blocks"`
	EmptyBlocks  int     `json:"empty_blocks"`
}

type BlockRow struct {
	X, Y int
	Data []byte
	Info BlockInfo
}

func collectBlockRows(records []EncodedBlock, w, h int) ([]BlockRow, error) {
	expected := (w / BlockSide) * (h / BlockSide)
	if len(records) != expected {
		return nil, fmt.Errorf("got %d records for %dx%d", len(records), w, h)
	}
	rows := make([]BlockRow, 0, len(records))
	i := 0
	for y := 0; y < h; y += BlockSide {
		for x := 0; x < w; x += BlockSide {
			info, err := records[i].Info()
			if err != nil {
				return nil, fmt.Errorf("block (%d,%d): %w", x, y, err)
			}
			rows = append(rows, BlockRow{X: x, Y: y, Data: records[i].Data, Info: info})
			i++
		}
	}
	return rows, nil
}

func modeCounts(rows []BlockRow) (opaque, mask, alpha4, empty int) {
	for _, row := range rows {
		switch row.Info.AlphaMode {
		case alphaOpaque:
			opaque++
		case alphaMask:
			mask++
		case alphaNibble:
			alpha4++
		case alphaEmpty:
			empty++
		}
	}
	return
}

func alphaModeName(mode int) string {
	switch mode {
	case alphaOpaque:
		return "opaque"
	case alphaMask:
		return "mask1"
	case alphaNibble:
		return "alpha4"
	case alphaEmpty:
		return "empty"
	default:
		return "?"
	}
}

func hexBytes(data []byte) string {
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, " ")
}

func blockLine(row BlockRow) string {
	if row.Info.AlphaMode == alphaEmpty {
		return fmt.Sprintf("(%02d,%02d) %2dB %-6s %s",
			row.X, row.Y, len(row.Data), alphaModeName(row.Info.AlphaMode), hexBytes(row.Data))
	}
	return fmt.Sprintf("(%02d,%02d) %2dB %-6s base=#%02x%02x%02x spans=%d/%d/%d %s",
		row.X, row.Y, len(row.Data), alphaModeName(row.Info.AlphaMode),
		row.Info.Base.R, row.Info.Base.G, row.Info.Base.B,
		row.Info.SpanR, row.Info.SpanG, row.Info.SpanB, hexBytes(row.Data))
}

func writeBlockText(path string, rows []BlockRow) error {
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(blockLine(row))
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeMetricsJSON(path string, results []ScenarioResult) error {
	payload := struct {
		Codec        string           `json:"codec"`
		ErrorAmplify int              `json:"error_amplify"`
		Scenarios    []ScenarioResult `json:"scenarios"`
	}{
		Codec:        "BD4X: RGB565 base + 2-bit per-channel increments; 1-bit alpha mask or 4-bit alpha",
		ErrorAmplify: errorAmplify,
		Scenarios:    results,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeTextReport(path string, results []ScenarioResult, rowsByID map[string][]BlockRow) error {
	var b strings.Builder
	b.WriteString("BD4X 4x4 base + delta compression report\n")
	b.WriteString("=======================================\n\n")
	b.WriteString("Layout in overview.png: rows = smooth / sharp edges / transparent holes;\n")
	b.WriteString("columns = original / decoded / RGBA error (max channel diff x8).\n")
	b.WriteString("Magenta in the error image means a transparent/visible classification break.\n\n")
	for _, r := range results {
		m := r.Metrics
		fmt.Fprintf(&b, "[%s] %dx%d  blocks=%d  bytes=%d  raw=%d  ratio=%.2fx\n",
			r.ID, r.Width, r.Height, r.BlockCount, r.StreamBytes, m.RawBytes, m.CompressionRatio)
		fmt.Fprintf(&b, "  block modes: opaque=%d mask1=%d alpha4=%d empty=%d\n",
			r.OpaqueBlocks, r.MaskBlocks, r.Alpha4Blocks, r.EmptyBlocks)
		fmt.Fprintf(&b, "  visible RGB MAE=%.3f  PSNR=%.2f dB  alpha MAE=%.3f  premult RGBA MAE=%.3f\n",
			m.VisibleRGBMAE, m.RGBPSNR, m.AlphaMAE, m.PremultipliedRGBAMAE)
		fmt.Fprintf(&b, "  ALPHA AUDIT: class errors=%d  hole fills=%d  hole erosion=%d  boundary alpha MAE=%.3f  boundary visible RGB MAE=%.3f\n",
			m.AlphaClassErrors, m.HoleFillErrors, m.HoleErosionErrors, m.BoundaryAlphaMAE, m.BoundaryRGBMAE)
		fmt.Fprintf(&b, "  worst RGB block MAE=%.3f at (%d,%d)\n",
			m.WorstBlockRGBMAE, m.WorstBlockX, m.WorstBlockY)
		if m.AlphaClassErrors == 0 && r.TransparentCount() > 0 {
			b.WriteString("  transparent edge audit: PASS (no visible/transparent flips)\n")
		}
		b.WriteString("  representative compressed bytes:\n")
		for _, row := range representativeRows(rowsByID[r.ID], 5) {
			fmt.Fprintf(&b, "    %s\n", blockLine(row))
		}
		fmt.Fprintf(&b, "  full block bytes: artifacts/blocks_%s.txt\n\n", r.ID)
	}
	b.WriteString("The mean error is intentionally split from alpha classification errors:\n")
	b.WriteString("a low RGB MAE cannot turn hole fills/erosion into a passing result.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (r ScenarioResult) TransparentCount() int {
	return r.Metrics.TransparentPixels
}

func representativeRows(rows []BlockRow, n int) []BlockRow {
	if len(rows) <= n {
		return rows
	}
	seen := map[int]bool{}
	var out []BlockRow
	for _, row := range rows {
		if !seen[row.Info.AlphaMode] {
			seen[row.Info.AlphaMode] = true
			out = append(out, row)
		}
	}
	for _, row := range rows {
		if len(out) >= n {
			break
		}
		duplicate := false
		for _, picked := range out {
			if picked.X == row.X && picked.Y == row.Y {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, row)
		}
	}
	return out
}

func writeHexDump(path string, stream []byte) error {
	var b strings.Builder
	for off := 0; off < len(stream); off += 24 {
		end := off + 24
		if end > len(stream) {
			end = len(stream)
		}
		fmt.Fprintf(&b, "%06x  %s\n", off, hexBytes(stream[off:end]))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func ensureArtifactDirs(root string) (string, string, error) {
	streamDir := filepath.Join(root, "streams")
	if err := os.MkdirAll(streamDir, 0o755); err != nil {
		return "", "", err
	}
	return root, streamDir, nil
}
