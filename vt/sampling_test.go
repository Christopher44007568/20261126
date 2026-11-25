package vt

import "testing"

func TestMissingPageUsesAncestorAndNeverNeighbor(t *testing.T) {
	info := DefaultInfo()
	tex := NewProceduralTexture(info)
	table := NewTable(16)

	target := PageID{Mip: 0, X: 4, Y: 5}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			neighbor := PageID{Mip: 0, X: target.X + dx, Y: target.Y + dy}
			page, err := tex.LoadPage(neighbor)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := table.Install(page); err != nil {
				t.Fatal(err)
			}
		}
	}
	root, err := tex.LoadPage(PageID{Mip: info.MaxMip, X: 0, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.Install(root); err != nil {
		t.Fatal(err)
	}

	cam := NewCameraWithViewHeight(0.5625, 0.6875, 0.125)
	frame := Render(cam, info, table, 128, 128)
	if err := frame.Verify(tex); err != nil {
		t.Fatal(err)
	}
	if frame.Metrics.ExactPixels != 0 {
		t.Fatalf("missing page produced %d exact hits", frame.Metrics.ExactPixels)
	}
	if frame.Metrics.FallbackPixels != 128*128 {
		t.Fatalf("fallback pixels = %d", frame.Metrics.FallbackPixels)
	}
	for i, pixel := range frame.Pixels {
		if pixel.Requested != target {
			t.Fatalf("pixel %d requested %s, want %s", i, pixel.Requested, target)
		}
		if pixel.Resolved != root.ID {
			t.Fatalf("pixel %d resolved %s; adjacent page substitution is forbidden", i, pixel.Resolved)
		}
	}
}

func TestDirectParentFallbackTransform(t *testing.T) {
	info := DefaultInfo()
	tex := NewProceduralTexture(info)
	table := NewTable(4)
	parentID := PageID{Mip: 1, X: 2, Y: 2}
	parent, err := tex.LoadPage(parentID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.Install(parent); err != nil {
		t.Fatal(err)
	}

	cam := NewCameraWithViewHeight(0.5625, 0.6875, 0.125)
	frame := Render(cam, info, table, 160, 160)
	if err := frame.Verify(tex); err != nil {
		t.Fatal(err)
	}
	for i, pixel := range frame.Pixels {
		if pixel.Status != StatusAncestor || pixel.Resolved != parentID || pixel.FallbackDepth != 1 {
			t.Fatalf("pixel %d = %+v, want direct parent %s", i, pixel, parentID)
		}
	}
}
