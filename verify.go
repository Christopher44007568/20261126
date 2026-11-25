package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
)

// imageImage 是输出图像的最小接口约束。
type imageImage interface {
	image.Image
}

func ioMultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}

// verifyCameraMatchesCanonical 逐纹素确认相机图等于虚拟纹理权威内容。
// 这是“重分配后不会读到另一块纹理”的硬性保证。
func verifyCameraMatchesCanonical(sim *Simulator, r Rect) error {
	n := frameTexels()
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			globalX := r.X0*PageTexels + sx
			globalY := r.Y0*PageTexels + sy
			got, _ := sim.runFramePixel(r, sx, sy)
			want := canonicalTexel(globalX, globalY)
			if got != want {
				return fmt.Errorf("纹素(%d,%d) 读到 %v，权威值应为 %v",
					globalX, globalY, got, want)
			}
		}
	}
	return nil
}

// runFramePixel 通过正确页表采样单个相机纹素。
func (s *Simulator) runFramePixel(r Rect, sx, sy int) (colorish, bool) {
	vp := VP{X: r.X0 + sx/PageTexels, Y: r.Y0 + sy/PageTexels}
	slot, ok := s.Table.Mapped(vp)
	if !ok {
		return colorish{}, false
	}
	return s.Phys.Sample(slot, sx%PageTexels, sy%PageTexels), true
}

// colorish 是 RGBA 纹素的别名，便于比较。
type colorish = color.RGBA
