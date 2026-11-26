// Package over implements material override patches.
//
// An override never redefines geometry or introduces new meshes: it only
// targets material properties of meshes that already exist in the original
// asset. Patches are validated as a whole; any single invalid entry rejects
// the whole patch and leaves the original asset untouched.
package over

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"materialover/scene"
)

// Property types that an override is allowed to touch.
const (
	PropColor   = "color"
	PropOpacity = "opacity"
)

// Change is one accepted override entry, recorded for the audit list.
type Change struct {
	MeshID   string      `json:"meshId"`
	Path     string      `json:"path"`
	Property string      `json:"property"`
	Before   interface{} `json:"before"`
	After    interface{} `json:"after"`
}

// rawPatch is the on-disk override format. Value is kept as raw JSON so each
// property type can enforce its own value shape strictly.
type rawPatch struct {
	MeshID string          `json:"meshId"`
	Path   string          `json:"path"`
	Value  json.RawMessage `json:"value"`
}

type rawFile struct {
	Patches []rawPatch `json:"patches"`
}

type colorValue struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

type planEntry struct {
	patch  rawPatch
	mesh   *scene.Mesh
	prop   string
	color  scene.RGB
	opac   float64
	change Change
}

// Apply loads an override file and applies it to a deep copy of the scene.
// On success the original scene is untouched and the returned scene carries
// the overridden materials; on failure no mesh is mutated and the returned
// change list is nil.
func Apply(s *scene.Scene, path string) (*scene.Scene, []Change, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read override: %w", err)
	}
	var rf rawFile
	if err := strictUnmarshal(raw, &rf); err != nil {
		return nil, nil, fmt.Errorf("parse override: %w", err)
	}
	if len(rf.Patches) == 0 {
		return nil, nil, fmt.Errorf("override: no patches present")
	}

	// Validate every entry before mutating anything: application is atomic.
	// All problems are collected so the rejection report explains every entry
	// that was refused, but no entry is applied when any one is invalid.
	plans := make([]planEntry, 0, len(rf.Patches))
	var problems []string
	for i, p := range rf.Patches {
		pe, err := planEntryFor(s, p)
		if err != nil {
			problems = append(problems, fmt.Sprintf("patch %d rejected: %v", i, err))
			continue
		}
		plans = append(plans, pe)
	}
	if len(problems) > 0 {
		return nil, nil, fmt.Errorf("override rejected atomically (%d problem(s)):\n  - %s",
			len(problems), joinStrings(problems, "\n  - "))
	}

	out := s.Clone()
	changes := make([]Change, 0, len(plans))
	for _, pe := range plans {
		mesh := out.Mesh(pe.mesh.ID)
		c := pe.change
		switch pe.prop {
		case PropColor:
			c.Before = mesh.Material.Color
			c.After = pe.color
			mesh.Material.Color = pe.color
		case PropOpacity:
			c.Before = mesh.Material.Opacity
			c.After = pe.opac
			mesh.Material.Opacity = pe.opac
		}
		changes = append(changes, c)
	}
	return out, changes, nil
}

func planEntryFor(s *scene.Scene, p rawPatch) (pe planEntry, err error) {
	if p.MeshID == "" {
		return pe, fmt.Errorf("meshId is required")
	}
	mesh := s.Mesh(p.MeshID)
	if mesh == nil {
		return pe, fmt.Errorf("target path %q: mesh %q does not exist in the original asset", p.Path, p.MeshID)
	}
	prop, ok := parseMaterialPath(p.Path)
	if !ok {
		return pe, fmt.Errorf("target path %q: only material property overrides are allowed (allowed: meshes.<id>.material.color, meshes.<id>.material.opacity)", p.Path)
	}
	pe.patch = p
	pe.mesh = mesh
	pe.prop = prop
	pe.change = Change{MeshID: p.MeshID, Path: p.Path, Property: prop}

	switch prop {
	case PropColor:
		var cv colorValue
		if err := strictUnmarshal(p.Value, &cv); err != nil {
			return pe, fmt.Errorf("property %q requires an object {r,g,b}: %w", prop, err)
		}
		c := scene.RGB{R: cv.R, G: cv.G, B: cv.B}
		for name, v := range map[string]float64{"r": c.R, "g": c.G, "b": c.B} {
			if v < 0 || v > 1 {
				return pe, fmt.Errorf("property %q: channel %s must be in [0,1], got %v", prop, name, v)
			}
		}
		pe.color = c
	case PropOpacity:
		var op float64
		if err := strictUnmarshal(p.Value, &op); err != nil {
			return pe, fmt.Errorf("property %q requires a number: %w", prop, err)
		}
		if op < 0 || op > 1 {
			return pe, fmt.Errorf("property %q must be in [0,1], got %v", prop, op)
		}
		pe.opac = op
	}
	return pe, nil
}

// parseMaterialPath accepts exactly:
//
//	meshes.<id>.material.color
//	meshes.<id>.material.opacity
//
// The id segment is informational here; existence is checked separately.
func parseMaterialPath(path string) (string, bool) {
	parts := splitPath(path)
	if len(parts) != 4 || parts[0] != "meshes" || parts[1] == "" || parts[2] != "material" {
		return "", false
	}
	switch parts[3] {
	case PropColor, PropOpacity:
		return parts[3], true
	default:
		return "", false
	}
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			if i > start {
				parts = append(parts, path[start:i])
			} else if i < len(path) {
				// empty segment => invalid path
				return []string{"__invalid__"}
			}
			start = i + 1
		}
	}
	return parts
}

// strictUnmarshal rejects unknown fields and duplicate keys so an override
// cannot smuggle geometry or typo'd property names past validation.
func strictUnmarshal(data []byte, v interface{}) error {
	if err := checkDuplicateKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err == nil {
		return fmt.Errorf("unexpected trailing JSON value")
	}
	return nil
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// checkDuplicateKeys walks the JSON token stream and rejects repeated keys
// at any object level (encoding/json silently keeps the last duplicate).
func checkDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	type frame struct {
		keys      map[string]bool
		expectKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				stack = append(stack, &frame{keys: map[string]bool{}, expectKey: true})
			case '[':
				stack = append(stack, nil)
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1] != nil {
					stack[len(stack)-1].expectKey = true
				}
			}
		case string:
			top := len(stack) - 1
			if top >= 0 && stack[top] != nil && stack[top].expectKey {
				if stack[top].keys[t] {
					return fmt.Errorf("duplicate JSON key %q", t)
				}
				stack[top].keys[t] = true
				stack[top].expectKey = false
			} else if top >= 0 && stack[top] != nil {
				stack[top].expectKey = true
			}
		default:
			top := len(stack) - 1
			if top >= 0 && stack[top] != nil {
				stack[top].expectKey = true
			}
		}
	}
	return nil
}
