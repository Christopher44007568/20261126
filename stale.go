package main

// StaleTable 镜像“忘记在淘汰时清 PTE”的错误实现。
// 它只记录分配写入，从不删除旧映射，因此重分配后旧虚拟页
// 仍会沿着悬空 PTE 采样到新属主的物理纹理。
type StaleTable struct {
	pte [PagesPerSide][PagesPerSide]int
}

func NewStaleTable() *StaleTable {
	s := &StaleTable{}
	for y := 0; y < PagesPerSide; y++ {
		for x := 0; x < PagesPerSide; x++ {
			s.pte[y][x] = -1
		}
	}
	return s
}

// MirrorApply 复刻正确页表的一次分配：错误表只写新映射、不清旧映射。
func (s *StaleTable) MirrorApply(ev AllocEvent) {
	s.pte[ev.Vp.Y][ev.Vp.X] = ev.Slot
}

// StaleSlot 返回悬空虚拟页在错误表中仍指向的物理槽。
func (s *StaleTable) StaleSlot(vp VP) int {
	return s.pte[vp.Y][vp.X]
}

// HazardPixels 对照正确页表逐纹素检查：对当前未映射的虚拟页，
// 错误表的悬空 PTE 会读到哪个属主，并统计读到“别人纹理”的纹素数。
func (s *StaleTable) HazardPixels(t *Table, phys *PhysicalTexture, rect Rect) int {
	hazard := 0
	for sy := 0; sy < CameraPages*PageTexels; sy++ {
		for sx := 0; sx < CameraPages*PageTexels; sx++ {
			vp := VP{
				X: rect.X0 + sx/PageTexels,
				Y: rect.Y0 + sy/PageTexels,
			}
			if _, mapped := t.Mapped(vp); mapped {
				continue
			}
			slot := s.StaleSlot(vp)
			if slot < 0 {
				continue
			}
			if owner, ok := t.Owner(slot); ok && owner != vp {
				// 悬空 PTE 指向的槽已属他人：比较采样纹素与权威纹素。
				got := phys.Sample(slot, sx%PageTexels, sy%PageTexels)
				want := canonicalTexel(vp.X*PageTexels+sx%PageTexels,
					vp.Y*PageTexels+sy%PageTexels)
				if got != want {
					hazard++
				}
			}
		}
	}
	return hazard
}
