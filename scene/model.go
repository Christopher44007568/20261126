// Package scene defines the small USD-like namespace scene used by the
// migration audit: hierarchical prim paths plus reference arcs, material
// bindings and camera targets.
package scene

// RGB is an 8-bit color.
type RGB struct{ R, G, B uint8 }

// Hex builds a color from 0xRRGGBB.
func Hex(v uint32) RGB {
	return RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}
}

// ShapeKind selects the top-down glyph used by the renderer.
type ShapeKind string

const (
	ShapeGroup     ShapeKind = "group"
	ShapeBox       ShapeKind = "box"
	ShapeSphere    ShapeKind = "sphere"
	ShapeCamera    ShapeKind = "camera"
	ShapeTarget    ShapeKind = "target"
	ShapeComponent ShapeKind = "component"
)

// Expect states whether a link is meant to survive migration or is one of the
// deliberately tracked edge cases.
type Expect string

const (
	// ExpectResolves must resolve both before and after correct migration.
	ExpectResolves Expect = "resolves"
	// ExpectBroken targets a prim that does not exist (a genuine dangling arc).
	ExpectBroken Expect = "broken"
	// ExpectDefault is an unresolved material binding that keeps a default.
	ExpectDefault Expect = "default"
)

// LinkKind enumerates the three relationship classes the audit protects.
type LinkKind string

const (
	KindReference       LinkKind = "reference"
	KindMaterialBinding LinkKind = "material_binding"
	KindCameraTarget    LinkKind = "camera_target"
)

// Mapping is the old-namespace -> new-path re-rooting.
type Mapping struct {
	OldRoot string `json:"old_root"`
	NewRoot string `json:"new_root"`
}

// Link is one reference / material binding / camera target arc.
type Link struct {
	ID        string   `json:"id"`
	Kind      LinkKind `json:"kind"`
	RelName   string   `json:"rel"`
	OwnerID   string   `json:"owner_id"`
	Target    string   `json:"target"`
	OldTarget string   `json:"old_target"`
	Asset     string   `json:"asset,omitempty"`
	Expect    Expect   `json:"expect"`
}

// Prim is one node in the namespace hierarchy.
type Prim struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	Type     string    `json:"type"`
	Shape    ShapeKind `json:"shape"`
	ParentID string    `json:"parent_id,omitempty"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	W        float64   `json:"w"`
	H        float64   `json:"h"`
	Color    RGB       `json:"color"`
	Tag      string    `json:"tag"`
}

// Stage is the whole scene.
type Stage struct {
	Name  string  `json:"name"`
	Prims []*Prim `json:"prims"`
	Links []*Link `json:"links"`
}

// PrimByID returns the prim with the stable id.
func (s *Stage) PrimByID(id string) *Prim {
	for _, p := range s.Prims {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// PrimByPath returns the prim whose current absolute path equals path.
func (s *Stage) PrimByPath(path string) *Prim {
	for _, p := range s.Prims {
		if p.Path == path {
			return p
		}
	}
	return nil
}

// LinkByID returns a link by its stable id.
func (s *Stage) LinkByID(id string) *Link {
	for _, l := range s.Links {
		if l.ID == id {
			return l
		}
	}
	return nil
}
