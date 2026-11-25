package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("hitgroupmap", flag.ContinueOnError)
	outDir := flags.String("out", "out", "directory for PNG and JSON artifacts")
	width := flags.Int("width", 720, "image width in pixels")
	height := flags.Int("height", 450, "image height in pixels")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *outDir == "" {
		return fmt.Errorf("-out must not be empty")
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("-width and -height must be positive")
	}

	scene := NewScene()
	correct := CorrectMapping()
	if err := correct.Validate(scene); err != nil {
		return fmt.Errorf("invalid reference mapping: %w", err)
	}
	broken, err := RemapOne(correct, ObjectStoneWall, GroupGlass)
	if err != nil {
		return err
	}
	if err := broken.Validate(scene); err != nil {
		return fmt.Errorf("invalid broken mapping: %w", err)
	}
	changes := MappingChanges(scene, correct, broken)
	if len(changes) != 1 {
		return fmt.Errorf("self-check requires exactly one changed mapping, got %d", len(changes))
	}

	referenceResult, err := Render(scene, correct, *width, *height)
	if err != nil {
		return err
	}
	brokenResult, err := Render(scene, broken, *width, *height)
	if err != nil {
		return err
	}
	first, err := FindFirstMismatch(scene, referenceResult, brokenResult)
	if err != nil {
		return err
	}
	if !first.Found {
		return fmt.Errorf("deliberate stone-wall/glass mapping error was not observable")
	}

	outputs := map[string]string{
		"material_index":        filepath.Join(*outDir, "material_index.png"),
		"hit_groups":            filepath.Join(*outDir, "hit_groups.png"),
		"shaded":                filepath.Join(*outDir, "shaded.png"),
		"broken_material_index": filepath.Join(*outDir, "broken_material_index.png"),
		"broken_hit_groups":     filepath.Join(*outDir, "broken_hit_groups.png"),
		"broken_shaded":         filepath.Join(*outDir, "broken_shaded.png"),
		"mapping_diff":          filepath.Join(*outDir, "mapping_diff.png"),
		"report":                filepath.Join(*outDir, "report.json"),
	}
	writes := []struct {
		path string
		fn   func(string) error
	}{
		{outputs["material_index"], func(path string) error { return writeMaterialIndexPNG(path, referenceResult) }},
		{outputs["hit_groups"], func(path string) error { return writeHitGroupPNG(path, referenceResult) }},
		{outputs["shaded"], func(path string) error { return writeShadedPNG(path, referenceResult) }},
		{outputs["broken_material_index"], func(path string) error { return writeMaterialIndexPNG(path, brokenResult) }},
		{outputs["broken_hit_groups"], func(path string) error { return writeHitGroupPNG(path, brokenResult) }},
		{outputs["broken_shaded"], func(path string) error { return writeShadedPNG(path, brokenResult) }},
		{outputs["mapping_diff"], func(path string) error { return writeDiffPNG(path, referenceResult, brokenResult, first) }},
	}
	for _, write := range writes {
		if err := write.fn(write.path); err != nil {
			return fmt.Errorf("write %s: %w", write.path, err)
		}
	}
	report := buildReport(scene, referenceResult, correct, broken, changes, first, outputs)
	if err := writeReport(outputs["report"], report); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "rendered %dx%d scene\n", *width, *height)
	fmt.Fprintf(stdout, "outputs: %s\n", *outDir)
	fmt.Fprintln(stdout, first.Summary())
	return nil
}
