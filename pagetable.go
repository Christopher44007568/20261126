package main

import (
	"container/list"
)

// AllocEvent 描述一次物理页分配，便于镜像到“错误页表”做对照。
type AllocEvent struct {
	Vp      VP
	Slot    int
	Evicted bool
	OldVp   VP
	OldSlot int
}

// Table 是稀疏纹理的虚拟页表 + 物理驻留池。
// pte[y][x] 为物理页槽号，-1 表示无映射；owner 是反向映射。
type Table struct {
	pte         [PagesPerSide][PagesPerSide]int
	ownerVp     [PhysicalPages]VP
	ownerValid  [PhysicalPages]bool
	lru         *list.List
	node        map[VP]*list.Element
	evictedOnce map[VP]bool
}

func NewTable() *Table {
	t := &Table{
		lru:         list.New(),
		node:        make(map[VP]*list.Element),
		evictedOnce: make(map[VP]bool),
	}
	for y := 0; y < PagesPerSide; y++ {
		for x := 0; x < PagesPerSide; x++ {
			t.pte[y][x] = -1
		}
	}
	return t
}

// Residency 判断虚拟页当前的命中类型，不改变 LRU 状态。
func (t *Table) Residency(vp VP) HitKind {
	if t.pte[vp.Y][vp.X] >= 0 {
		return HitResident
	}
	if t.evictedOnce[vp] {
		return HitDangling
	}
	return HitCold
}

// Touch 标记一次驻留页访问，把该页移到 LRU 队首（最近使用）。
func (t *Table) Touch(vp VP) {
	if el, ok := t.node[vp]; ok {
		t.lru.MoveToFront(el)
	}
}

// Allocate 为虚拟页分配物理槽；池满时淘汰 LRU 队尾页。
// 返回的事件记录被淘汰者，调用方据此把权威内容重新加载进槽位。
func (t *Table) Allocate(vp VP) AllocEvent {
	ev := AllocEvent{Vp: vp}

	if slot := t.findFreeSlot(); slot >= 0 {
		ev.Slot = slot
		t.bind(vp, slot)
		t.node[vp] = t.lru.PushFront(vp)
		return ev
	}

	// 池满：淘汰最久未使用的物理页。
	back := t.lru.Back()
	oldVp := back.Value.(VP)
	oldSlot := t.pte[oldVp.Y][oldVp.X]

	// 关键正确性步骤：同时解除 PTE 与反向属主映射，
	// 否则同一 UV 会沿着旧 PTE 读到已被别人占用的物理页。
	t.pte[oldVp.Y][oldVp.X] = -1
	t.ownerValid[oldSlot] = false
	t.lru.Remove(back)
	delete(t.node, oldVp)
	t.evictedOnce[oldVp] = true

	ev.Evicted = true
	ev.OldVp = oldVp
	ev.OldSlot = oldSlot
	ev.Slot = oldSlot

	t.bind(vp, oldSlot)
	t.node[vp] = t.lru.PushFront(vp)
	return ev
}

func (t *Table) bind(vp VP, slot int) {
	t.pte[vp.Y][vp.X] = slot
	t.ownerVp[slot] = vp
	t.ownerValid[slot] = true
}

func (t *Table) findFreeSlot() int {
	for slot := 0; slot < PhysicalPages; slot++ {
		if !t.ownerValid[slot] {
			return slot
		}
	}
	return -1
}

// Mapped 报告虚拟页当前映射到的物理槽，未映射时 ok 为 false。
func (t *Table) Mapped(vp VP) (slot int, ok bool) {
	s := t.pte[vp.Y][vp.X]
	return s, s >= 0
}

// Owner 返回物理槽当前属主，槽位空闲时 ok 为 false。
func (t *Table) Owner(slot int) (vp VP, ok bool) {
	return t.ownerVp[slot], t.ownerValid[slot]
}

// EvictedOnce 报告该虚拟页是否曾经被淘汰过。
func (t *Table) EvictedOnce(vp VP) bool {
	return t.evictedOnce[vp]
}

// CheckInvariants 校验页表双向一致性与 LRU 一致性，违例时返回描述。
func (t *Table) CheckInvariants() string {
	mapped := 0
	for y := 0; y < PagesPerSide; y++ {
		for x := 0; x < PagesPerSide; x++ {
			vp := VP{X: x, Y: y}
			slot := t.pte[y][x]
			if slot < 0 {
				continue
			}
			mapped++
			if !t.ownerValid[slot] || t.ownerVp[slot] != vp {
				return "PTE 指向的物理槽属主不匹配"
			}
			if _, ok := t.node[vp]; !ok {
				return "已映射页不在 LRU 链表中"
			}
		}
	}
	for slot := 0; slot < PhysicalPages; slot++ {
		if !t.ownerValid[slot] {
			continue
		}
		vp := t.ownerVp[slot]
		if t.pte[vp.Y][vp.X] != slot {
			return "物理槽属主没有指回原 PTE"
		}
	}
	if t.lru.Len() != mapped || len(t.node) != mapped {
		return "LRU 节点数与映射数不一致"
	}
	return ""
}
