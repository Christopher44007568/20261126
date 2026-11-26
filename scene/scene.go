// Package scene defines the original renderable asset: a small 2D scene made
// of meshes. Each mesh owns geometry and one material. Nothing in this package
// knows about overrides or rendering.
package scene

import (
	"encoding/json"
	"fmt"
	"os"
)

// RGB is a straight (non-premultiplied) color with channels in [0, 1].
type RGB struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

// Material is the only part of a mesh that an "over" patch may touch.
// Color and opacity are separate overridable properties.
type Material struct {
	Color   RGB     `json:"color"`
	Opacity float64 `json:"opacity"`
}

// Geometry holds every supported shape's fields; only the fields matching the
// mesh Kind are meaningful.
type Geometry struct {
	// rect
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`
	W float64 `json:"w,omitempty"`
	H float64 `json:"h,omitempty"`
	// circle
	CX     float64 `json:"cx,omitempty"`
	CY     float64 `json:"cy,omitempty"`
	Radius float64 `json:"radius,omitempty"`
	// polygon
	Points [][2]float64 `json:"points,omitempty"`
}

// Mesh is a drawable node: geometry plus a material.
type Mesh struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Geometry Geometry `json:"geometry"`
	Material Material `json:"material"`
}

// Scene is the full original asset.
type Scene struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Background *RGB    `json:"background"`
	Meshes     []*Mesh `json:"meshes"`
}

// Load reads and validates a scene JSON file.
func Load(path string) (*Scene, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scene: %w", err)
	}
	var s Scene
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parse scene: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks structural invariants of the original asset.
func (s *Scene) Validate() error {
	if s.Width <= 0 || s.Height <= 0 {
		return fmt.Errorf("scene: width and height must be positive")
	}
	if s.Background != nil {
		if err := checkRGB("background", *s.Background); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	for i, m := range s.Meshes {
		if m == nil {
			return fmt.Errorf("scene: mesh %d is null", i)
		}
		if m.ID == "" {
			return fmt.Errorf("scene: mesh %d has empty id", i)
		}
		if seen[m.ID] {
			return fmt.Errorf("scene: duplicate mesh id %q", m.ID)
		}
		seen[m.ID] = true
		if err := m.validate(); err != nil {
			return fmt.Errorf("scene: mesh %q: %w", m.ID, err)
		}
	}
	return nil
}

func (m *Mesh) validate() error {
	g := m.Geometry
	switch m.Kind {
	case "rect":
		if g.W <= 0 || g.H <= 0 {
			return fmt.Errorf("rect requires positive w and h")
		}
	case "circle":
		if g.Radius <= 0 {
			return fmt.Errorf("circle requires positive radius")
		}
	case "polygon":
		if len(g.Points) < 3 {
			return fmt.Errorf("polygon requires at least 3 points")
		}
	default:
		return fmt.Errorf("unknown geometry kind %q", m.Kind)
	}
	if err := checkRGB("material.color", m.Material.Color); err != nil {
		return err
	}
	if m.Material.Opacity < 0 || m.Material.Opacity > 1 {
		return fmt.Errorf("material.opacity must be in [0,1], got %v", m.Material.Opacity)
	}
	return nil
}

func checkRGB(where string, c RGB) error {
	for name, v := range map[string]float64{"r": c.R, "g": c.G, "b": c.B} {
		if v < 0 || v > 1 {
			return fmt.Errorf("%s.%s must be in [0,1], got %v", where, name, v)
		}
	}
	return nil
}

// Mesh looks up a mesh by id.
func (s *Scene) Mesh(id string) *Mesh {
	for _, m := range s.Meshes {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// Clone returns a deep copy so override application never mutates the
// original asset.
func (s *Scene) Clone() *Scene {
	cp := *s
	cp.Meshes = make([]*Mesh, len(s.Meshes))
	for i, m := range s.Meshes {
		mc := *m
		if len(m.Geometry.Points) > 0 {
			mc.Geometry.Points = append([][2]float64(nil), m.Geometry.Points...)
		}
		cp.Meshes[i] = &mc
	}
	if s.Background != nil {
		bg := *s.Background
		cp.Background = &bg
	}
	return &cp
}
