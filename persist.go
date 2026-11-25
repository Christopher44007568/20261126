package main

import (
	"encoding/json"
	"os"
)

// SaveFlatScene serializes a flattened scene as JSON (stdlib only).
func SaveFlatScene(fs *FlatScene, path string) error {
	data, err := json.MarshalIndent(fs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFlatScene restores a flattened scene from JSON. The result is
// fully self-contained: no hierarchy, transforms or overrides remain.
func LoadFlatScene(path string) (*FlatScene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fs FlatScene
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil, err
	}
	return &fs, nil
}
