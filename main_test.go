package main

import (
	"os"
	"path/filepath"
	"testing"
)

const e2eScene = `{
  "width": 48, "height": 48,
  "background": {"r":1,"g":1,"b":1},
  "meshes": [
    {"id":"m","kind":"rect","geometry":{"x":0,"y":0,"w":48,"h":48},
     "material":{"color":{"r":0,"g":0,"b":0},"opacity":1}}
  ]
}`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunCaseAccepted(t *testing.T) {
	dir := t.TempDir()
	scenePath := filepath.Join(dir, "scene.json")
	overPath := filepath.Join(dir, "over.json")
	outDir := filepath.Join(dir, "out")
	writeFile(t, scenePath, e2eScene)
	writeFile(t, overPath, `{"patches":[
	  {"meshId":"m","path":"meshes.m.material.color","value":{"r":0.1,"g":0.8,"b":0.45}},
	  {"meshId":"m","path":"meshes.m.material.opacity","value":0.4}
	]}`)

	rep, err := runCase("e2e-ok", scenePath, overPath, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Accepted {
		t.Fatalf("expected acceptance: %s", rep.Error)
	}
	if len(rep.OverriddenProps) != 1 || len(rep.OverriddenProps[0].Properties) != 2 {
		t.Fatalf("unexpected property list: %+v", rep.OverriddenProps)
	}
	for _, f := range []string{"base.png", "over.png", "final.png", "report.json"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Fatalf("missing output %s: %v", f, err)
		}
	}
	base, _ := os.ReadFile(filepath.Join(outDir, "base.png"))
	final, _ := os.ReadFile(filepath.Join(outDir, "final.png"))
	if string(base) == string(final) {
		t.Fatal("final image must differ from base when override is applied")
	}
}

func TestRunCaseRejectedPreservesOriginal(t *testing.T) {
	dir := t.TempDir()
	scenePath := filepath.Join(dir, "scene.json")
	overPath := filepath.Join(dir, "bad.json")
	outDir := filepath.Join(dir, "out")
	writeFile(t, scenePath, e2eScene)
	writeFile(t, overPath, `{"patches":[
	  {"meshId":"m","path":"meshes.m.material.opacity","value":0.5},
	  {"meshId":"m","path":"meshes.m.geometry.w","value":10},
	  {"meshId":"ghost","path":"meshes.ghost.material.color","value":{"r":1,"g":0,"b":0}}
	]}`)

	rep, err := runCase("e2e-bad", scenePath, overPath, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Accepted {
		t.Fatal("expected rejection")
	}
	if !rep.BasePreserved {
		t.Fatal("report must record base preservation")
	}
	base, err := os.ReadFile(filepath.Join(outDir, "base.png"))
	if err != nil {
		t.Fatal(err)
	}
	over, _ := os.ReadFile(filepath.Join(outDir, "over.png"))
	final, _ := os.ReadFile(filepath.Join(outDir, "final.png"))
	if string(over) != string(base) || string(final) != string(base) {
		t.Fatal("rejected override must leave every image byte-identical to base")
	}
}
