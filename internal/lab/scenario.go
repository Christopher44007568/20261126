package lab

import "fmt"

// Default palette of the Beacon Card prototype.
const (
	ColorBody    = "#5875E8"
	ColorHeader  = "#1E293B"
	ColorAccent  = "#22C55E"
	ColorWarning = "#F59E0B"
	ColorMuted   = "#94A3B8"
)

// NewBeaconCardPrototype builds the five-node prototype used by every
// instance. Node geometry is in local prototype coordinates.
func NewBeaconCardPrototype() *Prototype {
	return &Prototype{
		ID:   "proto-beacon-card",
		Name: "Beacon Card",
		Nodes: []*Node{
			{ID: "body", Label: "卡片主体", Kind: "rect", Fill: ColorBody, Geometry: Rect{X: 0, Y: 24, Width: 180, Height: 116}},
			{ID: "header", Label: "标题栏", Kind: "rect", Fill: ColorHeader, Geometry: Rect{X: 0, Y: 0, Width: 180, Height: 30}},
			{ID: "accent", Label: "强调圆点", Kind: "circle", Fill: ColorAccent, Geometry: Rect{X: 12, Y: 40, Width: 18, Height: 18}},
			{ID: "warning", Label: "警示角标", Kind: "rect", Fill: ColorWarning, Geometry: Rect{X: 140, Y: 36, Width: 28, Height: 20}},
			{ID: "footer", Label: "底部分隔", Kind: "line", Fill: ColorMuted, Geometry: Rect{X: 12, Y: 120, Width: 156, Height: 4}},
		},
	}
}

const instanceCount = 128

// InstanceCount is the number of instances in the standard scenario.
func InstanceCount() int { return instanceCount }

// InstanceID formats the zero-padded instance identifier.
func InstanceID(i int) string { return fmt.Sprintf("I-%03d", i) }

// BuildScenario creates the reproducible 128-instance document. The random
// choices are derived from the instance index, so output is deterministic
// across runs without any external dependency.
func BuildScenario() *Document {
	doc := NewDocument()
	proto := NewBeaconCardPrototype()
	_ = doc.AddPrototype(proto)

	nodeIDs := make([]string, 0, len(proto.Nodes))
	for _, node := range proto.Nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	localColors := []string{"#FF2E93", "#06B6D4", "#A855F7", "#EF4444", "#F97316"}

	colorOverrides := 0
	hiddenInstances := 0
	hiddenNodes := 0

	for i := 1; i <= instanceCount; i++ {
		in := &Instance{
			ID:          InstanceID(i),
			PrototypeID: proto.ID,
			OffsetX:     float64((i % 16) * 20),
			OffsetY:     float64(((i - 1) / 16) * 16),
		}

		// Deterministic local color overrides: every 10th instance, plus I-003.
		// The audit target I-007 deliberately has no seed override.
		if i%10 == 0 || i == 3 {
			nodeID := nodeIDs[(i/10)%len(nodeIDs)]
			in.SetColor(nodeID, localColors[(i/10)%len(localColors)])
			colorOverrides++
		}
		// Deterministic per-node hide: every 40th instance hides its warning.
		if i%40 == 0 {
			in.SetNodeVisible("warning", false)
			hiddenNodes++
		}
		// Deterministic whole-card hide: indices 16, 35, 64, 88, 105, 122.
		switch i {
		case 16, 35, 64, 88, 105, 122:
			in.Hidden = true
			hiddenInstances++
		}

		_ = doc.AddInstance(in)
	}

	// Expose the deterministic counts via scenario metadata for reporting.
	doc.meta = ScenarioMeta{
		ColorOverrides:  colorOverrides,
		HiddenInstances: hiddenInstances,
		HiddenNodes:     hiddenNodes,
	}
	return doc
}

// ScenarioMeta summarizes the seeded local differences.
type ScenarioMeta struct {
	ColorOverrides  int `json:"color_overrides"`
	HiddenInstances int `json:"hidden_instances"`
	HiddenNodes     int `json:"hidden_nodes"`
}

// Meta returns the scenario metadata.
func (d *Document) Meta() ScenarioMeta { return d.meta }
