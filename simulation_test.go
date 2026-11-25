package main

import "testing"

func TestThreeFramesReloadEvictedContent(t *testing.T) {
	sim := NewSimulator()
	frames := []FrameSpec{
		{Index: 1, Rect: Rect{X0: 0, Y0: 0}},
		{Index: 2, Rect: Rect{X0: 4, Y0: 4}},
		{Index: 3, Rect: Rect{X0: 0, Y0: 0}},
	}

	var results []FrameResult
	for _, spec := range frames {
		res := sim.RunFrame(spec)
		if msg := sim.Table.CheckInvariants(); msg != "" {
			t.Fatalf("第 %d 帧不变量异常: %s", spec.Index, msg)
		}
		if err := verifyCameraMatchesCanonical(sim, spec.Rect); err != nil {
			t.Fatalf("第 %d 帧相机内容错误: %v", spec.Index, err)
		}
		results = append(results, res)
	}

	third := results[2]
	if dangling := countKind(third.Kinds, HitDangling); dangling != 2 {
		t.Fatalf("第 3 帧应有 2 个悬空页，得到 %d", dangling)
	}
	if resident := countKind(third.Kinds, HitResident); resident != 2 {
		t.Fatalf("第 3 帧应有 2 个驻留命中，得到 %d", resident)
	}
	if third.Hazard != 2*PageTexels*PageTexels {
		t.Fatalf("错误表危害应为 2 页纹素数，得到 %d", third.Hazard)
	}

	// 重载后旧 UV 必须解析到内容匹配的物理槽。
	vp := VP{X: 0, Y: 0}
	slot, ok := sim.Table.Mapped(vp)
	if !ok {
		t.Fatalf("回视野后 %v 仍未映射", vp)
	}
	for i := 0; i < PageTexels; i++ {
		got := sim.Phys.Sample(slot, i, i)
		want := canonicalTexel(vp.X*PageTexels+i, vp.Y*PageTexels+i)
		if got != want {
			t.Fatalf("重载后 %v 纹素 %d 读到 %v，应为 %v", vp, i, got, want)
		}
	}
}

func TestStaleTableReadsOtherTexture(t *testing.T) {
	sim := NewSimulator()
	sim.RunFrame(FrameSpec{Index: 1, Rect: Rect{X0: 0, Y0: 0}})
	sim.RunFrame(FrameSpec{Index: 2, Rect: Rect{X0: 4, Y0: 4}})

	// 帧 2 淘汰 (0,0) 后，错误表仍让其指向已被 (4,5) 占用的 P0。
	vp := VP{X: 0, Y: 0}
	slot := sim.Stale.StaleSlot(vp)
	owner, ok := sim.Table.Owner(slot)
	if !ok || owner == vp {
		t.Fatalf("悬空 PTE 应指向他人属主，slot=%d owner=%v ok=%v", slot, owner, ok)
	}

	res := sim.RunFrame(FrameSpec{Index: 3, Rect: Rect{X0: 0, Y0: 0}})
	if res.Hazard == 0 {
		t.Fatal("重载前应能观测到错误表读到其他纹理")
	}
}
