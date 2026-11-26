package render

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

var expectedFiles = []string{
	"bit_gradient_puzzle.png",
	"sheet_8bit.png",
	"sheet_10bit_to_8bit.png",
	"sheet_12bit_to_8bit.png",
	"master_10bit_16bit.png",
	"master_12bit_16bit.png",
	"subcode_dither_ladder.png",
	"metrics.json",
	"index.html",
}

func Generate(out string) ([]string, error) {
	report := buildMetrics()
	report.OutputFiles = expectedFiles

	if err := savePNG(out, "bit_gradient_puzzle.png", renderPuzzle()); err != nil {
		return nil, err
	}
	for i, p := range Pipelines {
		name := "sheet_8bit.png"
		if i > 0 {
			name = fmt.Sprintf("sheet_%dbit_to_8bit.png", p.Bits)
		}
		if err := savePNG(out, name, renderSheet(p)); err != nil {
			return nil, err
		}
	}
	if err := savePNG(out, "subcode_dither_ladder.png", renderLadder(report)); err != nil {
		return nil, err
	}
	for _, bits := range []int{10, 12} {
		img, err := renderMaster(bits)
		if err != nil {
			return nil, err
		}
		if err := savePNG(out, fmt.Sprintf("master_%dbit_16bit.png", bits), img); err != nil {
			return nil, err
		}
	}
	if err := writeJSON(out, "metrics.json", report); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte(indexHTML(report)), 0o644); err != nil {
		return nil, err
	}
	return expectedFiles, nil
}

func savePNG(dir, name string, img image.Image) error {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(f, img); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func writeJSON(dir, name string, value any) error {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
