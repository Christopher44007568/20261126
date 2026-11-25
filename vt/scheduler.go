package vt

import "sync"

// Scheduler turns UV feedback histograms into de-duplicated page requests.
type Scheduler struct {
	info       Info
	mu         sync.Mutex
	generation uint64
	inflight   map[PageID]LoadRequest
	wanted     map[PageID]struct{}
}

func NewScheduler(info Info) *Scheduler {
	return &Scheduler{
		info:     info,
		inflight: make(map[PageID]LoadRequest),
		wanted:   make(map[PageID]struct{}),
	}
}

// Plan replaces the current wanted set and returns newly scheduled requests.
// Every missing ancestor is requested too, ensuring a legal lower-mip fallback.
func (s *Scheduler) Plan(fb *FeedbackBuffer, table *Table) []LoadRequest {
	if fb == nil {
		return nil
	}
	snapshot := table.Snapshot()

	s.mu.Lock()
	s.generation++
	generation := s.generation
	s.wanted = make(map[PageID]struct{})

	var requests []LoadRequest
	seen := make(map[PageID]struct{})
	for exact, count := range fb.Counts {
		for depth, id := range s.info.Chain(exact) {
			s.wanted[id] = struct{}{}
			if _, ok := snapshot[id]; ok {
				continue
			}
			if _, ok := s.inflight[id]; ok {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			reason := "feedback"
			if depth > 0 {
				reason = "ancestor-fallback"
			}
			req := LoadRequest{
				ID:         id,
				Priority:   id.Mip*1_000_000 + count*16 + (1024 - depth),
				Generation: generation,
				Reason:     reason,
			}
			s.inflight[id] = req
			requests = append(requests, req)
		}
	}
	s.mu.Unlock()

	sortRequests(requests)
	return requests
}

func sortRequests(requests []LoadRequest) {
	for i := 1; i < len(requests); i++ {
		for j := i; j > 0 && requests[j-1].Priority < requests[j].Priority; j-- {
			requests[j-1], requests[j] = requests[j], requests[j-1]
		}
	}
}

// Finish removes a load from the inflight set.
func (s *Scheduler) Finish(id PageID) {
	s.mu.Lock()
	delete(s.inflight, id)
	s.mu.Unlock()
}

// Accept reports whether a loaded page is still useful for the newest view.
func (s *Scheduler) Accept(id PageID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.wanted[id]
	return ok
}

func (s *Scheduler) InflightCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inflight)
}

func (s *Scheduler) WantedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.wanted)
}

// WantedIDs returns a copy of every exact page and fallback ancestor wanted
// for the newest camera frame.
func (s *Scheduler) WantedIDs() []PageID {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]PageID, 0, len(s.wanted))
	for id := range s.wanted {
		ids = append(ids, id)
	}
	return ids
}

func (s *Scheduler) Generation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation
}

// FeedbackCoverage computes exact-page residency over feedback texels.
func FeedbackCoverage(fb *FeedbackBuffer, snapshot map[PageID]SnapshotEntry) (total, exact int, rate float64) {
	if fb == nil {
		return 0, 0, 0
	}
	total = len(fb.Samples)
	for _, sample := range fb.Samples {
		if _, ok := snapshot[sample.Page]; ok {
			exact++
		}
	}
	if total > 0 {
		rate = float64(total-exact) / float64(total)
	}
	return total, exact, rate
}
