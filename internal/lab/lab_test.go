package lab

import "testing"

func TestScenarioShape(t *testing.T) {
	doc := BuildScenario()
	if got := doc.InstanceCount(); got != 128 {
		t.Fatalf("instance count = %d, want 128", got)
	}
	if len(doc.Prototypes()) != 1 {
		t.Fatalf("prototype count = %d, want 1", len(doc.Prototypes()))
	}
	meta := doc.Meta()
	if meta.ColorOverrides != 13 {
		t.Errorf("color overrides = %d, want 13", meta.ColorOverrides)
	}
	if meta.HiddenInstances != 6 {
		t.Errorf("hidden instances = %d, want 6", meta.HiddenInstances)
	}
	if meta.HiddenNodes != 3 {
		t.Errorf("hidden nodes = %d, want 3", meta.HiddenNodes)
	}
	visible := 0
	for _, in := range doc.Instances() {
		if !in.Hidden {
			visible++
		}
	}
	if visible != 122 {
		t.Errorf("visible = %d, want 122", visible)
	}
}

func TestColorIsolation(t *testing.T) {
	doc := BuildScenario()
	report, err := CheckColorIsolation(doc, "I-007", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass {
		t.Fatalf("isolation check failed: %+v", report)
	}
	proto, _ := doc.Prototype("proto-beacon-card")
	node, _ := proto.Node("body")
	if node.Fill != ColorBody {
		t.Errorf("prototype body fill = %s, want %s", node.Fill, ColorBody)
	}
	target, _ := doc.Instance("I-007")
	if got := target.ColorOverrides["body"]; got != "#FF2E93" {
		t.Errorf("local override = %q, want #FF2E93", got)
	}
	sibling, _ := doc.Instance("I-008")
	color, _ := sibling.EffectiveColor(proto, "body")
	if color != ColorBody {
		t.Errorf("sibling color = %s, want prototype default %s", color, ColorBody)
	}
}

func TestDeletionSafety(t *testing.T) {
	doc := BuildScenario()
	report, err := CheckDeletionSafety(doc, "I-128", "I-001")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass {
		t.Fatalf("deletion check failed: %+v", report)
	}
	if doc.InstanceCount() != 127 {
		t.Errorf("count after delete = %d, want 127", doc.InstanceCount())
	}
	if !doc.PrototypeExists("proto-beacon-card") {
		t.Error("prototype was removed when deleting an instance")
	}
	if doc.DeleteInstance("does-not-exist") {
		t.Error("delete of missing instance returned true")
	}
}

func TestGeometrySync(t *testing.T) {
	doc := BuildScenario()
	before, err := doc.Snapshot("I-001")
	if err != nil {
		t.Fatal(err)
	}
	report, err := CheckGeometrySync(doc, "I-001", 24, 18)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass {
		t.Fatalf("geometry check failed: %+v", report)
	}
	after, _ := doc.Snapshot("I-001")
	if after.Bounds != after.SelectionOutline {
		t.Error("bounds and selection outline diverged")
	}
	if len(before.RenderOutlines) != len(after.RenderOutlines) {
		t.Error("render outline count changed after translation")
	}
	if len(after.RenderOutlines) != len(before.Nodes) {
		t.Error("render outlines do not cover every visible node")
	}
}

func TestNoSharedMapsInCorrectModel(t *testing.T) {
	doc := BuildScenario()
	if issues := DetectSharedColorMaps(doc); len(issues) != 0 {
		t.Fatalf("unexpected shared maps: %+v", issues)
	}
}

func TestSharedMapFaultDetected(t *testing.T) {
	fault, err := SimulateSharedMapFault("I-037", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}
	if !fault.AuditDetectable {
		t.Error("audit failed to detect the shared map")
	}
	if fault.OwnerCount() != 24 {
		t.Errorf("owner count = %d, want 24", fault.OwnerCount())
	}
	if len(fault.AffectedInstanceIDs) != 24 {
		t.Errorf("affected = %d, want 24", len(fault.AffectedInstanceIDs))
	}
	if len(fault.UnexpectedInstanceIDs) != 23 {
		t.Errorf("collateral changes = %d, want 23", len(fault.UnexpectedInstanceIDs))
	}
}

func TestWritebackFaultFansOut(t *testing.T) {
	fault, err := SimulateWritebackFault("I-004", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}
	if fault.PrototypeColorAfter != "#FF2E93" {
		t.Errorf("prototype after = %s", fault.PrototypeColorAfter)
	}
	if len(fault.UnexpectedInstanceIDs) == 0 {
		t.Error("writeback fault affected no sibling instances")
	}
	for _, id := range fault.UnexpectedInstanceIDs {
		if id == "I-004" {
			t.Error("target instance must not appear in unexpected list")
		}
	}
}

func TestPerNodeVisibility(t *testing.T) {
	doc := BuildScenario()
	in, _ := doc.Instance("I-040")
	if in.IsNodeVisible("warning") {
		t.Error("I-040 warning should be hidden per scenario seeding")
	}
	in.SetNodeVisible("warning", true)
	if !in.IsNodeVisible("warning") {
		t.Error("warning did not become visible again")
	}
	in.Hidden = true
	if in.IsNodeVisible("body") {
		t.Error("whole-card hide should hide every node")
	}
}
