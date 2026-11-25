package workflow

import (
	"bytes"
	"fmt"

	"texturelab/material"
	"texturelab/texcontainer"
)

// RoundtripReport proves that writing and reloading caused no material shift.
type RoundtripReport struct {
	NoColorShift bool            `json:"no_color_shift"`
	MaterialName string          `json:"material_name"`
	Textures     []TextureReport `json:"textures"`
}

// TextureReport contains marker and per-level fingerprint comparisons.
type TextureReport struct {
	Role                        string           `json:"role"`
	Path                        string           `json:"path"`
	SpaceBefore                 string           `json:"space_before"`
	SpaceAfter                  string           `json:"space_after"`
	MarkerPreserved             bool             `json:"marker_preserved"`
	MetadataPreserved           bool             `json:"metadata_preserved"`
	PayloadIdentical            bool             `json:"payload_identical"`
	LinearFingerprintsIdentical bool             `json:"linear_fingerprints_identical"`
	Levels                      []LevelRoundtrip `json:"levels"`
}

// LevelRoundtrip stores before/after hashes for one mip level.
type LevelRoundtrip struct {
	Level               int    `json:"level"`
	Width               int    `json:"width"`
	Height              int    `json:"height"`
	PayloadSHA256Before string `json:"payload_sha256_before"`
	PayloadSHA256After  string `json:"payload_sha256_after"`
	LinearSHA256Before  string `json:"linear_sha256_before"`
	LinearSHA256After   string `json:"linear_sha256_after"`
	PayloadMatch        bool   `json:"payload_match"`
	LinearMatch         bool   `json:"linear_match"`
}

func verifyRoundtrip(paths Files, albedoBefore, albedoAfter, normalBefore, normalAfter *texcontainer.Texture) (*RoundtripReport, error) {
	albedoReport, err := compareForRoundtrip("albedo", paths.AlbedoContainer, albedoBefore, albedoAfter)
	if err != nil {
		return nil, err
	}
	normalReport, err := compareForRoundtrip("normal", paths.NormalContainer, normalBefore, normalAfter)
	if err != nil {
		return nil, err
	}
	report := &RoundtripReport{
		MaterialName: "reloaded-procedural-material",
		Textures:     []TextureReport{albedoReport, normalReport},
	}
	report.NoColorShift = albedoReport.perfect() && normalReport.perfect()
	if !report.NoColorShift {
		return nil, fmt.Errorf("workflow: container round trip changed material interpretation")
	}
	return report, nil
}

func (r TextureReport) perfect() bool {
	if !r.MarkerPreserved || !r.MetadataPreserved || !r.PayloadIdentical || !r.LinearFingerprintsIdentical {
		return false
	}
	for _, level := range r.Levels {
		if !level.PayloadMatch || !level.LinearMatch {
			return false
		}
	}
	return true
}

func compareForRoundtrip(role, path string, before, after *texcontainer.Texture) (TextureReport, error) {
	if before == nil || after == nil {
		return TextureReport{}, fmt.Errorf("workflow: nil %s texture in round trip", role)
	}
	if len(before.Levels) != len(after.Levels) {
		return TextureReport{}, fmt.Errorf("workflow: %s level count changed", role)
	}
	report := TextureReport{
		Role:              role,
		Path:              path,
		SpaceBefore:       before.Space.String(),
		SpaceAfter:        after.Space.String(),
		MarkerPreserved:   before.Space == after.Space,
		MetadataPreserved: mapsEqual(before.Metadata, after.Metadata),
		Levels:            make([]LevelRoundtrip, 0, len(before.Levels)),
	}
	report.PayloadIdentical = true
	report.LinearFingerprintsIdentical = true

	for i := range before.Levels {
		beforeLevel := before.Levels[i]
		afterLevel := after.Levels[i]
		payloadMatch := bytes.Equal(beforeLevel.Pixels, afterLevel.Pixels) &&
			beforeLevel.Width == afterLevel.Width && beforeLevel.Height == afterLevel.Height
		beforeLinear, err := material.LinearFingerprint(before, i)
		if err != nil {
			return TextureReport{}, err
		}
		afterLinear, err := material.LinearFingerprint(after, i)
		if err != nil {
			return TextureReport{}, err
		}
		levelReport := LevelRoundtrip{
			Level:               i,
			Width:               afterLevel.Width,
			Height:              afterLevel.Height,
			PayloadSHA256Before: hashBytes(beforeLevel.Pixels),
			PayloadSHA256After:  hashBytes(afterLevel.Pixels),
			LinearSHA256Before:  beforeLinear,
			LinearSHA256After:   afterLinear,
			PayloadMatch:        payloadMatch,
			LinearMatch:         beforeLinear == afterLinear,
		}
		report.Levels = append(report.Levels, levelReport)
		report.PayloadIdentical = report.PayloadIdentical && payloadMatch
		report.LinearFingerprintsIdentical = report.LinearFingerprintsIdentical && levelReport.LinearMatch
	}
	return report, nil
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}
