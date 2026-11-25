package rayao

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

const (
	DepthPNGFilename     = "depth.png"
	AnyHitShadowFilename = "anyhit_shadow.png"
	CompositePNGFilename = "composite.png"
	ManifestFilename     = "manifest.json"
)

// Manifest binds the three image files to one rendered scene version.
type Manifest struct {
	SceneName      string            `json:"scene_name"`
	SceneVersion   string            `json:"scene_version"`
	SceneHash      string            `json:"scene_hash"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	DepthNear      float64           `json:"depth_near"`
	DepthFar       float64           `json:"depth_far"`
	DepthEncoding  string            `json:"depth_encoding"`
	ShadowEncoding string            `json:"shadow_encoding"`
	Files          map[string]string `json:"files"`
	SHA256         map[string]string `json:"sha256"`
	Stats          Stats             `json:"ray_stats"`
}

// WriteOutputs writes depth, first-any-hit shadow/AO, composite, and manifest.
func WriteOutputs(frame *Frame, directory string) (*Manifest, error) {
	if frame == nil {
		return nil, os.ErrInvalid
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}

	images := map[string]image.Image{
		DepthPNGFilename:     frame.Depth,
		AnyHitShadowFilename: frame.AnyHitShadow,
		CompositePNGFilename: frame.Composite,
	}
	hashes := make(map[string]string, len(images))
	for name, img := range images {
		var buf bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(buf.Bytes())
		hashes[name] = hex.EncodeToString(sum[:])
		if err := os.WriteFile(filepath.Join(directory, name), buf.Bytes(), 0o644); err != nil {
			return nil, err
		}
	}

	manifest := Manifest{
		SceneName:      frame.SceneName,
		SceneVersion:   frame.SceneVersion,
		SceneHash:      frame.SceneHash,
		Width:          frame.Width,
		Height:         frame.Height,
		DepthNear:      frame.DepthNear,
		DepthFar:       frame.DepthFar,
		DepthEncoding:  "uint8 grayscale; white=near, black=far-or-miss; metric normalization uses depth_near/depth_far",
		ShadowEncoding: "uint8 grayscale; product of key-light any-hit visibility and deterministic hemisphere AO; white=visible, black=blocked",
		Files: map[string]string{
			"depth":     DepthPNGFilename,
			"anyhit":    AnyHitShadowFilename,
			"composite": CompositePNGFilename,
		},
		SHA256: hashes,
		Stats:  frame.Stats,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(directory, ManifestFilename), data, 0o644); err != nil {
		return nil, err
	}
	return &manifest, nil
}
