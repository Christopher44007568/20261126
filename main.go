package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	show := flag.Int("show-blocks", 5, "representative block records to print per scenario")
	flag.Parse()
	if err := run(*outDir, *show); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string, showBlocks int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	_, streamDir, err := ensureArtifactDirs(outDir)
	if err != nil {
		return err
	}

	scenarios := GenerateScenarios()
	decoded := make([]*Bitmap, len(scenarios))
	results := make([]ScenarioResult, len(scenarios))
	rowsByID := make(map[string][]BlockRow, len(scenarios))
	var allHex strings.Builder
	for i, scenario := range scenarios {
		stream, records, err := CompressBitmap(scenario.Original)
		if err != nil {
			return fmt.Errorf("%s compress: %w", scenario.ID, err)
		}
		decoded[i], err = DecompressBitmap(stream)
		if err != nil {
			return fmt.Errorf("%s decompress: %w", scenario.ID, err)
		}
		metrics, err := CompareMetrics(scenario.Original, decoded[i])
		if err != nil {
			return fmt.Errorf("%s metrics: %w", scenario.ID, err)
		}
		SetCompressedSize(&metrics, len(stream))
		rows, err := collectBlockRows(records, scenario.Original.W, scenario.Original.H)
		if err != nil {
			return err
		}
		opaque, mask, alpha4, empty := modeCounts(rows)
		result := ScenarioResult{
			ID:           scenario.ID,
			Title:        scenario.Title,
			Width:        scenario.Original.W,
			Height:       scenario.Original.H,
			Metrics:      metrics,
			StreamPath:   filepath.ToSlash(filepath.Join("streams", scenario.ID+".bd4")),
			StreamBytes:  len(stream),
			BlockCount:   len(rows),
			OpaqueBlocks: opaque,
			MaskBlocks:   mask,
			Alpha4Blocks: alpha4,
			EmptyBlocks:  empty,
		}
		results[i] = result
		rowsByID[scenario.ID] = rows

		streamPath := filepath.Join(outDir, "streams", scenario.ID+".bd4")
		if err := os.WriteFile(streamPath, stream, 0o644); err != nil {
			return err
		}
		if err := writeHexDump(filepath.Join(streamDir, scenario.ID+".hex.txt"), stream); err != nil {
			return err
		}
		if err := writeBlockText(filepath.Join(outDir, "blocks_"+scenario.ID+".txt"), rows); err != nil {
			return err
		}
		fmt.Fprintf(&allHex, "== %s (%d bytes) ==\n", scenario.ID, len(stream))
		for off := 0; off < len(stream); off += 24 {
			end := off + 24
			if end > len(stream) {
				end = len(stream)
			}
			fmt.Fprintf(&allHex, "%06x  %s\n", off, hexBytes(stream[off:end]))
		}
		allHex.WriteByte('\n')
	}

	_, err = writeRenderedArtifacts(outDir, scenarios, decoded)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "compressed_hex.txt"), []byte(allHex.String()), 0o644); err != nil {
		return err
	}
	if err := writeMetricsJSON(filepath.Join(outDir, "metrics.json"), results); err != nil {
		return err
	}
	if err := writeTextReport(filepath.Join(outDir, "report.txt"), results, rowsByID); err != nil {
		return err
	}
	if err := writeHTMLReport(filepath.Join(outDir, "report.html"), results, rowsByID); err != nil {
		return err
	}

	fmt.Println("BD4X 4x4 base+delta compression")
	for _, r := range results {
		m := r.Metrics
		fmt.Printf("%-8s %dx%d  %d B -> %d B  ratio %.2fx  RGB MAE %.2f  alpha flips %d\n",
			r.ID, r.Width, r.Height, m.RawBytes, r.StreamBytes, m.CompressionRatio,
			m.VisibleRGBMAE, m.AlphaClassErrors)
		if m.TransparentPixels > 0 {
			status := "PASS"
			if m.AlphaClassErrors > 0 {
				status = "FAIL"
			}
			fmt.Printf("         transparent edge audit: %s (fills %d, erosion %d, boundary alpha MAE %.2f)\n",
				status, m.HoleFillErrors, m.HoleErosionErrors, m.BoundaryAlphaMAE)
		}
		for _, row := range representativeRows(rowsByID[r.ID], showBlocks) {
			fmt.Printf("         %s\n", blockLine(row))
		}
	}
	fmt.Printf("rendered: %s\nreport: %s\nstreams: %s\n",
		filepath.Join(outDir, "overview.png"),
		filepath.Join(outDir, "report.html"),
		streamDir)
	return nil
}
