package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// App holds the absolute on-disk locations for one proof run.
type App struct {
	Root       string
	FixtureDir string
	WorkDir    string
	ArtDir     string
	Width      int
	Height     int
}

// NewApp resolves root to an absolute path and derives all child locations.
func NewApp(root string) (*App, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root %q: %w", root, err)
	}
	return &App{
		Root:       abs,
		FixtureDir: filepath.Join(abs, "fixtures", "showroom"),
		WorkDir:    filepath.Join(abs, "artifacts", "work", "showroom"),
		ArtDir:     filepath.Join(abs, "artifacts"),
		Width:      1080,
		Height:     720,
	}, nil
}

// SeedFixture materialises the pristine source fixture under fixtures/showroom.
func (a *App) SeedFixture() error {
	return writeSeedTree(a.FixtureDir)
}

// ResetWork replaces the working tree with a clean copy of the seed dataset.
func (a *App) ResetWork() error {
	if err := os.RemoveAll(a.WorkDir); err != nil {
		return fmt.Errorf("reset work dir: %w", err)
	}
	return writeSeedTree(a.WorkDir)
}

func writeSeedTree(dst string) error {
	for rel, body := range seedShowroom() {
		full := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", full, err)
		}
	}
	return nil
}

func (a *App) workPath(rel string) string {
	return filepath.Join(a.WorkDir, filepath.FromSlash(rel))
}

// findModuleRoot walks up from the working directory to the directory
// containing go.mod, so `go run .` keeps stable absolute output paths.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return ".", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ".", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// readDoc loads one JSON asset from the working tree.
func readDoc(abs string) (refDoc, map[string]any, error) {
	body, err := os.ReadFile(abs)
	if err != nil {
		return refDoc{}, nil, err
	}
	var doc refDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return refDoc{}, nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return refDoc{}, nil, err
	}
	return doc, raw, nil
}

// writeRaw writes a generic JSON document with stable two-space indentation.
func writeRaw(abs string, raw map[string]any) error {
	body, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, body, 0o644)
}

// setModelReference rewrites one references.<slot> entry in a model file.
func (a *App) setModelReference(modelRel, slot, newRef string) (string, error) {
	abs := a.workPath(modelRel)
	_, raw, err := readDoc(abs)
	if err != nil {
		return "", err
	}
	refs, _ := raw["references"].(map[string]any)
	if refs == nil {
		refs = map[string]any{}
		raw["references"] = refs
	}
	old, _ := refs[slot].(string)
	refs[slot] = newRef
	if err := writeRaw(abs, raw); err != nil {
		return old, err
	}
	return old, nil
}

// commitUpdatedMaterial writes the "after" live material asset.
//
// The proof deliberately models an external save that updated baseColor but
// dropped the required kind discriminator, so the resolver reports a contract
// error and the board renders the error overlay instead of a silent fallback.
func (a *App) commitUpdatedMaterial() error {
	abs := a.workPath("materials/live.json")
	updated := map[string]any{
		"name":        "Coral Dusk",
		"baseColor":   "#6C7BFF",
		"accentColor": "#FFB547",
		"inkColor":    "#10212B",
	}
	return writeRaw(abs, updated)
}
