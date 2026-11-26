package main

import (
	"os"
	"path/filepath"
	"sort"
)

type layerSource struct {
	ID       string
	FileName string
	Strength int
	Body     string
}

func sourceLayerDocs() []layerSource {
	return []layerSource{
		{ID: "L1-base", FileName: "01_base.usda", Strength: 10, Body: layerL1},
		{ID: "L2-warm", FileName: "02_warm.usda", Strength: 25, Body: layerL2},
		{ID: "L3-cool", FileName: "03_cool.usda", Strength: 30, Body: layerL3},
		{ID: "L4-spot", FileName: "04_spot.usda", Strength: 40, Body: layerL4},
	}
}

func WriteSourceLayers(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	docs := sourceLayerDocs()
	files := make([]string, 0, len(docs))
	for _, doc := range docs {
		name := filepath.Join(dir, doc.FileName)
		if err := os.WriteFile(name, []byte(doc.Body), 0o644); err != nil {
			return nil, err
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}
