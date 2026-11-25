package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSimulationSafety(t *testing.T) {
	sim, err := RunSimulation(96, 54)
	if err != nil {
		t.Fatal(err)
	}
	report := sim.BuildReport()
	if err := VerifyReport(report); err != nil {
		t.Fatal(err)
	}
	if report.Metrics.ReuseEvents == 0 {
		t.Fatal("expected at least one slot reuse")
	}
	if report.Metrics.BuildsOverlappedWithQueries == 0 {
		t.Fatal("expected builds to overlap active light-path queries")
	}
	if report.Metrics.SafeReuseEvents != report.Metrics.ReuseEvents {
		t.Fatalf("unsafe reuse: %d/%d", report.Metrics.SafeReuseEvents, report.Metrics.ReuseEvents)
	}

	gens := make(map[GenID]*Generation)
	for _, gen := range report.Generations {
		gens[gen.ID] = gen
		if gen.ReleasedAt == noTick || gen.LastRefDoneAt == noTick {
			t.Fatalf("generation %d was not fully retired", gen.ID)
		}
		if gen.ReleasedAt < gen.LastRefDoneAt {
			t.Fatalf("generation %d released t%d before last query t%d", gen.ID, gen.ReleasedAt, gen.LastRefDoneAt)
		}
	}
	if got := len(gens[3].BuildAttemptsAt); got != 2 || gens[3].BuildAttemptsAt[0] != 3 || gens[3].BuildAttemptsAt[1] != 4 {
		t.Fatalf("generation 3 should be blocked at t3 and built at t4, attempts=%v", gens[3].BuildAttemptsAt)
	}
	if gens[1].ReleasedAt != 4 || gens[1].LastRefDoneAt != 4 {
		t.Fatalf("generation 1 release must follow F1-B at t4: release=%d last=%d", gens[1].ReleasedAt, gens[1].LastRefDoneAt)
	}
	for _, ref := range report.References {
		if !ref.Released() {
			t.Fatalf("reference %s was not released", ref.Query)
		}
		if ref.ReleasedAt < ref.AcquiredAt {
			t.Fatalf("reference %s released before acquisition", ref.Query)
		}
	}
	for _, frame := range report.Frames {
		if frame.GhostPixels != 0 {
			t.Fatalf("frame %d has %d ghost pixels", frame.Frame, frame.GhostPixels)
		}
	}
}

func TestArenaRetirementQueueBlocksReuse(t *testing.T) {
	a := NewArena(2)
	a.SetTick(0)
	g1 := a.NewGeneration(1, 0)
	if !a.TryBuild(g1, testNodes(1, 2)) {
		t.Fatal("generation 1 build failed")
	}
	ref, err := a.Acquire(g1, "F1-A")
	if err != nil {
		t.Fatal(err)
	}

	a.SetTick(1)
	a.Retire(g1)
	g2 := a.NewGeneration(2, 1)
	if a.TryBuild(g2, testNodes(2, 2)) {
		t.Fatal("slot reuse must be blocked while a query reference is active")
	}
	if g1.ReleasedAt != noTick {
		t.Fatal("generation 1 released while its query was active")
	}

	a.SetTick(2)
	a.Release(ref)
	if g1.ReleasedAt != 2 {
		t.Fatalf("generation 1 release tick = %d, want 2", g1.ReleasedAt)
	}
	if g1.ReleasedAt < g1.LastRefDoneAt {
		t.Fatal("generation released before last query completion")
	}
	if !a.TryBuild(g2, testNodes(2, 2)) {
		t.Fatal("generation 2 build should proceed after refcount reaches zero")
	}
	reuses := a.Reuses()
	if len(reuses) != 2 {
		t.Fatalf("expected two slot reuse records, got %d", len(reuses))
	}
	for _, reuse := range reuses {
		if !reuse.Safe() {
			t.Fatalf("unsafe reuse record: %+v", reuse)
		}
	}
}

func TestReadGenerationDetectsDanglingSlot(t *testing.T) {
	a := NewArena(1)
	a.SetTick(0)
	g1 := a.NewGeneration(1, 0)
	if !a.TryBuild(g1, testNodes(1, 1)) {
		t.Fatal("build failed")
	}
	a.slots[0].node.Generation = 99
	_ = a.ReadGeneration(g1, "FORCED")
	dangling := a.Dangling()
	if len(dangling) != 1 {
		t.Fatalf("expected one dangling read, got %d", len(dangling))
	}
	if dangling[0].ExpectedGeneration != 1 || dangling[0].ActualGeneration != 99 {
		t.Fatalf("unexpected dangling record: %+v", dangling[0])
	}
}

func TestTimelineUsesFullTickScale(t *testing.T) {
	report := mustReport(t, 48, 27)
	if got := reportMaxTick(report); got != 8 {
		t.Fatalf("max tick = %d, want 8", got)
	}
	c := RenderReuseTimeline(report)
	left, right, maxTick := 190, 1130, 8
	x := func(tick int) int {
		return left + int(float64(tick)/float64(maxTick)*float64(right-left))
	}
	if got := pixelAt(c, x(2), 160); got != colGen1 {
		t.Fatalf("generation 1 slot color at t2 = %+v, want %+v", got, colGen1)
	}
	if got := pixelAt(c, x(6), 160); got != colGen3 {
		t.Fatalf("generation 3 slot color at t6 = %+v, want %+v", got, colGen3)
	}
	if got := pixelAt(c, x(2), 490); got != colGreen {
		t.Fatalf("query A bar at t2 = %+v, want %+v", got, colGreen)
	}
}

func TestWriteArtifacts(t *testing.T) {
	report := mustReport(t, 48, 27)
	dir := t.TempDir()
	if err := WriteArtifacts(report, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"frame_001.ppm", "frame_002.ppm", "frame_003.ppm",
		"reuse_intervals.ppm", "dangling_references.ppm",
		"report.json", "dangling_references.dot",
	} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	header, err := os.ReadFile(filepath.Join(dir, "frame_001.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(header, []byte("P6\n48 27\n255\n")) {
		t.Fatalf("invalid PPM header: %q", header[:20])
	}
	jsonData, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Metrics.DanglingRefs != 0 || decoded.Metrics.GhostPixels != 0 {
		t.Fatalf("unexpected report metrics: %+v", decoded.Metrics)
	}
	dot, err := os.ReadFile(filepath.Join(dir, "dangling_references.dot"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dot), "digraph acceleration_structure_references") {
		t.Fatal("DOT graph missing header")
	}
}

func mustReport(t *testing.T, width, height int) Report {
	t.Helper()
	sim, err := RunSimulation(width, height)
	if err != nil {
		t.Fatal(err)
	}
	report := sim.BuildReport()
	if err := VerifyReport(report); err != nil {
		t.Fatal(err)
	}
	return report
}

func testNodes(gen GenID, count int) []ASNode {
	nodes := make([]ASNode, count)
	for i := range nodes {
		nodes[i] = ASNode{
			Generation: gen,
			Object: SceneObject{
				ID: "test", Sphere: Sphere{Radius: 0.5}, Visible: true,
			},
		}
	}
	return nodes
}

func pixelAt(c *Canvas, x, y int) RGB {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return RGB{}
	}
	i := (y*c.W + x) * 3
	return RGB{c.Pix[i], c.Pix[i+1], c.Pix[i+2]}
}

func TestRunSimulationRejectsInvalidSize(t *testing.T) {
	if _, err := RunSimulation(0, 10); err == nil {
		t.Fatal("expected invalid size error")
	}
}
