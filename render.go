package main

import (
	"image"
	"image/color"
)

// frameSize 是相机图/掩码在单个纹素维度上的纹素数。
func frameTexels() int {
	return CameraPages * PageTexels
}

// pagesInRect 按行优先顺序枚举相机视口里的虚拟页。
func pagesInRect(r Rect) []VP {
	out := make([]VP, 0, CameraPages*CameraPages)
	for y := r.Y0; y < r.Y0+CameraPages; y++ {
		for x := r.X0; x < r.X0+CameraPages; x++ {
			out = append(out, VP{X: x, Y: y})
		}
	}
	return out
}

// drawCamera 用正确页表对物理纹理采样，生成当前相机图。
func drawCamera(t *Table, phys *PhysicalTexture, r Rect) *image.RGBA {
	n := frameTexels()
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			vp := VP{X: r.X0 + sx/PageTexels, Y: r.Y0 + sy/PageTexels}
			slot, ok := t.Mapped(vp)
			if !ok {
				// 正确流程里加载阶段结束后不应出现；用洋红兜底暴露问题。
				img.Set(sx, sy, color.RGBA{R: 0xff, B: 0xff, A: 0xff})
				continue
			}
			img.Set(sx, sy, phys.Sample(slot, sx%PageTexels, sy%PageTexels))
		}
	}
	return img
}

// drawMask 把每帧加载前的命中类型渲染成悬空采样掩码。
func drawMask(kinds map[VP]HitKind, r Rect) *image.RGBA {
	n := frameTexels()
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			vp := VP{X: r.X0 + sx/PageTexels, Y: r.Y0 + sy/PageTexels}
			switch kinds[vp] {
			case HitDangling:
				img.Set(sx, sy, colorDangling)
			case HitCold:
				img.Set(sx, sy, colorCold)
			default:
				img.Set(sx, sy, colorResident)
			}
		}
	}
	return img
}
