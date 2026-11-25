package main

import (
	"fmt"
	"image"
)

// FrameSpec 描述一帧的相机视口与序号（1 基）。
type FrameSpec struct {
	Index int
	Rect  Rect
}

// FrameResult 汇总一帧产物，供 main 输出图像与日志。
type FrameResult struct {
	Spec        FrameSpec
	Kinds       map[VP]HitKind
	Camera      *image.RGBA
	Mask        *image.RGBA
	StaleCamera *image.RGBA
	StateGrid   *image.RGBA
	Events      []AllocEvent
	Hazard      int
}

// Simulator 持有跨帧状态：正确页表、物理池与错误镜像表。
type Simulator struct {
	Table *Table
	Phys  *PhysicalTexture
	Stale *StaleTable
}

func NewSimulator() *Simulator {
	return &Simulator{
		Table: NewTable(),
		Phys:  &PhysicalTexture{},
		Stale: NewStaleTable(),
	}
}

// RunFrame 执行一帧：解析命中 → LRU 触碰 → 淘汰/重分配/重载 → 渲染。
func (s *Simulator) RunFrame(spec FrameSpec) FrameResult {
	visible := pagesInRect(spec.Rect)
	kinds := make(map[VP]HitKind, len(visible))

	// 阶段 1：在任何分配之前记录命中类型，作为悬空采样掩码依据。
	for _, vp := range visible {
		kinds[vp] = s.Table.Residency(vp)
	}
	mask := drawMask(kinds, spec.Rect)

	// 阶段 1b：在重载之前，用“忘记清 PTE”的错误表采样，
	// 此时悬空 UV 会读到已被别人占用的物理页。
	staleCamera := s.drawStaleCamera(spec.Rect)
	hazard := s.Stale.HazardPixels(s.Table, s.Phys, spec.Rect)

	// 阶段 2：驻留页先 touch，保证它们在本帧淘汰决策中被视为最近使用。
	for _, vp := range visible {
		if kinds[vp] == HitResident {
			s.Table.Touch(vp)
		}
	}

	// 阶段 3：缺失页逐个分配；池满则 LRU 淘汰并按新属主权威内容重载。
	var events []AllocEvent
	for _, vp := range visible {
		if kinds[vp] == HitResident {
			continue
		}
		ev := s.Table.Allocate(vp)
		s.Phys.Load(vp, ev.Slot)
		s.Stale.MirrorApply(ev)
		events = append(events, ev)
	}

	return FrameResult{
		Spec:        spec,
		Kinds:       kinds,
		Camera:      drawCamera(s.Table, s.Phys, spec.Rect),
		Mask:        mask,
		StaleCamera: staleCamera,
		StateGrid:   drawStateGrid(s.Table),
		Events:      events,
		Hazard:      hazard,
	}
}

// drawStaleCamera 用错误页表采样：未映射页沿悬空 PTE 读到当前属主纹理。
func (s *Simulator) drawStaleCamera(r Rect) *image.RGBA {
	n := frameTexels()
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			vp := VP{X: r.X0 + sx/PageTexels, Y: r.Y0 + sy/PageTexels}
			lx, ly := sx%PageTexels, sy%PageTexels
			if slot, ok := s.Table.Mapped(vp); ok {
				img.Set(sx, sy, s.Phys.Sample(slot, lx, ly))
				continue
			}
			slot := s.Stale.StaleSlot(vp)
			if slot < 0 {
				img.Set(sx, sy, colorCold)
				continue
			}
			img.Set(sx, sy, s.Phys.Sample(slot, lx, ly))
		}
	}
	return img
}

// DescribeEvent 把分配事件格式化成可读日志行。
func DescribeEvent(ev AllocEvent) string {
	if !ev.Evicted {
		return fmt.Sprintf("  分配: 虚拟页(%d,%d) -> 空物理槽 P%d（首次加载）",
			ev.Vp.X, ev.Vp.Y, ev.Slot)
	}
	return fmt.Sprintf(
		"  淘汰: 物理槽 P%d 的旧页(%d,%d) 被回收；新页(%d,%d) 重载进 P%d",
		ev.OldSlot, ev.OldVp.X, ev.OldVp.Y, ev.Vp.X, ev.Vp.Y, ev.Slot)
}
