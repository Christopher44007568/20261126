// Package report writes the machine-readable migration report, the red-box
// manifest and the human-readable Markdown audit.
package report

import (
	"encoding/json"
	"os"

	"nsmigrate/viz"
)

// ImageBoxes groups the red boxes of one rendered image.
type ImageBoxes struct {
	Image  string    `json:"image"`
	Width  int       `json:"width"`
	Height int       `json:"height"`
	Boxes  []viz.Box `json:"boxes"`
}

// RedBoxesManifest is written to red_boxes.json.
type RedBoxesManifest struct {
	BoxColor [3]int       `json:"box_color"`
	PpmNote  string       `json:"ppm"`
	Images   []ImageBoxes `json:"images"`
}

// WriteJSON writes indented JSON to path.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// WriteFile writes arbitrary text bytes.
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// NewRedBoxesManifest builds the manifest with the pure-red box color.
func NewRedBoxesManifest(images []ImageBoxes) RedBoxesManifest {
	return RedBoxesManifest{
		BoxColor: [3]int{255, 0, 0},
		PpmNote:  "Images are binary P6 PPM; .png previews are provided for convenience.",
		Images:   images,
	}
}
