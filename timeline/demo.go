package timeline

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed demo_scene.json
var demoSceneJSON []byte

// DemoScene loads the fixed irregular sample scene. The data is embedded so
// the compiled program can run from any working directory.
func DemoScene() (*Scene, error) {
	var scene Scene
	if err := json.Unmarshal(demoSceneJSON, &scene); err != nil {
		return nil, fmt.Errorf("timeline: decode embedded demo: %w", err)
	}
	if err := scene.Prepare(); err != nil {
		return nil, err
	}
	return &scene, nil
}
