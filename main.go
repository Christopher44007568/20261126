package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	layersDir    = "layers"
	artifactsDir = "artifacts"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return err
	}
	files, err := WriteSourceLayers(layersDir)
	if err != nil {
		return err
	}
	layers, err := LoadLayerFiles(files)
	if err != nil {
		return err
	}
	scene := Compose(layers)
	if err := verifyScene(scene); err != nil {
		return err
	}

	merged := RenderMergedScene(scene)
	canvas := RenderScene(scene)
	ppm := canvas.PPM()
	preview, err := canvas.PNG()
	if err != nil {
		return err
	}
	sourcesJSON, err := RenderOpinionJSON(scene)
	if err != nil {
		return err
	}
	outputs := map[string][]byte{
		"merged_scene.usda":    merged,
		"final.ppm":            ppm,
		"final_preview.png":    preview,
		"opinion_sources.json": sourcesJSON,
		"opinion_sources.txt":  RenderOpinionText(scene),
	}
	perm, err := runPermutationCheck(files)
	if err != nil {
		return err
	}
	outputs["merged_forward_order.usda"] = perm.forwardMerged
	outputs["merged_reverse_order.usda"] = perm.reverseMerged
	outputs["scene_forward_order.ppm"] = perm.forwardPPM
	outputs["scene_reverse_order.ppm"] = perm.reversePPM
	outputs["permutation_check.txt"] = perm.report
	if err := writeArtifacts(outputs); err != nil {
		return err
	}
	if err := writeChecksums(outputs); err != nil {
		return err
	}

	fmt.Printf("layers: %d -> %s\n", len(layers), strings.Join(files, ", "))
	fmt.Printf("merged scene: %s/merged_scene.usda\n", artifactsDir)
	fmt.Printf("final image: %s/final.ppm (%dx%d)\n", artifactsDir, canvasW, canvasH)
	fmt.Printf("strongest /Stage/StackProbe opinion: %s strength=%d (%s:%d)\n",
		perm.scene.Winner("/Stage/StackProbe", "inputs:displayColor").LayerID,
		perm.scene.Winner("/Stage/StackProbe", "inputs:displayColor").Strength,
		perm.scene.Winner("/Stage/StackProbe", "inputs:displayColor").File,
		perm.scene.Winner("/Stage/StackProbe", "inputs:displayColor").Line)
	fmt.Printf("24/24 load permutations identical: %t\n", perm.allIdentical)
	return nil
}

func verifyScene(scene *Scene) error {
	wantWinners := map[string]string{
		"/Stage/PanelA":       "L2-warm",
		"/Stage/PanelA/Knob":  "L3-cool",
		"/Stage/PanelB":       "L3-cool",
		"/Stage/PanelC":       "L4-spot",
		"/Stage/PanelC/Badge": "L4-spot",
		"/Stage/PanelD":       "L1-base",
		"/Stage/PanelD/Tag":   "L2-warm",
		"/Stage/StackProbe":   "L4-spot",
	}
	for path, want := range wantWinners {
		op := scene.Winner(path, "inputs:displayColor")
		if op == nil {
			return fmt.Errorf("missing displayColor opinion at %s", path)
		}
		if op.LayerID != want {
			return fmt.Errorf("%s strongest opinion = %s, want %s", path, op.LayerID, want)
		}
	}
	stack := scene.Attribute("/Stage/StackProbe", "inputs:displayColor")
	if len(stack.Opinions) != 4 {
		return fmt.Errorf("/Stage/StackProbe has %d opinions, want 4", len(stack.Opinions))
	}
	for _, bad := range []string{
		"/Stage/PanelA/Knob/Knob",
		"/Stage/PanelD/Tag/Tag",
		"/Stage/PanelC/Badge/Badge",
	} {
		if scene.Prims[bad] != nil {
			return fmt.Errorf("local over changed a child path: unexpected %s", bad)
		}
	}
	return nil
}

func writeArtifacts(outputs map[string][]byte) error {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(artifactsDir, name), outputs[name], 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeChecksums(outputs map[string][]byte) error {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		if name == "checksums.sha256" || name == "checksums.txt" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var sums strings.Builder
	for _, name := range names {
		sum := sha256.Sum256(outputs[name])
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
	}
	if err := os.WriteFile(filepath.Join(artifactsDir, "checksums.sha256"), []byte(sums.String()), 0o644); err != nil {
		return err
	}
	human := "SHA-256 checksums\n=================\n" + sums.String() + "\n(verify from artifacts/: sha256sum -c checksums.sha256)\n"
	return os.WriteFile(filepath.Join(artifactsDir, "checksums.txt"), []byte(human), 0o644)
}
