package lab

// RenderNode is one visible piece of an instance at snapshot time.
type RenderNode struct {
	NodeID     string `json:"node_id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Fill       string `json:"fill"`
	Geometry   Rect   `json:"geometry"`
	Visible    bool   `json:"visible"`
	ColorLocal bool   `json:"color_local"`
}

// Snapshot is the immutable render state of an instance. Bounds, selection
// outline and per-node render outlines are all computed in one pass, so they
// can never drift apart when the instance moves.
type Snapshot struct {
	InstanceID       string       `json:"instance_id"`
	PrototypeID      string       `json:"prototype_id"`
	OffsetX          float64      `json:"offset_x"`
	OffsetY          float64      `json:"offset_y"`
	InstanceHidden   bool         `json:"instance_hidden"`
	Bounds           Rect         `json:"bounds"`
	SelectionOutline Rect         `json:"selection_outline"`
	RenderOutlines   []Rect       `json:"render_outlines"`
	Nodes            []RenderNode `json:"nodes"`
}

// Translate moves an instance by (dx, dy). Geometry is recomputed eagerly via
// Snapshot, so the bounding box and every render outline move together.
func (d *Document) Translate(instanceID string, dx, dy float64) (*Snapshot, error) {
	in, err := d.Instance(instanceID)
	if err != nil {
		return nil, err
	}
	in.OffsetX += dx
	in.OffsetY += dy
	return d.Snapshot(instanceID)
}

// Snapshot computes the full render state for one instance.
func (d *Document) Snapshot(instanceID string) (*Snapshot, error) {
	in, err := d.Instance(instanceID)
	if err != nil {
		return nil, err
	}
	proto, err := d.Prototype(in.PrototypeID)
	if err != nil {
		return nil, err
	}

	snap := &Snapshot{
		InstanceID:     in.ID,
		PrototypeID:    proto.ID,
		OffsetX:        in.OffsetX,
		OffsetY:        in.OffsetY,
		InstanceHidden: in.Hidden,
		RenderOutlines: make([]Rect, 0, len(proto.Nodes)),
		Nodes:          make([]RenderNode, 0, len(proto.Nodes)),
	}

	visibleNodes := make([]*Node, 0, len(proto.Nodes))
	for _, node := range proto.Nodes {
		fill := node.Fill
		_, localColor := in.ColorOverrides[node.ID]
		if localColor {
			fill = in.ColorOverrides[node.ID]
		}
		visible := in.IsNodeVisible(node.ID)

		geometry := node.Geometry
		geometry.X += in.OffsetX
		geometry.Y += in.OffsetY

		snap.Nodes = append(snap.Nodes, RenderNode{
			NodeID:     node.ID,
			Label:      node.Label,
			Kind:       node.Kind,
			Fill:       fill,
			Geometry:   geometry,
			Visible:    visible,
			ColorLocal: localColor,
		})
		if visible {
			visibleNodes = append(visibleNodes, &Node{Geometry: geometry})
			snap.RenderOutlines = append(snap.RenderOutlines, geometry)
		}
	}

	// Bounds and selection outline share the same source of truth: the union
	// of currently visible node geometries in this single snapshot.
	snap.Bounds = unionRects(visibleNodes)
	snap.SelectionOutline = snap.Bounds
	return snap, nil
}
