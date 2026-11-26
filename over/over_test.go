package over

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"materialover/scene"
)

func testScene() *scene.Scene {
	bg := scene.RGB{R: 0.9, G: 0.9, B: 0.9}
	return &scene.Scene{
		Width:      32,
		Height:     32,
		Background: &bg,
		Meshes: []*scene.Mesh{
			{
				ID:       "a",
				Kind:     "rect",
				Geometry: scene.Geometry{X: 0, Y: 0, W: 10, H: 10},
				Material: scene.Material{
					Color:   scene.RGB{R: 1, G: 0, B: 0},
					Opacity: 1,
				},
			},
			{
				ID:       "b",
				Kind:     "circle",
				Geometry: scene.Geometry{CX: 16, CY: 16, Radius: 8},
				Material: scene.Material{
					Color:   scene.RGB{R: 0, G: 1, B: 0},
					Opacity: 0.5,
				},
			},
		},
	}
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplyValidOverride(t *testing.T) {
	s := testScene()
	p := writeTemp(t, "o.json", `{
	  "patches": [
	    {"meshId":"a","path":"meshes.a.material.color","value":{"r":0,"g":0,"b":1}},
	    {"meshId":"b","path":"meshes.b.material.opacity","value":0.25}
	  ]
	}`)
	out, changes, err := Apply(s, p)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if got := out.Mesh("a").Material.Color; got != (scene.RGB{R: 0, G: 0, B: 1}) {
		t.Fatalf("color override wrong: %+v", got)
	}
	if got := out.Mesh("b").Material.Opacity; got != 0.25 {
		t.Fatalf("opacity override wrong: %v", got)
	}
	if len(changes) != 2 {
		t.Fatalf("want 2 changes, got %d", len(changes))
	}
	// Original asset stays untouched.
	if got := s.Mesh("a").Material.Color; got != (scene.RGB{R: 1, G: 0, B: 0}) {
		t.Fatalf("original scene mutated: %+v", got)
	}
}

func TestApplyRejections(t *testing.T) {
	s := testScene()
	cases := map[string]string{
		"unknown mesh":         `{"patches":[{"meshId":"nope","path":"meshes.nope.material.color","value":{"r":1,"g":1,"b":1}}]}`,
		"geometry path":        `{"patches":[{"meshId":"a","path":"meshes.a.geometry.x","value":5}]}`,
		"root material path":   `{"patches":[{"meshId":"a","path":"material.color","value":{"r":1,"g":1,"b":1}}]}`,
		"unknown property":     `{"patches":[{"meshId":"a","path":"meshes.a.material.shininess","value":1}]}`,
		"bad color channel":    `{"patches":[{"meshId":"a","path":"meshes.a.material.color","value":{"r":2,"g":0,"b":0}}]}`,
		"opacity out of range": `{"patches":[{"meshId":"b","path":"meshes.b.material.opacity","value":-0.1}]}`,
		"opacity wrong type":   `{"patches":[{"meshId":"b","path":"meshes.b.material.opacity","value":"half"}]}`,
		"color wrong shape":    `{"patches":[{"meshId":"a","path":"meshes.a.material.color","value":[1,0,0]}]}`,
		"unknown json field":   `{"patches":[{"meshId":"a","path":"meshes.a.material.color","value":{"r":1,"g":0,"b":0},"extra":1}]}`,
		"duplicate json keys":  `{"patches":[{"meshId":"a","path":"meshes.a.material.color","value":{"r":1,"r":0,"b":0,"g":0}}]}`,
		"empty segment":        `{"patches":[{"meshId":"a","path":"meshes..material.color","value":{"r":1,"g":0,"b":0}}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			p := writeTemp(t, "o.json", body)
			before := s.Clone()
			out, changes, err := Apply(s, p)
			if err == nil {
				t.Fatalf("expected rejection, got success")
			}
			if out != nil || changes != nil {
				t.Fatalf("rejected patch must return nil outputs")
			}
			if !reflect.DeepEqual(before, s) {
				t.Fatalf("rejected patch mutated the original scene")
			}
		})
	}
}

func TestApplyIsAtomic(t *testing.T) {
	s := testScene()
	// First entry is valid, second is invalid: nothing may be applied.
	body := `{"patches":[
	  {"meshId":"a","path":"meshes.a.material.color","value":{"r":0,"g":0,"b":1}},
	  {"meshId":"ghost","path":"meshes.ghost.material.opacity","value":0.1}
	]}`
	p := writeTemp(t, "o.json", body)
	before := s.Clone()
	if _, _, err := Apply(s, p); err == nil {
		t.Fatal("expected atomic rejection")
	}
	if !reflect.DeepEqual(before, s) {
		t.Fatal("scene mutated despite atomic rejection")
	}
}

func TestParseMaterialPath(t *testing.T) {
	if prop, ok := parseMaterialPath("meshes.a.material.color"); !ok || prop != PropColor {
		t.Fatal("valid color path rejected")
	}
	if _, ok := parseMaterialPath("meshes.a.geometry.x"); ok {
		t.Fatal("geometry path accepted")
	}
	if _, ok := parseMaterialPath("meshes.a.material"); ok {
		t.Fatal("short path accepted")
	}
}
