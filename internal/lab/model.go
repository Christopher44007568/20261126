// Package lab implements a prototype/instance document model where hundreds
// of instances reference a single prototype by stable ID.
//
// Instances never own a copy of the prototype node tree and never write back
// into it. Per-instance color and visibility differences live in dedicated
// maps on the instance itself, so editing one instance cannot mutate the
// shared prototype or any sibling instance.
package lab

import (
	"errors"
	"fmt"
)

// Rect is an axis-aligned bounding box in document coordinates.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Node is a single renderable piece of a prototype.
type Node struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Fill     string `json:"fill"`
	Geometry Rect   `json:"geometry"`
}

// Prototype is the shared master document. It is referenced by instances via
// PrototypeID; the instances hold no pointer to this struct.
type Prototype struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Nodes []*Node `json:"nodes"`
}

// Node looks up a prototype node by ID.
func (p *Prototype) Node(nodeID string) (*Node, error) {
	for _, node := range p.Nodes {
		if node.ID == nodeID {
			return node, nil
		}
	}
	return nil, fmt.Errorf("lab: prototype %q has no node %q", p.ID, nodeID)
}

// Bounds returns the union bounding box of every prototype node.
func (p *Prototype) Bounds() Rect {
	return unionRects(p.Nodes)
}

// Instance references a prototype by ID and stores local overrides only.
//
// ColorOverrides maps nodeID -> fill color. HiddenNodes maps nodeID -> true
// for per-node visibility changes. Both maps are owned exclusively by this
// instance and are never shared with another instance.
type Instance struct {
	ID             string            `json:"id"`
	PrototypeID    string            `json:"prototype_id"`
	OffsetX        float64           `json:"offset_x"`
	OffsetY        float64           `json:"offset_y"`
	Hidden         bool              `json:"hidden"`
	ColorOverrides map[string]string `json:"color_overrides"`
	HiddenNodes    map[string]bool   `json:"hidden_nodes"`
}

// EffectiveColor resolves a node fill: local override wins, otherwise the
// prototype default. The prototype is never modified.
func (in *Instance) EffectiveColor(p *Prototype, nodeID string) (string, error) {
	if _, err := p.Node(nodeID); err != nil {
		return "", err
	}
	if color, ok := in.ColorOverrides[nodeID]; ok {
		return color, nil
	}
	node, _ := p.Node(nodeID)
	return node.Fill, nil
}

// IsNodeVisible reports whether a node is visible through this instance.
func (in *Instance) IsNodeVisible(nodeID string) bool {
	return !in.Hidden && !in.HiddenNodes[nodeID]
}

// SetColor installs a per-instance color override. It only writes to the
// instance's own override map, which is lazily allocated per instance.
func (in *Instance) SetColor(nodeID, color string) {
	if in.ColorOverrides == nil {
		in.ColorOverrides = make(map[string]string)
	}
	in.ColorOverrides[nodeID] = color
}

// ResetColor removes a local override, restoring the prototype default.
func (in *Instance) ResetColor(nodeID string) {
	delete(in.ColorOverrides, nodeID)
}

// SetNodeVisible toggles per-node visibility on this instance only.
func (in *Instance) SetNodeVisible(nodeID string, visible bool) {
	if in.HiddenNodes == nil {
		in.HiddenNodes = make(map[string]bool)
	}
	if visible {
		delete(in.HiddenNodes, nodeID)
		return
	}
	in.HiddenNodes[nodeID] = true
}

var (
	// ErrPrototypeNotFound is returned when a referenced prototype is absent.
	ErrPrototypeNotFound = errors.New("lab: prototype not found")
	// ErrInstanceNotFound is returned when an instance ID does not exist.
	ErrInstanceNotFound = errors.New("lab: instance not found")
)

// Document is the registry of prototypes and instances.
type Document struct {
	prototypes map[string]*Prototype
	instances  map[string]*Instance
	order      []string
	meta       ScenarioMeta
}

// NewDocument creates an empty document.
func NewDocument() *Document {
	return &Document{
		prototypes: make(map[string]*Prototype),
		instances:  make(map[string]*Instance),
	}
}

// AddPrototype registers a prototype.
func (d *Document) AddPrototype(p *Prototype) error {
	if p == nil || p.ID == "" {
		return errors.New("lab: prototype requires an ID")
	}
	d.prototypes[p.ID] = p
	return nil
}

// AddInstance registers an instance, validating that its prototype exists.
func (d *Document) AddInstance(in *Instance) error {
	if in == nil || in.ID == "" {
		return errors.New("lab: instance requires an ID")
	}
	if _, ok := d.prototypes[in.PrototypeID]; !ok {
		return fmt.Errorf("%w: %q", ErrPrototypeNotFound, in.PrototypeID)
	}
	if in.ColorOverrides == nil {
		in.ColorOverrides = make(map[string]string)
	}
	if in.HiddenNodes == nil {
		in.HiddenNodes = make(map[string]bool)
	}
	if _, exists := d.instances[in.ID]; !exists {
		d.order = append(d.order, in.ID)
	}
	d.instances[in.ID] = in
	return nil
}

// Prototype returns a registered prototype.
func (d *Document) Prototype(id string) (*Prototype, error) {
	p, ok := d.prototypes[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrPrototypeNotFound, id)
	}
	return p, nil
}

// Instance returns a registered instance.
func (d *Document) Instance(id string) (*Instance, error) {
	in, ok := d.instances[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInstanceNotFound, id)
	}
	return in, nil
}

// Instances returns all instances in insertion order.
func (d *Document) Instances() []*Instance {
	out := make([]*Instance, 0, len(d.order))
	for _, id := range d.order {
		out = append(out, d.instances[id])
	}
	return out
}

// Prototypes returns all registered prototypes.
func (d *Document) Prototypes() []*Prototype {
	out := make([]*Prototype, 0, len(d.prototypes))
	for _, p := range d.prototypes {
		out = append(out, p)
	}
	return out
}

// DeleteInstance removes an instance from the document. Removing an instance
// never touches the prototype registry, so a prototype outlives every
// instance that references it.
func (d *Document) DeleteInstance(id string) bool {
	if _, ok := d.instances[id]; !ok {
		return false
	}
	delete(d.instances, id)
	for i, existing := range d.order {
		if existing == id {
			d.order = append(d.order[:i], d.order[i+1:]...)
			break
		}
	}
	return true
}

// InstanceCount reports how many instances are registered.
func (d *Document) InstanceCount() int { return len(d.instances) }

// PrototypeExists reports whether a prototype is still registered.
func (d *Document) PrototypeExists(id string) bool {
	_, ok := d.prototypes[id]
	return ok
}

func unionRects(nodes []*Node) Rect {
	if len(nodes) == 0 {
		return Rect{}
	}
	bounds := nodes[0].Geometry
	for _, node := range nodes[1:] {
		r := node.Geometry
		x0 := min(bounds.X, r.X)
		y0 := min(bounds.Y, r.Y)
		x1 := max(bounds.X+bounds.Width, r.X+r.Width)
		y1 := max(bounds.Y+bounds.Height, r.Y+r.Height)
		bounds = Rect{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}
	}
	return bounds
}
