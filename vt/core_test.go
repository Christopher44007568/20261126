package vt

import "testing"

func TestPageGeometryAndAncestors(t *testing.T) {
	info := DefaultInfo()
	if info.PagesPerSide != 8 || info.MaxMip != 3 {
		t.Fatalf("unexpected default geometry: %+v", info)
	}

	id := info.PageAtUV(0.51, 0.63, 0)
	if id != (PageID{Mip: 0, X: 4, Y: 5}) {
		t.Fatalf("PageAtUV = %v", id)
	}
	parent, ok := info.Parent(id)
	if !ok || parent != (PageID{Mip: 1, X: 2, Y: 2}) {
		t.Fatalf("parent = %v, ok=%v", parent, ok)
	}
	root := info.Ancestor(id, info.MaxMip)
	if root != (PageID{Mip: 3, X: 0, Y: 0}) {
		t.Fatalf("root ancestor = %v", root)
	}
}

func TestFeedbackContainsExactlyAlignedVisiblePages(t *testing.T) {
	info := DefaultInfo()
	cam := NewCameraWithViewHeight(0.3125, 0.25, 0.25)
	fb := BuildFeedback(cam, 600, 400, 8, info)

	if len(fb.Counts) != 6 {
		t.Fatalf("visible pages = %d, want 6: %v", len(fb.Counts), fb.Counts)
	}
	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			id := PageID{Mip: 0, X: x, Y: y}
			if fb.Counts[id] == 0 {
				t.Fatalf("missing feedback page %s", id)
			}
		}
	}
}

func TestLRUEvictionKeepsTouchedPage(t *testing.T) {
	info := DefaultInfo()
	tex := NewProceduralTexture(info)
	table := NewTable(2)

	pageA, err := tex.LoadPage(PageID{Mip: 0, X: 0, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	pageB, err := tex.LoadPage(PageID{Mip: 0, X: 1, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	pageC, err := tex.LoadPage(PageID{Mip: 0, X: 2, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.Install(pageA); err != nil {
		t.Fatal(err)
	}
	if _, err := table.Install(pageB); err != nil {
		t.Fatal(err)
	}
	if !table.Has(pageA.ID) {
		t.Fatal("page A should be resident")
	}
	evicted, err := table.Install(pageC)
	if err != nil {
		t.Fatal(err)
	}
	if evicted == nil || evicted.ID != pageB.ID {
		t.Fatalf("evicted = %v, want page B", evicted)
	}
	if !table.Has(pageA.ID) || !table.Has(pageC.ID) || table.Has(pageB.ID) {
		t.Fatal("LRU state incorrect after replacement")
	}
}
