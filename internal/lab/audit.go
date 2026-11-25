package lab

import "reflect"

// ColorIsolationReport captures the result of editing one instance color.
type ColorIsolationReport struct {
	TargetInstanceID      string   `json:"target_instance_id"`
	NodeID                string   `json:"node_id"`
	RequestedColor        string   `json:"requested_color"`
	PrototypeColorBefore  string   `json:"prototype_color_before"`
	PrototypeColorAfter   string   `json:"prototype_color_after"`
	PrototypeChanged      bool     `json:"prototype_changed"`
	ChangedInstanceIDs    []string `json:"changed_instance_ids"`
	UnexpectedInstanceIDs []string `json:"unexpected_instance_ids"`
	OtherInstanceChanges  int      `json:"other_instance_change_count"`
	Pass                  bool     `json:"pass"`
}

func effectiveColors(doc *Document) map[string]map[string]string {
	out := make(map[string]map[string]string)
	for _, in := range doc.Instances() {
		proto, _ := doc.Prototype(in.PrototypeID)
		colors := make(map[string]string)
		for _, node := range proto.Nodes {
			color, _ := in.EffectiveColor(proto, node.ID)
			colors[node.ID] = color
		}
		out[in.ID] = colors
	}
	return out
}

// CheckColorIsolation edits one instance and verifies the prototype and all
// sibling instances keep their previous effective colors.
func CheckColorIsolation(doc *Document, targetID, nodeID, color string) (*ColorIsolationReport, error) {
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
	beforeProto := node.Fill

	target.SetColor(nodeID, color)

	after := effectiveColors(doc)
	report := &ColorIsolationReport{
		TargetInstanceID:      targetID,
		NodeID:                nodeID,
		RequestedColor:        color,
		PrototypeColorBefore:  beforeProto,
		PrototypeColorAfter:   node.Fill,
		PrototypeChanged:      node.Fill != beforeProto,
		ChangedInstanceIDs:    []string{},
		UnexpectedInstanceIDs: []string{},
	}
	for id, colors := range before {
		for n, c := range colors {
			if after[id][n] != c {
				report.ChangedInstanceIDs = append(report.ChangedInstanceIDs, id)
				if id != targetID {
					report.UnexpectedInstanceIDs = append(report.UnexpectedInstanceIDs, id)
				}
				break
			}
		}
	}
	report.OtherInstanceChanges = len(report.UnexpectedInstanceIDs)
	report.Pass = !report.PrototypeChanged &&
		report.OtherInstanceChanges == 0 &&
		len(report.ChangedInstanceIDs) == 1 &&
		report.ChangedInstanceIDs[0] == targetID
	return report, nil
}

// DeletionSafetyReport captures deleting an instance while the prototype and
// every surviving instance remain intact.
type DeletionSafetyReport struct {
	DeletedInstanceID       string `json:"deleted_instance_id"`
	DeleteReturned          bool   `json:"delete_returned"`
	InstanceCountBefore     int    `json:"instance_count_before"`
	InstanceCountAfter      int    `json:"instance_count_after"`
	PrototypeID             string `json:"prototype_id"`
	PrototypeExistsAfter    bool   `json:"prototype_exists_after"`
	DeletedInstanceExists   bool   `json:"deleted_instance_exists"`
	SurvivingInstanceID     string `json:"surviving_instance_id"`
	SurvivingInstanceExists bool   `json:"surviving_instance_exists"`
	Pass                    bool   `json:"pass"`
}

// CheckDeletionSafety deletes one instance and checks the prototype survives.
func CheckDeletionSafety(doc *Document, deletedID, survivorID string) (*DeletionSafetyReport, error) {
	target, err := doc.Instance(deletedID)
	if err != nil {
		return nil, err
	}
	before := doc.InstanceCount()
	ok := doc.DeleteInstance(deletedID)
	_, targetErr := doc.Instance(deletedID)
	_, survivorErr := doc.Instance(survivorID)

	report := &DeletionSafetyReport{
		DeletedInstanceID:       deletedID,
		DeleteReturned:          ok,
		InstanceCountBefore:     before,
		InstanceCountAfter:      doc.InstanceCount(),
		PrototypeID:             target.PrototypeID,
		PrototypeExistsAfter:    doc.PrototypeExists(target.PrototypeID),
		DeletedInstanceExists:   targetErr == nil,
		SurvivingInstanceID:     survivorID,
		SurvivingInstanceExists: survivorErr == nil,
	}
	report.Pass = ok &&
		report.PrototypeExistsAfter &&
		!report.DeletedInstanceExists &&
		report.SurvivingInstanceExists &&
		report.InstanceCountAfter == before-1
	return report, nil
}

// GeometrySyncReport verifies bounds, selection outline and render outlines
// translate together.
type GeometrySyncReport struct {
	InstanceID                 string  `json:"instance_id"`
	DeltaX                     float64 `json:"delta_x"`
	DeltaY                     float64 `json:"delta_y"`
	BoundsTranslated           bool    `json:"bounds_translated"`
	SelectionOutlineTranslated bool    `json:"selection_outline_translated"`
	RenderOutlineTranslated    bool    `json:"render_outline_translated"`
	RenderOutlineCountBefore   int     `json:"render_outline_count_before"`
	RenderOutlineCountAfter    int     `json:"render_outline_count_after"`
	Pass                       bool    `json:"pass"`
}

// CheckGeometrySync translates an instance and compares all geometry layers.
func CheckGeometrySync(doc *Document, instanceID string, dx, dy float64) (*GeometrySyncReport, error) {
	before, err := doc.Snapshot(instanceID)
	if err != nil {
		return nil, err
	}
	after, err := doc.Translate(instanceID, dx, dy)
	if err != nil {
		return nil, err
	}

	report := &GeometrySyncReport{
		InstanceID:               instanceID,
		DeltaX:                   dx,
		DeltaY:                   dy,
		RenderOutlineCountBefore: len(before.RenderOutlines),
		RenderOutlineCountAfter:  len(after.RenderOutlines),
	}
	report.BoundsTranslated = approx(after.Bounds.X, before.Bounds.X+dx) &&
		approx(after.Bounds.Y, before.Bounds.Y+dy) &&
		approx(after.Bounds.Width, before.Bounds.Width) &&
		approx(after.Bounds.Height, before.Bounds.Height)
	report.SelectionOutlineTranslated = approx(after.SelectionOutline.X, before.SelectionOutline.X+dx) &&
		approx(after.SelectionOutline.Y, before.SelectionOutline.Y+dy)
	report.RenderOutlineTranslated = len(before.RenderOutlines) == len(after.RenderOutlines)
	if report.RenderOutlineTranslated {
		for i := range before.RenderOutlines {
			beforeRect, afterRect := before.RenderOutlines[i], after.RenderOutlines[i]
			if !approx(afterRect.X, beforeRect.X+dx) || !approx(afterRect.Y, beforeRect.Y+dy) {
				report.RenderOutlineTranslated = false
				break
			}
		}
	}
	report.Pass = report.BoundsTranslated &&
		report.SelectionOutlineTranslated &&
		report.RenderOutlineTranslated &&
		report.RenderOutlineCountBefore == report.RenderOutlineCountAfter
	return report, nil
}

// SharedMapIssue describes a group of instances whose override maps share the
// same underlying map, meaning one edit would fan out to every member.
type SharedMapIssue struct {
	NodeID     string   `json:"node_id"`
	OwnerIDs   []string `json:"owner_instance_ids"`
	OwnerCount int      `json:"owner_count"`
}

// DetectSharedColorMaps inspects map identity across instances. In the correct
// model every instance owns a distinct map and this returns no issues.
func DetectSharedColorMaps(doc *Document) []SharedMapIssue {
	pointerToOwners := make(map[uintptr][]string)
	for _, in := range doc.Instances() {
		if len(in.ColorOverrides) == 0 {
			continue
		}
		ptr := reflect.ValueOf(in.ColorOverrides).Pointer()
		pointerToOwners[ptr] = append(pointerToOwners[ptr], in.ID)
	}
	issues := make([]SharedMapIssue, 0)
	for _, owners := range pointerToOwners {
		if len(owners) > 1 {
			issues = append(issues, SharedMapIssue{
				NodeID:     "*",
				OwnerIDs:   owners,
				OwnerCount: len(owners),
			})
		}
	}
	return issues
}

func approx(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}
