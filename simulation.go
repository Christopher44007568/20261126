package main

import (
	"fmt"
)

const (
	defaultWidth    = 480
	defaultHeight   = 270
	arenaCapacity   = 8
	objectsPerFrame = 4
)

type QueryRecord struct {
	ID         string              `json:"id"`
	Frame      int                 `json:"frame"`
	Sequence   string              `json:"sequence"`
	Generation GenID               `json:"generation"`
	StartTick  int                 `json:"start_tick"`
	DoneTick   int                 `json:"done_tick"`
	Slots      []int               `json:"slots"`
	Dangling   []DanglingReference `json:"dangling_reads,omitempty"`
}

type FrameResult struct {
	Frame        int      `json:"frame"`
	Name         string   `json:"name"`
	CompleteTick int      `json:"complete_tick"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	GhostPixels  int      `json:"ghost_pixels"`
	Queries      []string `json:"queries"`
	Pixels       []byte   `json:"-"`
}

type Metrics struct {
	FramesRendered              int `json:"frames_rendered"`
	GhostPixels                 int `json:"ghost_pixels"`
	DanglingRefs                int `json:"dangling_refs"`
	PrematureReleases           int `json:"premature_releases"`
	ReuseEvents                 int `json:"reuse_events"`
	SafeReuseEvents             int `json:"safe_reuse_events"`
	BuildsOverlappedWithQueries int `json:"builds_overlapped_with_queries"`
}

type Report struct {
	Policy      string              `json:"policy"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	Frames      []FrameResult       `json:"frames"`
	Generations []*Generation       `json:"generations"`
	Slots       []SlotRecord        `json:"slots"`
	Queries     []*QueryRecord      `json:"queries"`
	References  []*Reference        `json:"references"`
	Reuses      []ReuseRecord       `json:"reuse_events"`
	Dangling    []DanglingReference `json:"dangling_references"`
	Events      []EventRecord       `json:"events"`
	Metrics     Metrics             `json:"metrics"`
}

type queryTask struct {
	record  *QueryRecord
	gen     *Generation
	ref     *Reference
	parity  int
	doneAt  int
	frame   *frameCapture
	started bool
	done    bool
}

type frameCapture struct {
	index      int
	scene      Scene
	width      int
	height     int
	pixels     []byte
	queries    []*queryTask
	completeAt int
}

type framePlan struct {
	frame     *frameCapture
	requestAt int
	doneTicks [2]int
	gen       *Generation
	built     bool
	requested bool
}

type Simulation struct {
	SceneFrames []Scene
	Width       int
	Height      int
	Arena       *Arena
	frames      []*frameCapture
	plans       []*framePlan
	queries     []*queryTask
	overlap     int
	endTick     int
}

func RunSimulation(width, height int) (*Simulation, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image size %dx%d", width, height)
	}
	scenes := frames()
	if len(scenes) != 3 {
		return nil, fmt.Errorf("expected three scene frames, got %d", len(scenes))
	}
	sim := &Simulation{
		SceneFrames: scenes,
		Width:       width,
		Height:      height,
		Arena:       NewArena(arenaCapacity),
	}

	requestTicks := []int{0, 1, 3}
	doneTicks := [][2]int{{3, 4}, {4, 5}, {6, 7}}
	for i, scene := range scenes {
		capture := &frameCapture{
			index: i, scene: scene, width: width, height: height,
			pixels: make([]byte, width*height*3), completeAt: noTick,
		}
		sim.frames = append(sim.frames, capture)
		plan := &framePlan{
			frame: capture, requestAt: requestTicks[i], doneTicks: doneTicks[i],
		}
		sim.plans = append(sim.plans, plan)
	}

	maxDone := 0
	for i, plan := range sim.plans {
		plan.gen = sim.Arena.NewGeneration(i+1, plan.requestAt)
		for j := 0; j < 2; j++ {
			if plan.doneTicks[j] > maxDone {
				maxDone = plan.doneTicks[j]
			}
			sequence := "A"
			if j == 1 {
				sequence = "B"
			}
			record := &QueryRecord{
				ID:         fmt.Sprintf("F%d-%s", i+1, sequence),
				Frame:      i + 1,
				Sequence:   sequence,
				Generation: plan.gen.ID,
				StartTick:  noTick,
				DoneTick:   plan.doneTicks[j],
			}
			task := &queryTask{
				record: record, gen: plan.gen, parity: j,
				doneAt: plan.doneTicks[j], frame: plan.frame,
			}
			sim.queries = append(sim.queries, task)
			plan.frame.queries = append(plan.frame.queries, task)
		}
	}
	sim.endTick = maxDone + 1

	for tick := 0; tick <= sim.endTick; tick++ {
		sim.Arena.SetTick(tick)

		for _, task := range sim.queries {
			if !task.started || task.done || tick < task.doneAt {
				continue
			}
			sim.completeQuery(tick, task)
		}

		for i, plan := range sim.plans {
			if plan.built || tick < plan.requestAt {
				continue
			}
			if !plan.requested {
				if i > 0 {
					sim.Arena.Retire(sim.plans[i-1].gen)
				}
				plan.requested = true
			}
			nodes := make([]ASNode, 0, len(plan.frame.scene.Objects))
			for _, object := range plan.frame.scene.Objects {
				nodes = append(nodes, ASNode{Generation: plan.gen.ID, Object: object})
			}
			if !sim.Arena.TryBuild(plan.gen, nodes) {
				continue
			}
			plan.built = true
			for _, other := range sim.Arena.Generations() {
				if other.ID != plan.gen.ID && other.refs > 0 {
					sim.overlap++
					break
				}
			}
			sim.issueFrameQueries(tick, plan)
		}
	}

	last := sim.plans[len(sim.plans)-1]
	if last.gen != nil {
		sim.Arena.Retire(last.gen)
	}
	return sim, nil
}

func (s *Simulation) issueFrameQueries(tick int, plan *framePlan) {
	for _, task := range plan.frame.queries {
		if task.started {
			continue
		}
		ref, err := s.Arena.Acquire(plan.gen, task.record.ID)
		if err != nil {
			panic(err)
		}
		task.ref = ref
		task.started = true
		task.record.StartTick = tick
		task.record.Slots = append([]int(nil), plan.gen.Slots...)
		if task.doneAt <= tick {
			task.doneAt = tick + 1 + task.parity
			if end := task.doneAt + 1; end > s.endTick {
				s.endTick = end
			}
		}
		task.record.DoneTick = task.doneAt
	}
}

func (s *Simulation) completeQuery(tick int, task *queryTask) {
	before := len(s.Arena.Dangling())
	nodes := s.Arena.ReadGeneration(task.gen, task.record.ID)
	after := s.Arena.Dangling()
	if len(after) > before {
		task.record.Dangling = append(task.record.Dangling, after[before:]...)
	}
	objects := make([]SceneObject, 0, len(nodes))
	for _, node := range nodes {
		objects = append(objects, node.Object)
	}
	for y := 0; y < task.frame.height; y++ {
		if y%2 != task.parity {
			continue
		}
		for x := 0; x < task.frame.width; x++ {
			rgb := sampleScene(objects, x, y, task.frame.width, task.frame.height)
			i := (y*task.frame.width + x) * 3
			copy(task.frame.pixels[i:i+3], rgb[:])
		}
	}
	task.done = true
	task.record.DoneTick = tick
	s.Arena.Release(task.ref)
	if task.frame.completeAt < tick {
		task.frame.completeAt = tick
	}
}

func (s *Simulation) BuildReport() Report {
	report := Report{
		Policy:      "generation-tagged slots + per-query references + retirement queue",
		Width:       s.Width,
		Height:      s.Height,
		Generations: s.Arena.Generations(),
		Slots:       s.Arena.Slots(),
		References:  s.Arena.References(),
		Reuses:      s.Arena.Reuses(),
		Dangling:    s.Arena.Dangling(),
		Events:      s.Arena.Events(),
	}
	for _, task := range s.queries {
		report.Queries = append(report.Queries, task.record)
	}
	for _, capture := range s.frames {
		oracle := renderOracle(capture.scene, capture.width, capture.height)
		ghost := countPixelDifferences(capture.pixels, oracle)
		frame := FrameResult{
			Frame: capture.index + 1, Name: capture.scene.Name,
			CompleteTick: capture.completeAt, Width: capture.width, Height: capture.height,
			GhostPixels: ghost, Pixels: capture.pixels,
		}
		for _, q := range capture.queries {
			frame.Queries = append(frame.Queries, q.record.ID)
		}
		report.Frames = append(report.Frames, frame)
		report.Metrics.GhostPixels += ghost
		if capture.completeAt != noTick {
			report.Metrics.FramesRendered++
		}
	}
	report.Metrics.DanglingRefs = len(report.Dangling)
	for _, g := range report.Generations {
		if g.ReleasedAt != noTick && g.LastRefDoneAt != noTick && g.ReleasedAt < g.LastRefDoneAt {
			report.Metrics.PrematureReleases++
		}
	}
	report.Metrics.ReuseEvents = len(report.Reuses)
	for _, reuse := range report.Reuses {
		if reuse.Safe() {
			report.Metrics.SafeReuseEvents++
		}
	}
	report.Metrics.BuildsOverlappedWithQueries = s.overlap
	return report
}

func countPixelDifferences(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	count := 0
	for i := 0; i+2 < n; i += 3 {
		if a[i] != b[i] || a[i+1] != b[i+1] || a[i+2] != b[i+2] {
			count++
		}
	}
	return count
}
