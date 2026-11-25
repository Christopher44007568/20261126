package main

import "image/color"

// PhysicalTexture 是驻留池，逐槽保存当前属主与实际纹素。
// 槽位被重新分配时，纹素会按新属主的权威内容整体重写。
type PhysicalTexture struct {
	texels [PhysicalPages][PageTexels * PageTexels]color.RGBA
}

// Load 用虚拟页的权威内容覆盖整个物理槽，绝不保留旧属主数据。
func (p *PhysicalTexture) Load(vp VP, slot int) {
	for ly := 0; ly < PageTexels; ly++ {
		for lx := 0; lx < PageTexels; lx++ {
			globalX := vp.X*PageTexels + lx
			globalY := vp.Y*PageTexels + ly
			p.texels[slot][ly*PageTexels+lx] = canonicalTexel(globalX, globalY)
		}
	}
}

// Sample 取物理槽内局部坐标的纹素。
func (p *PhysicalTexture) Sample(slot, lx, ly int) color.RGBA {
	return p.texels[slot][ly*PageTexels+lx]
}
