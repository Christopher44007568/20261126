package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CheckReport struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

type FileReport struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	PixelHash string `json:"pixelHash,omitempty"`
}

type BindingReport struct {
	Node       string  `json:"node"`
	Slot       string  `json:"slot"`
	Variant    Variant `json:"variant"`
	MaterialID string  `json:"materialId"`
	Shader     string  `json:"shader"`
	SourcePath string  `json:"sourcePath"`
	Resolved   bool    `json:"resolved"`
}

type PixelComparison struct {
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	DifferentPixels int    `json:"differentPixels"`
	Identical       bool   `json:"identical"`
	PixelHashBefore string `json:"pixelHashBefore"`
	PixelHashAfter  string `json:"pixelHashAfter"`
}

type RestoreReport struct {
	OverrideWinter PixelComparison `json:"overrideToInheritedWinter"`
	OriginalSummer PixelComparison `json:"switchBackToSummer"`
	RebindRepair   PixelComparison `json:"nameOnlyBrokenThenRebound"`
}

type Report struct {
	Schema          string             `json:"schema"`
	Asset           string             `json:"asset"`
	VariantSet      string             `json:"variantSet"`
	OutputDirectory string             `json:"outputDirectory"`
	GoVersion       string             `json:"goVersion"`
	Pass            bool               `json:"pass"`
	Checks          []CheckReport      `json:"checks"`
	Outputs         []FileReport       `json:"outputs"`
	CurrentPath     Resolution         `json:"currentSelectionPath"`
	PropagationPath Resolution         `json:"propagationDemoPath"`
	Selections      map[string]Variant `json:"authoredSelections"`
	Bindings        []BindingReport    `json:"bindings"`
	Restore         RestoreReport      `json:"restore"`
}

func compareImages(before, after *Image) PixelComparison {
	result := PixelComparison{
		Width:           before.W,
		Height:          before.H,
		PixelHashBefore: before.PixelSHA256(),
		PixelHashAfter:  after.PixelSHA256(),
	}
	if before.W != after.W || before.H != after.H {
		result.DifferentPixels = before.W*before.H + after.W*after.H
		result.Identical = false
		return result
	}
	for y := 0; y < before.H; y++ {
		for x := 0; x < before.W; x++ {
			i := before.offset(x, y)
			if before.Pix[i] != after.Pix[i] ||
				before.Pix[i+1] != after.Pix[i+1] ||
				before.Pix[i+2] != after.Pix[i+2] {
				result.DifferentPixels++
			}
		}
	}
	result.Identical = result.DifferentPixels == 0 && result.PixelHashBefore == result.PixelHashAfter
	return result
}

func sha256File(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data)), nil
}

func (r *Report) writePPM(outDir, name string, im *Image) error {
	path := filepath.Join(outDir, name)
	if err := WritePPMFile(path, im); err != nil {
		return err
	}
	sum, size, err := sha256File(path)
	if err != nil {
		return err
	}
	r.Outputs = append(r.Outputs, FileReport{
		Name: name, Path: path, Bytes: size, SHA256: sum,
		Width: im.W, Height: im.H, PixelHash: im.PixelSHA256(),
	})
	return nil
}

func (r *Report) writeJSON(outDir, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	sum, size, err := sha256File(path)
	if err != nil {
		return err
	}
	r.Outputs = append(r.Outputs, FileReport{Name: name, Path: path, Bytes: size, SHA256: sum})
	return nil
}

func writeText(outDir, name, text string) (FileReport, error) {
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return FileReport{}, err
	}
	sum, size, err := sha256File(path)
	if err != nil {
		return FileReport{}, err
	}
	return FileReport{Name: name, Path: path, Bytes: size, SHA256: sum}, nil
}

func selectionPathText(current, propagation Resolution, stage *Stage, restore RestoreReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "variant set: %s\n", current.VariantSet)
	b.WriteString("\n[current selection path]\n")
	writeResolutionText(&b, current)

	b.WriteString("\n[parent-child propagation demo]\n")
	writeResolutionText(&b, propagation)

	b.WriteString("\n[restore verification]\n")
	fmt.Fprintf(&b, "override -> inherited winter: identical=%v differentPixels=%d\n",
		restore.OverrideWinter.Identical, restore.OverrideWinter.DifferentPixels)
	fmt.Fprintf(&b, "switch back to summer:       identical=%v differentPixels=%d\n",
		restore.OriginalSummer.Identical, restore.OriginalSummer.DifferentPixels)
	fmt.Fprintf(&b, "name-only then rebind:       identical=%v differentPixels=%d\n",
		restore.RebindRepair.Identical, restore.RebindRepair.DifferentPixels)

	b.WriteString("\n[authored selections]\n")
	if len(stage.selections) == 0 {
		b.WriteString("(none)\n")
	}
	for _, path := range SortedSelectionKeys(stage.selections) {
		fmt.Fprintf(&b, "%s=%s\n", path, stage.selections[path])
	}
	return b.String()
}

func writeResolutionText(b *strings.Builder, resolution Resolution) {
	fmt.Fprintf(b, "target:  %s\n", resolution.TargetPath)
	fmt.Fprintf(b, "active:  %s\n", resolution.Variant)
	fmt.Fprintf(b, "source:  %s\n", resolution.SourcePath)
	b.WriteString("chain:\n")
	for _, step := range resolution.Steps {
		mark := "inherit"
		if step.Authored {
			mark = "AUTHOR " + string(step.AuthoredVariant)
		}
		fmt.Fprintf(b, "  - %-42s %-18s active=%s source=%s\n",
			step.Path, mark, step.Active, step.SourcePath)
	}
}

func bindingReports(stage *Stage) []BindingReport {
	var out []BindingReport
	for _, node := range stage.orderedNodes {
		for _, slot := range node.Slots {
			binding := node.Bindings[slot]
			out = append(out, BindingReport{
				Node: node.Path, Slot: slot, Variant: binding.Variant,
				MaterialID: binding.MaterialID, Shader: binding.Shader,
				SourcePath: binding.SourcePath, Resolved: binding.Resolved(),
			})
		}
	}
	return out
}
