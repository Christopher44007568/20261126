package vt

import (
	"context"
	"testing"
	"time"
)

func TestAsyncLoadMoveAndReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := DefaultInfo()
	tex := NewProceduralTexture(info)
	table := NewTable(12)
	loader := NewAsyncLoader(ctx, 4, 64, func(loadCtx context.Context, id PageID) (*Page, error) {
		return tex.LoadPage(id)
	})
	defer loader.Close()
	engine := NewEngine(info, table, NewScheduler(info), loader, 8)

	camA := NewCameraWithViewHeight(0.3125, 0.25, 0.25)
	engine.Update(camA, 300, 200)
	if err := engine.WaitUntil(ctx, engine.Idle); err != nil {
		t.Fatal(err)
	}
	frameA := engine.Render(camA, 300, 200)
	if err := frameA.Verify(tex); err != nil {
		t.Fatal(err)
	}
	if frameA.Metrics.MissRate != 0 {
		t.Fatalf("settled miss rate = %f", frameA.Metrics.MissRate)
	}

	camB := NewCameraWithViewHeight(0.6875, 0.75, 0.25)
	engine.Update(camB, 300, 200)
	if err := engine.WaitUntil(ctx, engine.Idle); err != nil {
		t.Fatal(err)
	}
	frameB := engine.Render(camB, 300, 200)
	if err := frameB.Verify(tex); err != nil {
		t.Fatal(err)
	}
	if frameB.Metrics.MissRate != 0 {
		t.Fatalf("moved settled miss rate = %f", frameB.Metrics.MissRate)
	}
	if table.Count() > table.Capacity() {
		t.Fatalf("resident %d exceeds capacity", table.Count())
	}
	if table.EvictionCount() == 0 {
		t.Fatal("expected LRU replacements after moving across the wall")
	}
}

func TestSchedulerRejectsStaleViewportPages(t *testing.T) {
	info := DefaultInfo()
	scheduler := NewScheduler(info)
	emptyTable := NewTable(12)

	oldPage := PageID{Mip: 0, X: 1, Y: 1}
	newPage := PageID{Mip: 0, X: 6, Y: 6}
	scheduler.Plan(&FeedbackBuffer{Counts: map[PageID]int{oldPage: 1}}, emptyTable)
	scheduler.Plan(&FeedbackBuffer{Counts: map[PageID]int{newPage: 1}}, emptyTable)

	if scheduler.Accept(oldPage) {
		t.Fatal("old off-screen page is still accepted after viewport generation change")
	}
	if !scheduler.Accept(newPage) {
		t.Fatal("new visible page should be accepted")
	}
	root := PageID{Mip: info.MaxMip, X: 0, Y: 0}
	if !scheduler.Accept(root) {
		t.Fatal("shared root ancestor should remain accepted")
	}
}
