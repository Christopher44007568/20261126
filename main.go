package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	root := flag.String("root", ".", "repository root containing fixtures/ and receiving artifacts/")
	flag.Parse()

	rootDir := *root
	if rootDir == "." {
		if modRoot, err := findModuleRoot(); err == nil {
			rootDir = modRoot
		}
	}

	app, err := NewApp(rootDir)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(app.ArtDir, 0o755); err != nil {
		fatal(err)
	}

	manifest, err := app.BuildManifest()
	if err != nil {
		fatal(err)
	}

	boards := []struct {
		file  string
		spec  BoardSpec
		image *ImageInfo
	}{
		{"before.png", BoardSpec{
			Key: "before", Title: "REFERENCE CHANGE PROOF",
			Subtitle:     "BEFORE REFERENCE / BOTH PATHS RESOLVE TO THE SAME CORAL MATERIAL",
			Footer:       "BEFORE: REFERENCED INSTANCE AND INDEPENDENT COPY MATCH",
			FooterAccent: colAmber, Stage: manifest.Before,
		}, &manifest.Before.Image},
		{"after.png", BoardSpec{
			Key: "after", Title: "AFTER REFERENCE CHANGE",
			Subtitle:     "LIVE MATERIAL UPDATED THROUGH THE REFERENCE CHAIN / COPY UNCHANGED",
			Footer:       "AFTER: ONLY THE REFERENCED SCENE LOCATION UPDATED",
			FooterAccent: colGreen, Stage: manifest.After,
		}, &manifest.After.Image},
		{"missing.png", BoardSpec{
			Key: "missing", Title: "MISSING ASSET CONTRACT",
			Subtitle:     "GEOMETRY REFERENCE IN THE LIVE MODEL NOW POINTS TO AN ABSENT FILE",
			Footer:       "MISSING: READABLE PLACEHOLDER + ERROR OVERLAY / NOT AN EMPTY SCENE",
			FooterAccent: colRed, Stage: manifest.Missing,
		}, &manifest.Missing.Image},
	}

	for _, b := range boards {
		canvas := RenderBoard(b.spec)
		info, err := writePNG(canvas, filepath.Join(app.ArtDir, b.file), b.file)
		if err != nil {
			fatal(err)
		}
		*b.image = info
		fmt.Printf("rendered %s (%dx%d, %d bytes, sha %.12s)\n", info.Name, info.Width, info.Height, info.Bytes, info.SHA256)
	}

	if err := writeManifest(manifest); err != nil {
		fatal(err)
	}

	if err := app.ResetWork(); err != nil {
		fatal(err)
	}

	printChangeLog(manifest)

	if failures := Verify(manifest, app.ArtDir); len(failures) > 0 {
		for _, msg := range failures {
			fmt.Println("SELFCHECK FAIL:", msg)
		}
		os.Exit(1)
	}
	fmt.Println("self-check: all reference-chain and placeholder checks passed")
}

func writePNG(c *Canvas, fullPath, name string) (ImageInfo, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, c.Img); err != nil {
		return ImageInfo{}, err
	}
	if err := os.WriteFile(fullPath, buf.Bytes(), 0o644); err != nil {
		return ImageInfo{}, err
	}
	sum := sha256.Sum256(buf.Bytes())
	return ImageInfo{
		Name:   name,
		Path:   fullPath,
		Width:  c.W(),
		Height: c.H(),
		Bytes:  int64(buf.Len()),
		SHA256: hex.EncodeToString(sum[:]),
	}, nil
}

func writeManifest(m *Manifest) error {
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(m.ManifestPath, body, 0o644)
}

func printChangeLog(m *Manifest) {
	fmt.Println()
	fmt.Println("Final path source for every change:")
	for _, ch := range m.Changes {
		fmt.Printf("- [%s] %s: %s -> %s\n", ch.Kind, ch.Field, ch.Before, ch.After)
		fmt.Printf("    changed : %s\n", ch.ChangedPath)
		fmt.Printf("    final   : %s\n", ch.FinalPath)
		fmt.Printf("    source  : %s\n", ch.Source)
	}
	fmt.Println()
	fmt.Printf("Reference locality: onlyReferencingPathsUpdate=%v\n", m.Impact.OnlyReferencingPathsUpdate)
	fmt.Printf("  updated   %s via %s\n", m.Impact.UpdatedScene, m.Impact.UpdatedSource)
	fmt.Printf("  unchanged %s via %s\n", m.Impact.UnchangedScene, m.Impact.UnchangedSource)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "refproof:", err)
	os.Exit(1)
}
