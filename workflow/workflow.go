// Package workflow runs the deterministic texture-container demonstration.
package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"texturelab/atlas"
	"texturelab/catalog"
	"texturelab/colorspace"
	"texturelab/diagnose"
	"texturelab/material"
	"texturelab/texcontainer"
)

// MipSizes are the three requested levels, largest first.
var MipSizes = []int{512, 256, 128}

// Files contains all generated artifact paths.
type Files struct {
	AlbedoContainer           string `json:"albedo_container"`
	NormalContainer           string `json:"normal_container"`
	MislabeledNormalContainer string `json:"mislabeled_normal_container"`
	AlbedoAtlas               string `json:"albedo_atlas_png"`
	NormalAtlas               string `json:"normal_atlas_png"`
	DeltaAtlas                string `json:"normal_srgb_delta_atlas_png"`
	MetadataCSV               string `json:"metadata_csv"`
	MetadataJSON              string `json:"metadata_json"`
	MetadataMarkdown          string `json:"metadata_markdown"`
	DiagnosticJSON            string `json:"diagnostic_json"`
	DiagnosticMarkdown        string `json:"diagnostic_markdown"`
	RoundtripJSON             string `json:"roundtrip_json"`
}

// Result is returned to the CLI and integration tests.
type Result struct {
	OutDir     string           `json:"out_dir"`
	Files      Files            `json:"files"`
	Rows       []catalog.Row    `json:"metadata_rows"`
	Diagnostic *diagnose.Report `json:"normal_srgb_diagnostic"`
	Roundtrip  *RoundtripReport `json:"roundtrip"`
}

// Run generates all textures and reports under outDir.
func Run(outDir string) (*Result, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	paths := makePaths(outDir)

	albedoTexture, _, err := buildAlbedo()
	if err != nil {
		return nil, err
	}
	normalTexture, _, err := buildNormal()
	if err != nil {
		return nil, err
	}
	mislabeledTexture := normalTexture.Retag(colorspace.SRGB)
	mislabeledTexture.Metadata["mislabeled"] = "true"
	mislabeledTexture.Metadata["correct_colorspace"] = colorspace.Linear.String()

	albedoInfo, err := writeContainer(paths.AlbedoContainer, albedoTexture)
	if err != nil {
		return nil, err
	}
	normalInfo, err := writeContainer(paths.NormalContainer, normalTexture)
	if err != nil {
		return nil, err
	}
	mislabeledInfo, err := writeContainer(paths.MislabeledNormalContainer, mislabeledTexture)
	if err != nil {
		return nil, err
	}

	reloadedMaterial, err := material.LoadMaterial(
		"reloaded-procedural-material", paths.AlbedoContainer, paths.NormalContainer)
	if err != nil {
		return nil, err
	}
	albedoReloaded := reloadedMaterial.Albedo
	normalReloaded := reloadedMaterial.Normal
	mislabeledReloaded, err := material.LoadFile(paths.MislabeledNormalContainer)
	if err != nil {
		return nil, err
	}

	roundtrip, err := verifyRoundtrip(paths, albedoTexture, albedoReloaded, normalTexture, normalReloaded)
	if err != nil {
		return nil, err
	}
	if err := catalog.WriteJSONFile(paths.RoundtripJSON, roundtrip); err != nil {
		return nil, err
	}

	albedoSheet, err := atlas.HorizontalSheet(imagesFromTexture(albedoReloaded), nil)
	if err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(paths.AlbedoAtlas, albedoSheet.Image); err != nil {
		return nil, err
	}
	normalSheet, err := atlas.HorizontalSheet(imagesFromTexture(normalReloaded), nil)
	if err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(paths.NormalAtlas, normalSheet.Image); err != nil {
		return nil, err
	}

	diagnosticReport, err := diagnose.CompareInterpretations(
		"procedural-normal", normalReloaded, mislabeledReloaded, diagnose.DefaultThreshold)
	if err != nil {
		return nil, err
	}
	if !diagnosticReport.Mismatch || diagnosticReport.FirstFailingLevel != 0 {
		return nil, fmt.Errorf("workflow: expected stable normal/sRGB mismatch from level 0, got %+v", diagnosticReport)
	}
	if err := catalog.WriteJSONFile(paths.DiagnosticJSON, diagnosticReport); err != nil {
		return nil, err
	}
	if err := diagnose.WriteMarkdown(paths.DiagnosticMarkdown, diagnosticReport); err != nil {
		return nil, err
	}

	deltaImages, err := deltaImages(normalReloaded, mislabeledReloaded, 3.0)
	if err != nil {
		return nil, err
	}
	deltaSheet, err := atlas.HorizontalSheet(deltaImages, nil)
	if err != nil {
		return nil, err
	}
	if err := atlas.WritePNG(paths.DeltaAtlas, deltaSheet.Image); err != nil {
		return nil, err
	}

	rows, err := buildRows(rowInputs{
		paths:      paths,
		albedo:     rowAsset{texture: albedoReloaded, info: albedoInfo, sheet: albedoSheet, role: "albedo", asset: "procedural-albedo"},
		normal:     rowAsset{texture: normalReloaded, info: normalInfo, sheet: normalSheet, role: "normal", asset: "procedural-normal"},
		mislabeled: rowAsset{texture: mislabeledReloaded, info: mislabeledInfo, sheet: normalSheet, role: "normal-mislabeled-srgb", asset: "procedural-normal"},
		diagnostic: diagnosticReport,
	})
	if err != nil {
		return nil, err
	}
	if err := catalog.WriteCSV(paths.MetadataCSV, rows); err != nil {
		return nil, err
	}
	if err := catalog.WriteJSON(paths.MetadataJSON, rows); err != nil {
		return nil, err
	}
	if err := catalog.WriteMarkdown(paths.MetadataMarkdown, rows); err != nil {
		return nil, err
	}

	return &Result{
		OutDir: outDir, Files: paths, Rows: rows,
		Diagnostic: diagnosticReport, Roundtrip: roundtrip,
	}, nil
}

func makePaths(dir string) Files {
	return Files{
		AlbedoContainer:           filepath.Join(dir, "albedo_srgb.ktxc"),
		NormalContainer:           filepath.Join(dir, "normal_linear.ktxc"),
		MislabeledNormalContainer: filepath.Join(dir, "normal_mislabeled_srgb.ktxc"),
		AlbedoAtlas:               filepath.Join(dir, "albedo_mip_atlas.png"),
		NormalAtlas:               filepath.Join(dir, "normal_mip_atlas.png"),
		DeltaAtlas:                filepath.Join(dir, "normal_srgb_delta_atlas.png"),
		MetadataCSV:               filepath.Join(dir, "mip_metadata.csv"),
		MetadataJSON:              filepath.Join(dir, "mip_metadata.json"),
		MetadataMarkdown:          filepath.Join(dir, "mip_metadata.md"),
		DiagnosticJSON:            filepath.Join(dir, "normal_srgb_diagnostic.json"),
		DiagnosticMarkdown:        filepath.Join(dir, "normal_srgb_diagnostic.md"),
		RoundtripJSON:             filepath.Join(dir, "roundtrip_report.json"),
	}
}

func writeContainer(path string, texture *texcontainer.Texture) (*texcontainer.Info, error) {
	data, err := texcontainer.Encode(texture)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return texcontainer.Inspect(data)
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
