package main

import "image/color"

// hashVP 为虚拟页生成稳定的非零 32 位值，避免依赖随机数。
func hashVP(vp VP) uint32 {
	h := uint32(2166136261)
	h ^= uint32(vp.X) + 1
	h *= 16777619
	h ^= uint32(vp.Y) + 1
	h *= 16777619
	h ^= uint32(vp.X*73 + vp.Y*19)
	h *= 16777619
	return h
}

// pageBaseColor 返回虚拟页的唯一基色，三通道都保留足够亮度。
func pageBaseColor(vp VP) color.RGBA {
	h := hashVP(vp)
	return color.RGBA{
		R: uint8(64 + h%176),
		G: uint8(64 + (h>>8)%176),
		B: uint8(64 + (h>>16)%176),
		A: 0xff,
	}
}

// canonicalTexel 是虚拟纹理在 (x, y) 处的权威纹素。
// 物理页加载时必须用它填充，保证重分配后内容与虚拟坐标一致，
// 而不是保留上一个占用者的纹理。
func canonicalTexel(x, y int) color.RGBA {
	vp := VP{X: x / PageTexels, Y: y / PageTexels}
	lx := x % PageTexels
	ly := y % PageTexels
	base := pageBaseColor(vp)

	// 页边画一圈深色边框，方便在相机图里辨认页边界。
	if lx == 0 || ly == 0 || lx == PageTexels-1 || ly == PageTexels-1 {
		return color.RGBA{R: base.R / 5, G: base.G / 5, B: base.B / 5, A: 0xff}
	}

	// 页内棋盘图案：同一虚拟页的图案固定，不同页不会混淆。
	if (lx/4+ly/4)%2 == 0 {
		return base
	}
	return color.RGBA{
		R: clampByte(int(base.R) + 34),
		G: clampByte(int(base.G) + 34),
		B: clampByte(int(base.B) + 34),
		A: 0xff,
	}
}

// pageCode 返回页左上角的识别码（0..9），用于在放大图上区分页面。
func pageCode(vp VP) int {
	return int(hashVP(vp) % 10)
}

func clampByte(v int) uint8 {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return uint8(v)
}
