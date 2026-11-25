package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
)

type Manifest struct {
	Title      string      `json:"title"`
	Canvas     CanvasSpec  `json:"canvas"`
	Patches    []PatchSpec `json:"patches"`
	Groups     []GroupSpec `json:"groups"`
	Thresholds Thresholds  `json:"thresholds"`
	Output     ManifestOut `json:"output"`
}

type CanvasSpec struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ManifestOut struct {
	Amplification int `json:"amplification"`
}

type PatchSpec struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Kind   string   `json:"kind"`
	Rect   RectSpec `json:"rect"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Group  string   `json:"group"`
	Order  int      `json:"order"`
}

type RectSpec struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r RectSpec) imageRect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

func (r RectSpec) valid() bool {
	return r.W > 0 && r.H > 0
}

type GroupSpec struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
}

type Thresholds struct {
	DeltaE              float64 `json:"deltaE"`
	Brightness          float64 `json:"brightness"`
	HighlightDeltaE     float64 `json:"highlightDeltaE"`
	HighlightBrightness float64 `json:"highlightBrightness"`
	SkinDeltaE          float64 `json:"skinDeltaE"`
	SkinBrightness      float64 `json:"skinBrightness"`
	ClipRatio           float64 `json:"clipRatio"`
	HighlightClipRatio  float64 `json:"highlightClipRatio"`
	MonotonicityDeltaL  float64 `json:"monotonicityDeltaL"`
	ClipHigh            int     `json:"clipHigh"`
	ClipLow             int     `json:"clipLow"`
}

func defaultThresholds() Thresholds {
	return Thresholds{
		DeltaE:              2.0,
		Brightness:          5.0,
		HighlightDeltaE:     1.5,
		HighlightBrightness: 2.5,
		SkinDeltaE:          2.5,
		SkinBrightness:      3.0,
		ClipRatio:           0.01,
		HighlightClipRatio:  0.005,
		MonotonicityDeltaL:  0.4,
		ClipHigh:            254,
		ClipLow:             1,
	}
}

func loadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	m.applyDefaults()
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) applyDefaults() {
	if m.Canvas.Width <= 0 || m.Canvas.Height <= 0 {
		m.Canvas.Width = 720
		m.Canvas.Height = 540
	}
	if m.Title == "" {
		m.Title = "PATCH CHECK"
	}
	if m.Output.Amplification <= 0 {
		m.Output.Amplification = 8
	}
	d := defaultThresholds()
	supplied := m.Thresholds
	m.Thresholds = d
	mergeThresholds(&m.Thresholds, supplied)
	for i := range m.Patches {
		p := &m.Patches[i]
		if p.ID == "" {
			p.ID = fmt.Sprintf("P%02d", i+1)
		}
		if p.Label == "" {
			p.Label = p.ID
		}
		if p.Kind == "" {
			p.Kind = "patch"
		}
		p.Kind = strings.ToLower(p.Kind)
	}
}

func mergeThresholds(dst *Thresholds, src Thresholds) {
	if src.DeltaE > 0 {
		dst.DeltaE = src.DeltaE
	}
	if src.Brightness > 0 {
		dst.Brightness = src.Brightness
	}
	if src.HighlightDeltaE > 0 {
		dst.HighlightDeltaE = src.HighlightDeltaE
	}
	if src.HighlightBrightness > 0 {
		dst.HighlightBrightness = src.HighlightBrightness
	}
	if src.SkinDeltaE > 0 {
		dst.SkinDeltaE = src.SkinDeltaE
	}
	if src.SkinBrightness > 0 {
		dst.SkinBrightness = src.SkinBrightness
	}
	if src.ClipRatio > 0 {
		dst.ClipRatio = src.ClipRatio
	}
	if src.HighlightClipRatio > 0 {
		dst.HighlightClipRatio = src.HighlightClipRatio
	}
	if src.MonotonicityDeltaL > 0 {
		dst.MonotonicityDeltaL = src.MonotonicityDeltaL
	}
	if src.ClipHigh > 0 {
		dst.ClipHigh = src.ClipHigh
	}
	if src.ClipLow > 0 {
		dst.ClipLow = src.ClipLow
	}
}

func (m *Manifest) validate() error {
	if len(m.Patches) == 0 {
		return errors.New("manifest must contain at least one patch")
	}
	ids := make(map[string]bool)
	groups := make(map[string]bool)
	for _, g := range m.Groups {
		if g.ID == "" {
			return errors.New("group id must not be empty")
		}
		switch strings.ToLower(g.Direction) {
		case "increasing", "decreasing", "auto", "none":
		default:
			return fmt.Errorf("group %q has invalid direction %q", g.ID, g.Direction)
		}
		groups[g.ID] = true
	}
	for i := range m.Patches {
		p := &m.Patches[i]
		if ids[p.ID] {
			return fmt.Errorf("duplicate patch id %q", p.ID)
		}
		ids[p.ID] = true
		if !p.Rect.valid() {
			return fmt.Errorf("patch %q has an empty rectangle", p.ID)
		}
		if p.Rect.X < 0 || p.Rect.Y < 0 ||
			p.Rect.X+p.Rect.W > m.Canvas.Width ||
			p.Rect.Y+p.Rect.H > m.Canvas.Height {
			return fmt.Errorf("patch %q is outside the %dx%d canvas", p.ID, m.Canvas.Width, m.Canvas.Height)
		}
		if p.Group != "" && !groups[p.Group] {
			return fmt.Errorf("patch %q references unknown group %q", p.ID, p.Group)
		}
		if p.Source != "" {
			if _, _, _, err := parseHexColor(p.Source); err != nil {
				return fmt.Errorf("patch %q source: %w", p.ID, err)
			}
		}
		if p.Target != "" {
			if _, _, _, err := parseHexColor(p.Target); err != nil {
				return fmt.Errorf("patch %q target: %w", p.ID, err)
			}
		}
	}
	return nil
}

func parseHexColor(value string) (uint8, uint8, uint8, error) {
	s := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(s) == 3 {
		s = strings.Repeat(string(s[0]), 2) + strings.Repeat(string(s[1]), 2) + strings.Repeat(string(s[2]), 2)
	}
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid color %q (want #RRGGBB)", value)
	}
	r, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid color %q", value)
	}
	g, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid color %q", value)
	}
	b, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid color %q", value)
	}
	return uint8(r), uint8(g), uint8(b), nil
}
