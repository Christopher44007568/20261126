package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"dds-preview/internal/dds"
)

// WritePreview emits one PPM per face/mip plus an original-size material
// sphere. It must only be called after a successful preview load.
func WritePreview(tex *dds.Texture, res *Result, outDir string) error {
	if tex == nil {
		return fmt.Errorf("no valid texture to preview")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := removeOldImages(outDir); err != nil {
		return err
	}

	artifacts := make([]Artifact, 0, tex.FaceCount*tex.MipCount+1)
	for fi := 0; fi < tex.FaceCount; fi++ {
		for mi := 0; mi < tex.MipCount; mi++ {
			img := tex.Faces[fi].Mips[mi]
			name := fmt.Sprintf("mip_%d.ppm", mi)
			if tex.FaceCount > 1 {
				name = fmt.Sprintf("face_%d_mip_%d.ppm", fi, mi)
			}
			path := filepath.Join(outDir, name)
			ppm, err := PPM(img)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if err := writeFile(path, ppm, 0o644); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			artifacts = append(artifacts, Artifact{
				Kind: "mip", Face: fi, Level: mi, Path: path,
				Width: img.Width, Height: img.Height,
			})
		}
	}

	sphere := RenderSphere(tex)
	path := filepath.Join(outDir, "sphere.ppm")
	ppm, err := PPM(sphere)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := writeFile(path, ppm, 0o644); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	artifacts = append(artifacts, Artifact{
		Kind: "sphere", Face: 0, Level: 0, Path: path,
		Width: sphere.Width, Height: sphere.Height,
	})
	if res != nil {
		res.Artifacts = artifacts
	}
	return nil
}

func removeOldImages(outDir string) error {
	patterns := []string{"mip_*.ppm", "face_*_mip_*.ppm", "sphere*.ppm"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(outDir, pattern))
		if err != nil {
			return err
		}
		for _, match := range matches {
			if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// WriteReport always writes a machine-readable field-level report. Images are
// intentionally not touched, which preserves the previous valid texture.
func WriteReport(res *Result, outDir string) error {
	if res == nil {
		return fmt.Errorf("nil result")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if res.Errors == nil {
		res.Errors = []dds.FieldError{}
	}
	sort.SliceStable(res.Artifacts, func(i, j int) bool {
		if res.Artifacts[i].Face != res.Artifacts[j].Face {
			return res.Artifacts[i].Face < res.Artifacts[j].Face
		}
		return res.Artifacts[i].Level < res.Artifacts[j].Level
	})
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(outDir, "report.json"), data, 0o644)
}
