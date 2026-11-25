package main

import "image/color"

// Material describes how a shape is painted. Fill/Stroke may be nil
// (meaning "not painted"). StrokeWidth is in local units.
type Material struct {
	Fill        color.NRGBA
	HasFill     bool
	Stroke      color.NRGBA
	HasStroke   bool
	StrokeWidth float64
}

// MaterialOverride selectively replaces material fields of every
// descendant shape. A nil field means "do not override this field".
// Deeper overrides win over shallower ones, field by field.
type MaterialOverride struct {
	Fill        *color.NRGBA
	Stroke      *color.NRGBA
	StrokeWidth *float64
}

func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

func rgbaPtr(c color.NRGBA) *color.NRGBA { return &c }

func f64Ptr(v float64) *float64 { return &v }

// ShapeKind enumerates the supported primitives.
type ShapeKind int

const (
	KindRect ShapeKind = iota
	KindEllipse
	KindPolygon
)

// Node is a scene-graph node. A node is either a shape (Kind set) or a
// group/layer (Children non-empty). Groups may carry a transform, a
// hidden flag, an opacity multiplier and a material override.
type Node struct {
	Name      string
	Hidden    bool
	Opacity   float64 // multiplies alpha of the whole subtree; 0 means 1
	Transform Matrix

	// Shape fields (leaf nodes).
	Kind     ShapeKind
	W, H     float64 // rect / ellipse bounding box, centered at origin
	Points   []Point // polygon vertices (local space)
	Material Material

	// Group fields.
	Override *MaterialOverride
	Children []*Node
}

func (n *Node) isGroup() bool { return len(n.Children) > 0 }

func (n *Node) opacity() float64 {
	if n.Opacity <= 0 {
		return 1
	}
	return n.Opacity
}

// Layer is a top-level compositing unit. Layers render in order.
type Layer struct {
	Name    string
	Hidden  bool
	Opacity float64 // 0 means 1
	Root    *Node
}

func (l *Layer) opacity() float64 {
	if l.Opacity <= 0 {
		return 1
	}
	return l.Opacity
}

// Scene is an ordered stack of layers over an opaque background.
type Scene struct {
	Width, Height int
	Background    color.NRGBA
	Layers        []*Layer
}

// FlatShape is a flattened, render-ready primitive: world-space
// geometry, fully resolved material, preserved hidden flag.
type FlatShape struct {
	Name     string
	Hidden   bool
	Kind     ShapeKind
	Points   []Point // world-space outline (rect/ellipse are polygonized)
	Material Material
}

// FlatScene is the flattened form of a Scene: a single ordered list of
// world-space shapes with no hierarchy, transforms or overrides left.
type FlatScene struct {
	Width, Height int
	Background    color.NRGBA
	Shapes        []FlatShape
}
