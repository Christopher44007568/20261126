package main

import (
	"fmt"
)

type GenID uint64

const noTick = -1

const (
	eventBuildRequested   = "build_requested"
	eventBuildBlocked     = "build_blocked"
	eventBuildCompleted   = "build_completed"
	eventRefAcquired      = "ref_acquired"
	eventRefReleased      = "ref_released"
	eventGenerationRetire = "generation_retired"
	eventGenerationFree   = "generation_released"
	eventSlotReused       = "slot_reused"
)

type ASNode struct {
	Generation GenID       `json:"generation"`
	Object     SceneObject `json:"object"`
}

type Generation struct {
	ID               GenID `json:"id"`
	Frame            int   `json:"frame"`
	Slots            []int `json:"slots"`
	BuildRequestedAt int   `json:"build_requested_at"`
	BuildAttemptsAt  []int `json:"build_attempts_at"`
	BuiltAt          int   `json:"built_at"`
	RetiredAt        int   `json:"retired_at"`
	ReleasedAt       int   `json:"released_at"`
	LastRefDoneAt    int   `json:"last_ref_done_at"`
	RefsIssued       int   `json:"refs_issued"`
	refs             int
	retired          bool
	built            bool
}

func (g *Generation) Built() bool   { return g.built }
func (g *Generation) Retired() bool { return g.retired }
func (g *Generation) ActiveRefs() int {
	return g.refs
}

type Reference struct {
	Query      string `json:"query"`
	Generation GenID  `json:"generation"`
	AcquiredAt int    `json:"acquired_at"`
	ReleasedAt int    `json:"released_at"`
	released   bool
}

func (r *Reference) Released() bool { return r.released }

type SlotRecord struct {
	Index       int   `json:"index"`
	Generation  GenID `json:"generation"`
	AllocatedAt int   `json:"allocated_at"`
	FreedAt     int   `json:"freed_at"`
	ReuseCount  int   `json:"reuse_count"`
	LastOwner   GenID `json:"last_owner"`
}

type ReuseRecord struct {
	Slot              int   `json:"slot"`
	From              GenID `json:"from_generation"`
	To                GenID `json:"to_generation"`
	WaitStart         int   `json:"wait_start"`
	ReusedAt          int   `json:"reused_at"`
	LastFromRefDone   int   `json:"last_from_ref_done"`
	ActiveRefsAtReuse int   `json:"active_refs_at_reuse"`
}

func (r ReuseRecord) Safe() bool {
	return r.ActiveRefsAtReuse == 0 && r.ReusedAt >= r.LastFromRefDone
}

type DanglingReference struct {
	Query              string `json:"query"`
	Slot               int    `json:"slot"`
	ExpectedGeneration GenID  `json:"expected_generation"`
	ActualGeneration   GenID  `json:"actual_generation"`
	Tick               int    `json:"tick"`
}

type EventRecord struct {
	Tick   int    `json:"tick"`
	Kind   string `json:"kind"`
	Gen    GenID  `json:"generation,omitempty"`
	From   GenID  `json:"from_generation,omitempty"`
	Slot   *int   `json:"slot,omitempty"`
	Query  string `json:"query,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type arenaSlot struct {
	node        ASNode
	used        bool
	allocatedAt int
	freedAt     int
	reuseCount  int
	lastOwner   GenID
}

type Arena struct {
	capacity int
	now      int
	slots    []arenaSlot
	free     []int
	gens     map[GenID]*Generation
	queue    []GenID
	events   []EventRecord
	reuses   []ReuseRecord
	dangling []DanglingReference
	refs     []*Reference
	nextGen  GenID
}

func NewArena(capacity int) *Arena {
	if capacity < 1 {
		panic("arena capacity must be positive")
	}
	a := &Arena{
		capacity: capacity,
		slots:    make([]arenaSlot, capacity),
		gens:     make(map[GenID]*Generation),
	}
	a.free = make([]int, 0, capacity)
	for i := 0; i < capacity; i++ {
		a.free = append(a.free, i)
	}
	return a
}

func (a *Arena) SetTick(tick int) {
	if tick < a.now {
		panic("arena tick cannot move backwards")
	}
	a.now = tick
}

func (a *Arena) Tick() int { return a.now }

func (a *Arena) NewGeneration(frame, requestedAt int) *Generation {
	a.nextGen++
	g := &Generation{
		ID:               a.nextGen,
		Frame:            frame,
		BuildRequestedAt: requestedAt,
		BuiltAt:          noTick,
		RetiredAt:        noTick,
		ReleasedAt:       noTick,
		LastRefDoneAt:    noTick,
	}
	a.gens[g.ID] = g
	a.events = append(a.events, EventRecord{
		Tick: requestedAt, Kind: eventBuildRequested, Gen: g.ID,
		Detail: fmt.Sprintf("frame %d requests a new generation", frame),
	})
	return g
}

func (a *Arena) TryBuild(g *Generation, nodes []ASNode) bool {
	if g == nil {
		return false
	}
	if g.built {
		return true
	}
	g.BuildAttemptsAt = append(g.BuildAttemptsAt, a.now)
	a.drainRetirementQueue()
	if len(a.free) < len(nodes) {
		a.events = append(a.events, EventRecord{
			Tick: a.now, Kind: eventBuildBlocked, Gen: g.ID,
			Detail: fmt.Sprintf("need %d free slots, have %d", len(nodes), len(a.free)),
		})
		return false
	}

	g.Slots = make([]int, 0, len(nodes))
	for _, node := range nodes {
		index := a.free[0]
		a.free = a.free[1:]
		slot := &a.slots[index]
		previous := slot.lastOwner
		slot.node = ASNode{Generation: g.ID, Object: node.Object}
		slot.used = true
		slot.allocatedAt = a.now
		slot.freedAt = noTick
		slot.reuseCount++
		slot.lastOwner = g.ID
		g.Slots = append(g.Slots, index)

		if previous != 0 && previous != g.ID {
			from := a.gens[previous]
			lastDone := noTick
			echo := 0
			if from != nil {
				lastDone = from.LastRefDoneAt
				echo = from.refs
			}
			a.reuses = append(a.reuses, ReuseRecord{
				Slot: index, From: previous, To: g.ID,
				WaitStart: g.BuildRequestedAt, ReusedAt: a.now,
				LastFromRefDone: lastDone, ActiveRefsAtReuse: echo,
			})
			slotNumber := index
			a.events = append(a.events, EventRecord{
				Tick: a.now, Kind: eventSlotReused, Gen: g.ID, From: previous, Slot: &slotNumber,
				Detail: fmt.Sprintf("last ref done t%d, active refs at reuse %d", lastDone, echo),
			})
		}
	}
	g.built = true
	g.BuiltAt = a.now
	a.events = append(a.events, EventRecord{
		Tick: a.now, Kind: eventBuildCompleted, Gen: g.ID,
		Detail: fmt.Sprintf("slots %v", g.Slots),
	})
	return true
}

func (a *Arena) Acquire(g *Generation, query string) (*Reference, error) {
	if g == nil || !g.built {
		return nil, fmt.Errorf("generation is not built")
	}
	if g.retired {
		return nil, fmt.Errorf("generation %d is retired", g.ID)
	}
	g.refs++
	g.RefsIssued++
	ref := &Reference{
		Query: query, Generation: g.ID, AcquiredAt: a.now, ReleasedAt: noTick,
	}
	a.refs = append(a.refs, ref)
	a.events = append(a.events, EventRecord{
		Tick: a.now, Kind: eventRefAcquired, Gen: g.ID, Query: query,
	})
	return ref, nil
}

func (a *Arena) Release(ref *Reference) {
	if ref == nil || ref.released {
		return
	}
	g := a.gens[ref.Generation]
	if g == nil {
		return
	}
	ref.released = true
	ref.ReleasedAt = a.now
	g.refs--
	g.LastRefDoneAt = a.now
	a.events = append(a.events, EventRecord{
		Tick: a.now, Kind: eventRefReleased, Gen: g.ID, Query: ref.Query,
		Detail: fmt.Sprintf("active refs %d", g.refs),
	})
	a.drainRetirementQueue()
}

func (a *Arena) Retire(g *Generation) {
	if g == nil || g.retired {
		return
	}
	g.retired = true
	g.RetiredAt = a.now
	a.queue = append(a.queue, g.ID)
	a.events = append(a.events, EventRecord{
		Tick: a.now, Kind: eventGenerationRetire, Gen: g.ID,
		Detail: fmt.Sprintf("queued with %d active refs", g.refs),
	})
	a.drainRetirementQueue()
}

func (a *Arena) drainRetirementQueue() {
	for i := 0; i < len(a.queue); {
		g := a.gens[a.queue[i]]
		if g == nil || g.refs > 0 {
			i++
			continue
		}
		a.releaseGeneration(g)
		a.queue = append(a.queue[:i], a.queue[i+1:]...)
	}
}

func (a *Arena) releaseGeneration(g *Generation) {
	if g.ReleasedAt != noTick {
		return
	}
	if g.refs > 0 {
		panic("attempted to release generation with active references")
	}
	if g.LastRefDoneAt != noTick && a.now < g.LastRefDoneAt {
		panic("attempted to release generation before its last query completed")
	}
	g.ReleasedAt = a.now
	for _, index := range g.Slots {
		slot := &a.slots[index]
		if slot.node.Generation != g.ID {
			continue
		}
		slot.used = false
		slot.freedAt = a.now
		a.free = append(a.free, index)
	}
	a.events = append(a.events, EventRecord{
		Tick: a.now, Kind: eventGenerationFree, Gen: g.ID,
		Detail: fmt.Sprintf("returned slots %v after last ref t%d", g.Slots, g.LastRefDoneAt),
	})
}

func (a *Arena) ReadGeneration(g *Generation, query string) []ASNode {
	if g == nil {
		return nil
	}
	nodes := make([]ASNode, 0, len(g.Slots))
	for _, index := range g.Slots {
		slot := &a.slots[index]
		if slot.node.Generation != g.ID {
			a.dangling = append(a.dangling, DanglingReference{
				Query: query, Slot: index, ExpectedGeneration: g.ID,
				ActualGeneration: slot.node.Generation, Tick: a.now,
			})
		}
		nodes = append(nodes, slot.node)
	}
	return nodes
}

func (a *Arena) Generations() []*Generation {
	out := make([]*Generation, 0, len(a.gens))
	for i := GenID(1); i <= a.nextGen; i++ {
		if g := a.gens[i]; g != nil {
			out = append(out, g)
		}
	}
	return out
}

func (a *Arena) Slots() []SlotRecord {
	out := make([]SlotRecord, 0, len(a.slots))
	for i, slot := range a.slots {
		out = append(out, SlotRecord{
			Index: i, Generation: slot.node.Generation,
			AllocatedAt: slot.allocatedAt, FreedAt: slot.freedAt,
			ReuseCount: slot.reuseCount, LastOwner: slot.lastOwner,
		})
	}
	return out
}

func (a *Arena) Reuses() []ReuseRecord {
	out := make([]ReuseRecord, len(a.reuses))
	copy(out, a.reuses)
	return out
}

func (a *Arena) References() []*Reference {
	out := make([]*Reference, len(a.refs))
	copy(out, a.refs)
	return out
}

func (a *Arena) Dangling() []DanglingReference {
	out := make([]DanglingReference, len(a.dangling))
	copy(out, a.dangling)
	return out
}

func (a *Arena) Events() []EventRecord {
	out := make([]EventRecord, len(a.events))
	copy(out, a.events)
	return out
}
