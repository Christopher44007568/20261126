package main

import "testing"

func TestAllocateFillsFreeSlotsFirst(t *testing.T) {
	tab := NewTable()
	for i := 0; i < PhysicalPages; i++ {
		vp := VP{X: i, Y: 0}
		ev := tab.Allocate(vp)
		if ev.Evicted || ev.Slot != i {
			t.Fatalf("第 %d 次分配应落入空槽 P%d，得到 %+v", i, i, ev)
		}
	}
	if msg := tab.CheckInvariants(); msg != "" {
		t.Fatalf("池占满后不变量异常: %s", msg)
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	tab := NewTable()
	first := VP{X: 0, Y: 0}
	others := []VP{{X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
		{X: 4, Y: 0}, {X: 5, Y: 0}}

	tab.Allocate(first)
	for _, vp := range others {
		tab.Allocate(vp)
	}
	// first 是最久未使用页；触碰 others[1] 后它应变成最新。
	tab.Touch(others[1])

	ev := tab.Allocate(VP{X: 0, Y: 5})
	if !ev.Evicted || ev.OldVp != first {
		t.Fatalf("应淘汰最久未使用的 %v，得到事件 %+v", first, ev)
	}
	if slot, ok := tab.Mapped(first); ok {
		t.Fatalf("被淘汰页 %v 仍映射到 P%d", first, slot)
	}
	if tab.Residency(first) != HitDangling {
		t.Fatalf("被淘汰页应标记为悬空")
	}
	if msg := tab.CheckInvariants(); msg != "" {
		t.Fatalf("淘汰后不变量异常: %s", msg)
	}
}
