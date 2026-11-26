package main

import "fmt"

type Handle struct {
	Tier        Tier
	Generation  uint64
	BudgetBytes int
	pkg         *Package
}

type Transition struct {
	From          Tier   `json:"from"`
	To            Tier   `json:"to"`
	BudgetBytes   int    `json:"budget_bytes"`
	Generation    uint64 `json:"generation"`
	ResidentBytes int    `json:"resident_bytes"`
	Reason        string `json:"reason"`
}

type Manager struct {
	detailed *Mesh
	coarse   *Mesh
	plan     BudgetPlan

	budget     int
	generation uint64
	current    *Handle

	BuildCounts   map[Tier]int `json:"build_counts"`
	EvictionCount int          `json:"eviction_count"`
	Transitions   []Transition `json:"transitions"`
}

func NewManager(detailed, coarse *Mesh, plan BudgetPlan) *Manager {
	return &Manager{
		detailed: detailed,
		coarse:   coarse,
		plan:     plan,
		BuildCounts: map[Tier]int{
			TierFull: 0, TierCompressed: 0, TierFallback: 0,
		},
	}
}

func (m *Manager) Plan() BudgetPlan { return m.plan }

func (m *Manager) CurrentHandle() *Handle { return m.current }

func (m *Manager) ResidentBytes() int {
	if m.current == nil || m.current.pkg == nil {
		return 0
	}
	return m.current.pkg.Bytes
}

func (m *Manager) SetBudget(limit int, reason string) (*Handle, error) {
	tier := m.plan.TierForLimit(limit)
	if tier == TierFallback && limit < m.plan.Low.RequiredBytes {
		return nil, fmt.Errorf("%w: limit=%d required=%d", ErrBudgetTooSmall, limit, m.plan.Low.RequiredBytes)
	}
	if m.current != nil && m.current.Tier == tier {
		m.budget = limit
		m.current.BudgetBytes = limit
		return m.current, nil
	}
	return m.activate(tier, limit, reason)
}

// RestoreFull always evicts the degraded package and builds a fresh full BVH.
// It never reuses a node slice or handle from before the budget reduction.
func (m *Manager) RestoreFull(reason string) (*Handle, error) {
	return m.activate(TierFull, m.plan.High.LimitBytes, reason)
}

func (m *Manager) activate(tier Tier, limit int, reason string) (*Handle, error) {
	var from Tier
	if m.current != nil {
		from = m.current.Tier
		m.current.pkg.Evict()
		m.current = nil
		m.EvictionCount++
	}

	m.generation++
	generation := m.generation

	var pkg *Package
	var err error
	switch tier {
	case TierFull:
		pkg = NewFullPackage(m.detailed, generation)
	case TierCompressed:
		pkg, err = NewCompressedPackage(m.detailed, generation)
	case TierFallback:
		pkg = NewFallbackPackage(m.coarse, generation)
	}
	if err != nil {
		return nil, err
	}
	if pkg.Bytes > limit {
		pkg.Evict()
		return nil, fmt.Errorf("%w: %s needs %d bytes but budget is %d", ErrBudgetTooSmall, tier, pkg.Bytes, limit)
	}

	pkg.Generation = generation
	handle := &Handle{Tier: tier, Generation: generation, BudgetBytes: limit, pkg: pkg}
	m.current = handle
	m.budget = limit
	m.BuildCounts[tier]++
	m.Transitions = append(m.Transitions, Transition{
		From: from, To: tier, BudgetBytes: limit, Generation: generation,
		ResidentBytes: pkg.Bytes, Reason: reason,
	})
	return handle, nil
}

func (m *Manager) Render(handle *Handle, camera Camera) (*RenderResult, error) {
	if handle == nil {
		return nil, ErrStaleHandle
	}
	if m.current == nil || handle.Generation != m.generation || handle != m.current {
		return nil, fmt.Errorf("%w: handle gen=%d current gen=%d", ErrStaleHandle, handle.Generation, m.generation)
	}
	if handle.pkg == nil || handle.pkg.Evicted {
		return nil, fmt.Errorf("%w: package evicted", ErrStaleHandle)
	}
	result, err := handle.pkg.Render(camera)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *Manager) RenderCurrent(camera Camera) (*RenderResult, error) {
	return m.Render(m.current, camera)
}
