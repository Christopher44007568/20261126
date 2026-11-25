package aliaslab

import "testing"

func TestFindFirstContaminationUsesRowMajorConnectedComponent(t *testing.T) {
	ref := NewImage(10, 8)
	bad := ref.Clone()
	bad.Set(2, 4, RGB{255, 0, 0})
	bad.Set(7, 2, RGB{0, 255, 0})
	bad.Set(8, 2, RGB{0, 255, 0})
	bad.Set(8, 3, RGB{0, 255, 0})

	got, ok := FindFirstContamination(ref, bad)
	if !ok {
		t.Fatal("expected contamination")
	}
	if got.FirstX != 7 || got.FirstY != 2 {
		t.Fatalf("first pixel=(%d,%d), want (7,2)", got.FirstX, got.FirstY)
	}
	want := Bounds{MinX: 7, MinY: 2, MaxX: 8, MaxY: 3}
	if got.Bounds != want {
		t.Fatalf("bounds=%+v, want %+v", got.Bounds, want)
	}
	if got.RegionPixels != 3 {
		t.Fatalf("region pixels=%d, want 3", got.RegionPixels)
	}
	if got.TotalPixels != 4 {
		t.Fatalf("total pixels=%d, want 4", got.TotalPixels)
	}
}

func TestHighlightFirstRegionMarksBounds(t *testing.T) {
	ref := NewImage(40, 30)
	bad := ref.Clone()
	for y := 10; y <= 13; y++ {
		for x := 12; x <= 17; x++ {
			bad.Set(x, y, RGB{1, 2, 3})
		}
	}
	c, ok := FindFirstContamination(ref, bad)
	if !ok {
		t.Fatal("expected contamination")
	}
	marked := bad.Clone()
	HighlightFirstRegion(marked, c)

	if marked.At(c.FirstX, c.FirstY) != (RGB{230, 35, 35}) {
		t.Fatal("first contaminated pixel was not replaced by the red crosshair")
	}
	outer := Bounds{
		MinX: c.Bounds.MinX - 5, MinY: c.Bounds.MinY - 5,
		MaxX: c.Bounds.MaxX + 5, MaxY: c.Bounds.MaxY + 5,
	}
	if marked.At(outer.MinX, outer.MinY) != (RGB{5, 8, 14}) {
		t.Fatal("outer black frame missing")
	}
	inner := Bounds{
		MinX: c.Bounds.MinX - 2, MinY: c.Bounds.MinY - 2,
		MaxX: c.Bounds.MaxX + 2, MaxY: c.Bounds.MaxY + 2,
	}
	if marked.At(inner.MinX, inner.MinY) != (RGB{255, 212, 59}) {
		t.Fatal("inner yellow frame missing")
	}
}
