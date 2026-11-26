// Command materialover demonstrates a material "over" engine.
//
// With no arguments it runs the bundled demo (one valid override file and one
// invalid one). Given arguments it runs a single case:
//
//	materialover <scene.json> <override.json> <outdir>
//
// It renders three images per case (base / over / final), prints the list of
// overridden properties, and — when an override exceeds the allowed target
// paths or uses a wrong property type — rejects the whole patch and preserves
// the original picture byte-for-byte.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"materialover/over"
	"materialover/render"
	"materialover/scene"
)

type propertyInfo struct {
	Property string      `json:"property"`
	Path     string      `json:"path"`
	Before   interface{} `json:"before"`
	After    interface{} `json:"after"`
}

type meshReport struct {
	MeshID     string         `json:"meshId"`
	Properties []propertyInfo `json:"properties"`
}

type caseReport struct {
	Name            string       `json:"name"`
	Accepted        bool         `json:"accepted"`
	Error           string       `json:"error,omitempty"`
	OverriddenProps []meshReport `json:"overriddenProperties,omitempty"`
	Images          imageFiles   `json:"images"`
	BasePreserved   bool         `json:"basePreserved,omitempty"`
}

type imageFiles struct {
	Base  string `json:"base"`
	Over  string `json:"over"`
	Final string `json:"final"`
}

type fullReport struct {
	Cases []caseReport `json:"cases"`
}

func main() {
	if len(os.Args) == 1 {
		if err := runDemo(); err != nil {
			fail(err)
		}
		return
	}
	if len(os.Args) != 4 {
		fail(fmt.Errorf("usage: %s <scene.json> <override.json> <outdir>", filepath.Base(os.Args[0])))
	}
	rep, err := runCase("custom", os.Args[1], os.Args[2], os.Args[3])
	if err != nil {
		fail(err)
	}
	printSummary(rep)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func runDemo() error {
	root := "demo"
	out := filepath.Join("out", "valid")
	valid, err := runCase("valid", filepath.Join(root, "scene.json"),
		filepath.Join(root, "override_valid.json"), out)
	if err != nil {
		return err
	}
	printSummary(valid)

	outBad := filepath.Join("out", "invalid")
	bad, err := runCase("invalid", filepath.Join(root, "scene.json"),
		filepath.Join(root, "override_invalid.json"), outBad)
	if err != nil {
		return err
	}
	printSummary(bad)

	report := fullReport{Cases: []caseReport{valid, bad}}
	return writeJSON(filepath.Join("out", "report.json"), report)
}

func runCase(name, scenePath, overridePath, outDir string) (caseReport, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return caseReport{}, fmt.Errorf("create out dir: %w", err)
	}
	s, err := scene.Load(scenePath)
	if err != nil {
		return caseReport{}, err
	}

	// The base image is the original asset and is rendered exactly once. On
	// rejection every output reuses these bytes, so the original picture is
	// preserved rather than re-rendered.
	baseImg := render.RenderScene(s)
	basePath := filepath.Join(outDir, "base.png")
	baseBytes, err := encodePNG(baseImg)
	if err != nil {
		return caseReport{}, err
	}
	if err := os.WriteFile(basePath, baseBytes, 0o644); err != nil {
		return caseReport{}, err
	}

	rep := caseReport{
		Name: name,
		Images: imageFiles{
			Base:  relPath(basePath),
			Over:  relPath(filepath.Join(outDir, "over.png")),
			Final: relPath(filepath.Join(outDir, "final.png")),
		},
	}

	applied, changes, applyErr := over.Apply(s, overridePath)
	if applyErr != nil {
		// Rejected: no geometry or material changed. over.png and final.png
		// keep the original picture; the "over" layer contributes nothing.
		rep.Accepted = false
		rep.Error = applyErr.Error()
		rep.BasePreserved = true
		if err := os.WriteFile(filepath.Join(outDir, "over.png"), baseBytes, 0o644); err != nil {
			return rep, err
		}
		if err := os.WriteFile(filepath.Join(outDir, "final.png"), baseBytes, 0o644); err != nil {
			return rep, err
		}
		return rep, writeJSON(filepath.Join(outDir, "report.json"), rep)
	}

	targetIDs := changedMeshIDs(changes)
	// Over layer: only the touched meshes, with override materials, on
	// transparency. Meshes are referenced from the applied scene, never
	// redefined or duplicated.
	var touched []*scene.Mesh
	for _, m := range applied.Meshes {
		if targetIDs[m.ID] {
			touched = append(touched, m)
		}
	}
	overImg := render.RenderOver(applied, touched)
	underImg := render.RenderExcept(s, keysOf(targetIDs))
	finalImg := render.Compose(underImg, overImg)

	if err := writePNG(filepath.Join(outDir, "over.png"), overImg); err != nil {
		return rep, err
	}
	if err := writePNG(filepath.Join(outDir, "final.png"), finalImg); err != nil {
		return rep, err
	}

	rep.Accepted = true
	rep.OverriddenProps = buildMeshReports(changes)
	return rep, writeJSON(filepath.Join(outDir, "report.json"), rep)
}

func changedMeshIDs(changes []over.Change) map[string]bool {
	ids := map[string]bool{}
	for _, c := range changes {
		ids[c.MeshID] = true
	}
	return ids
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func buildMeshReports(changes []over.Change) []meshReport {
	order := []string{}
	byMesh := map[string]*meshReport{}
	for _, c := range changes {
		r, ok := byMesh[c.MeshID]
		if !ok {
			r = &meshReport{MeshID: c.MeshID}
			byMesh[c.MeshID] = r
			order = append(order, c.MeshID)
		}
		r.Properties = append(r.Properties, propertyInfo{
			Property: c.Property,
			Path:     c.Path,
			Before:   c.Before,
			After:    c.After,
		})
	}
	out := make([]meshReport, 0, len(order))
	for _, id := range order {
		out = append(out, *byMesh[id])
	}
	return out
}

func printSummary(r caseReport) {
	fmt.Printf("=== case %q ===\n", r.Name)
	fmt.Printf("images: base=%s over=%s final=%s\n", r.Images.Base, r.Images.Over, r.Images.Final)
	if !r.Accepted {
		fmt.Println("result: REJECTED, original picture preserved")
		fmt.Println(r.Error)
		fmt.Println()
		return
	}
	fmt.Println("result: ACCEPTED, overridden property list:")
	for _, m := range r.OverriddenProps {
		for _, p := range m.Properties {
			fmt.Printf("  - mesh %q | %s | %s: %v -> %v\n",
				m.MeshID, p.Property, p.Path, p.Before, p.After)
		}
	}
	fmt.Println()
}

func encodePNG(img *image.NRGBA) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writePNG(path string, img *image.NRGBA) error {
	b, err := encodePNG(img)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeJSON(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func relPath(p string) string {
	if rel, err := filepath.Rel(".", p); err == nil {
		return rel
	}
	return p
}
