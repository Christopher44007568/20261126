package lab

import "sort"

// SharedMapFault reproduces the "shared override table" bug: a group of
// instances is handed the same map instance, so an edit through one instance
// changes the effective color of every owner.
type SharedMapFault struct {
	Schema                string   `json:"schema"`
	Kind                  string   `json:"kind"`
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	Mechanism             string   `json:"mechanism"`
	TargetInstanceID      string   `json:"target_instance_id"`
	NodeID                string   `json:"node_id"`
	EditColor             string   `json:"edit_color"`
	AffectedInstanceIDs   []string `json:"affected_instance_ids"`
	UnexpectedInstanceIDs []string `json:"unexpected_instance_ids"`
	AuditDetectable       bool     `json:"audit_detectable"`
}

// OwnerCount reports how many instances share the faulty map.
func (f *SharedMapFault) OwnerCount() int { return len(f.AffectedInstanceIDs) }

// SimulateSharedMapFault builds a fresh scenario, forces 24 instances to
// share one override map, then edits the target through that map.
func SimulateSharedMapFault(targetID, nodeID, color string) (*SharedMapFault, error) {
	doc := BuildScenario()
	target, err := doc.Instance(targetID)
	if err != nil {
		return nil, err
	}
	proto, err := doc.Prototype(target.PrototypeID)
	if err != nil {
		return nil, err
	}
	if _, err := proto.Node(nodeID); err != nil {
		return nil, err
	}

	// The bug: one map handed out to 24 instances, including the edit target.
	shared := map[string]string{nodeID: ColorBody}
	victims := doc.Instances()
	if len(victims) > 24 {
		victims = victims[:24]
	}
	found := false
	for _, in := range victims {
		if in.ID == targetID {
			found = true
			break
		}
	}
	if !found {
		victims[0] = target
	}
	for _, in := range victims {
		in.ColorOverrides = shared
	}

	before := effectiveColors(doc)
	shared[nodeID] = color
	after := effectiveColors(doc)

	affected := make([]string, 0)
	unexpected := make([]string, 0)
	for _, in := range doc.Instances() {
		changed := false
		for n := range before[in.ID] {
			if after[in.ID][n] != before[in.ID][n] {
				changed = true
				break
			}
		}
		if changed {
			affected = append(affected, in.ID)
			if in.ID != targetID {
				unexpected = append(unexpected, in.ID)
			}
		}
	}

	issues := DetectSharedColorMaps(doc)
	return &SharedMapFault{
		Schema:                "instancelab.fault/v1",
		Kind:                  "shared_override_map",
		Title:                 "错误共享颜色：多实例共用同一张覆盖表",
		Description:           "24 个实例被分配了同一个 map，编辑其中一个实例的颜色会穿透到所有共享者。",
		Mechanism:             "instances share one ColorOverrides map instead of per-instance maps",
		TargetInstanceID:      targetID,
		NodeID:                nodeID,
		EditColor:             color,
		AffectedInstanceIDs:   affected,
		UnexpectedInstanceIDs: unexpected,
		AuditDetectable:       len(issues) > 0,
	}, nil
}

// WritebackFault reproduces the "edit writes back to the prototype" bug: an
// instance mutation changes Prototype.Nodes directly, so every instance that
// still inherits the prototype default changes color too.
type WritebackFault struct {
	Schema                string   `json:"schema"`
	Kind                  string   `json:"kind"`
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	Mechanism             string   `json:"mechanism"`
	TargetInstanceID      string   `json:"target_instance_id"`
	NodeID                string   `json:"node_id"`
	EditColor             string   `json:"edit_color"`
	PrototypeColorAfter   string   `json:"prototype_color_after"`
	AffectedInstanceIDs   []string `json:"affected_instance_ids"`
	UnexpectedInstanceIDs []string `json:"unexpected_instance_ids"`
}

// SimulateWritebackFault mutates the shared prototype through one instance.
func SimulateWritebackFault(targetID, nodeID, color string) (*WritebackFault, error) {
	doc := BuildScenario()
	target, err := doc.Instance(targetID)
	if err != nil {
		return nil, err
	}
	proto, err := doc.Prototype(target.PrototypeID)
	if err != nil {
		return nil, err
	}
	node, err := proto.Node(nodeID)
	if err != nil {
		return nil, err
	}

	before := effectiveColors(doc)

	// The bug: instance "edit" writes straight into the shared prototype.
	node.Fill = color

	after := effectiveColors(doc)
	affected := make([]string, 0)
	unexpected := make([]string, 0)
	for _, in := range doc.Instances() {
		changed := false
		for n := range before[in.ID] {
			if after[in.ID][n] != before[in.ID][n] {
				changed = true
				break
			}
		}
		if changed {
			affected = append(affected, in.ID)
			if in.ID != targetID {
				unexpected = append(unexpected, in.ID)
			}
		}
	}
	sort.Strings(affected)
	sort.Strings(unexpected)

	return &WritebackFault{
		Schema:                "instancelab.fault/v1",
		Kind:                  "prototype_writeback",
		Title:                 "错误共享颜色：实例编辑回写了原型",
		Description:           "实例没有独立覆盖项，而是直接修改主原型；所有未本地覆盖该节点的实例全部变色。",
		Mechanism:             "instance mutation writes to shared Prototype.Nodes[].Fill",
		TargetInstanceID:      targetID,
		NodeID:                nodeID,
		EditColor:             color,
		PrototypeColorAfter:   node.Fill,
		AffectedInstanceIDs:   affected,
		UnexpectedInstanceIDs: unexpected,
	}, nil
}
