package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"atlaspad/atlas"
)

const (
	atlasWidth  = 512
	atlasHeight = 384
	mipLevels   = 4
)

type Result struct {
	OutputDir       string                 `json:"output_dir"`
	SafeOK          bool                   `json:"safe_ok"`
	TightViolations int                    `json:"tight_violations"`
	SafeViolations  int                    `json:"safe_violations"`
	MaxDistanceSafe float64                `json:"max_distance_safe"`
	Utilization     atlas.Utilization      `json:"utilization"`
	SafeReport      atlas.ValidationReport `json:"-"`
	TightReport     atlas.ValidationReport `json:"-"`
}

func Run(outDir string) (*Result, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	sprites := sampleSprites()

	tight, err := atlas.BuildAtlas(sprites, atlas.BuildOptions{
		MipLevels: mipLevels,
		Extrude:   false,
		Pack: atlas.PackOptions{
			Width: atlasWidth, Height: atlasHeight,
			AllowRotation: true, Alignment: 1,
			Policy: atlas.PaddingPolicy{Padding: 0},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build tight atlas: %w", err)
	}

	safe, err := atlas.BuildAtlas(sprites, atlas.BuildOptions{
		MipLevels: mipLevels,
		Extrude:   true,
		Pack: atlas.PackOptions{
			Width: atlasWidth, Height: atlasHeight,
			AllowRotation: true, Alignment: 8,
			Policy: atlas.PaddingPolicy{Padding: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build padded atlas: %w", err)
	}

	validateOpts := atlas.ValidateOptions{Grid: 11, ColorTolerance: 24, MaxViolations: 1600}
	tightReport := atlas.ValidateSampling(tight, "tight-no-padding", validateOpts)
	safeReport := atlas.ValidateSampling(safe, "mip-expanded-padding", validateOpts)
	util := atlas.CalculateUtilization(safe.Layout, mipLevels)

	result := &Result{
		OutputDir: outDir, SafeOK: safeReport.OK,
		TightViolations: tightReport.ViolationCount,
		SafeViolations:  safeReport.ViolationCount,
		MaxDistanceSafe: safeReport.MaxDistance,
		Utilization:     util, SafeReport: safeReport, TightReport: tightReport,
	}

	if err := writeJSON(filepath.Join(outDir, "layout.json"), safe.Layout); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(outDir, "utilization.json"), util); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(outDir, "validation_tight.json"), tightReport); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(outDir, "validation_safe.json"), safeReport); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(outDir, "result.json"), result); err != nil {
		return nil, err
	}

	if err := atlas.WritePNG(filepath.Join(outDir, "atlas_layout.png"), atlas.RenderLayout(safe)); err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(filepath.Join(outDir, "atlas_safe.png"), safe.Base); err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(filepath.Join(outDir, "atlas_tight.png"), tight.Base); err != nil {
		return nil, err
	}
	tightHighlight := atlas.RenderValidationOverlay(tight, tightReport)
	safeHighlight := atlas.RenderValidationOverlay(safe, safeReport)
	if err := atlas.WritePNG(filepath.Join(outDir, "bleed_highlight.png"), tightHighlight); err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(filepath.Join(outDir, "bleed_highlight_safe.png"), safeHighlight); err != nil {
		return nil, err
	}
	comparison := atlas.RenderSideBySide(tightHighlight, safeHighlight, 16)
	if err := atlas.WritePNG(filepath.Join(outDir, "bleed_comparison.png"), comparison); err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(filepath.Join(outDir, "mip_padding.png"), atlas.RenderMipContactSheet(safe)); err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(filepath.Join(outDir, "utilization.png"), atlas.RenderUtilization(util)); err != nil {
		return nil, err
	}
	if err := WriteMarkdownReport(filepath.Join(outDir, "report.md"), tight, safe, tightReport, safeReport, util); err != nil {
		return nil, err
	}

	if !safeReport.OK {
		return result, fmt.Errorf("padded atlas failed runtime sampling validation: %d violations", safeReport.ViolationCount)
	}
	return result, nil
}

func writeJSON(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return err
	}
	return f.Close()
}
