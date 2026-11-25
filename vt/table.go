package vt

import (
	"container/list"
	"errors"
	"image"
	"sort"
	"sync"
)

var (
	ErrInvalidPage = errors.New("invalid virtual page")
	ErrCacheFull   = errors.New("page cache is full")
	ErrNoPage      = errors.New("page is not resident")
)

// ResidentPage is a page-table entry backed by one physical cache slot.
type ResidentPage struct {
	ID       PageID
	Slot     int
	Image    *image.RGBA
	lastUsed uint64
}

// SnapshotEntry is an immutable view used by one rendered frame.
type SnapshotEntry struct {
	Slot  int
	Image *image.RGBA
}

// Table is a locked page table and LRU physical page cache.
type Table struct {
	mu       sync.Mutex
	capacity int
	pages    map[PageID]*list.Element
	lru      *list.List
	free     []int
	clock    uint64

	Installs  int
	Evictions int
}

// NewTable creates capacity physical slots.
func NewTable(capacity int) *Table {
	if capacity < 1 {
		capacity = 1
	}
	free := make([]int, capacity)
	for i := range free {
		free[i] = capacity - i - 1
	}
	return &Table{
		capacity: capacity,
		pages:    make(map[PageID]*list.Element, capacity),
		lru:      list.New(),
		free:     free,
	}
}

func (t *Table) Capacity() int { return t.capacity }

func (t *Table) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lru.Len()
}

func (t *Table) EvictionCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Evictions
}

// Has reports whether the exact page is resident.
func (t *Table) Has(id PageID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	el, ok := t.pages[id]
	if !ok {
		return false
	}
	t.lru.MoveToFront(el)
	t.clock++
	el.Value.(*ResidentPage).lastUsed = t.clock
	return true
}

// Snapshot returns all resident pages without exposing the LRU pointers.
func (t *Table) Snapshot() map[PageID]SnapshotEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[PageID]SnapshotEntry, len(t.pages))
	for id, el := range t.pages {
		page := el.Value.(*ResidentPage)
		out[id] = SnapshotEntry{Slot: page.Slot, Image: page.Image}
	}
	return out
}

// Touch marks pages used by the completed camera frame as recently used.
func (t *Table) Touch(ids []PageID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clock++
	for _, id := range ids {
		if el, ok := t.pages[id]; ok {
			t.lru.MoveToFront(el)
			el.Value.(*ResidentPage).lastUsed = t.clock
		}
	}
}

// Install places a loaded page in its ID's table entry and a physical slot.
// A full cache evicts the least recently used page. The returned page, if any,
// is only a notification; its slot has already been freed.
func (t *Table) Install(page *Page) (*ResidentPage, error) {
	if page == nil || page.Image == nil {
		return nil, ErrInvalidPage
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clock++
	t.Installs++

	if el, ok := t.pages[page.ID]; ok {
		resident := el.Value.(*ResidentPage)
		resident.Image = page.Image
		resident.lastUsed = t.clock
		t.lru.MoveToFront(el)
		return nil, nil
	}

	var evicted *ResidentPage
	if len(t.free) == 0 {
		oldest := t.lru.Back()
		if oldest == nil {
			return nil, ErrCacheFull
		}
		evicted = oldest.Value.(*ResidentPage)
		t.lru.Remove(oldest)
		delete(t.pages, evicted.ID)
		t.free = append(t.free, evicted.Slot)
		t.Evictions++
	}

	slot := t.free[len(t.free)-1]
	t.free = t.free[:len(t.free)-1]
	resident := &ResidentPage{
		ID:       page.ID,
		Slot:     slot,
		Image:    page.Image,
		lastUsed: t.clock,
	}
	el := t.lru.PushFront(resident)
	t.pages[page.ID] = el
	if evicted != nil {
		evictedCopy := *evicted
		return &evictedCopy, nil
	}
	return nil, nil
}

// IDs returns resident IDs sorted by mip and coordinates.
func (t *Table) IDs() []PageID {
	t.mu.Lock()
	defer t.mu.Unlock()
	ids := make([]PageID, 0, len(t.pages))
	for id := range t.pages {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Mip != ids[j].Mip {
			return ids[i].Mip < ids[j].Mip
		}
		if ids[i].Y != ids[j].Y {
			return ids[i].Y < ids[j].Y
		}
		return ids[i].X < ids[j].X
	})
	return ids
}

// Resolve finds the exact page or the nearest resident ancestor. It never
// substitutes an adjacent page at the same mip.
func Resolve(
	snapshot map[PageID]SnapshotEntry,
	info Info,
	id PageID,
) (PageID, SnapshotEntry, int, bool) {
	chain := info.Chain(id)
	for depth, candidate := range chain {
		if entry, ok := snapshot[candidate]; ok {
			return candidate, entry, depth, true
		}
	}
	var zero SnapshotEntry
	return id, zero, len(chain), false
}

// SortedSnapshot is convenient for diagnostics and visualization.
func SortedSnapshot(snapshot map[PageID]SnapshotEntry) []struct {
	ID    PageID
	Entry SnapshotEntry
} {
	out := make([]struct {
		ID    PageID
		Entry SnapshotEntry
	}, 0, len(snapshot))
	for id, entry := range snapshot {
		out = append(out, struct {
			ID    PageID
			Entry SnapshotEntry
		}{id, entry})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].ID, out[j].ID
		if a.Mip != b.Mip {
			return a.Mip < b.Mip
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	return out
}
