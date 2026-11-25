package main

import "image/color"

// 虚拟纹理被切成 PagesPerSide x PagesPerSide 个虚拟页。
const PagesPerSide = 6

// 每个虚拟页内部的纹素数量（页为正方形）。
const PageTexels = 16

// 物理纹理（驻留池）中可用的物理页数量。
const PhysicalPages = 6

// 相机每帧可见的页数（方形视口）。
const CameraPages = 2

// 输出图像时每个纹素放大的像素数，便于肉眼观察。
const OutputScale = 4

// 状态色块中单个虚拟页的边长（像素）。
const StateBlock = 28

// VP 标识一个虚拟页坐标。
type VP struct {
	X int
	Y int
}

// Rect 用页坐标表示方形相机视口 [X0, X0+CameraPages) x [Y0, Y0+CameraPages)。
type Rect struct {
	X0 int
	Y0 int
}

// 命中类型，用于悬空采样掩码。
type HitKind int

const (
	HitCold     HitKind = iota // 从未驻留，需要首次加载
	HitResident                // 当前已驻留，直接命中物理页
	HitDangling                // 曾驻留但物理页已被回收（悬空）
)

// RGB 掩码颜色（不透明）。
var (
	colorCold     = color.RGBA{R: 0xff, G: 0xc4, B: 0x00, A: 0xff} // 黄：冷缺失
	colorResident = color.RGBA{R: 0x1b, G: 0x9e, B: 0x42, A: 0xff} // 绿：驻留命中
	colorDangling = color.RGBA{R: 0xd6, G: 0x28, B: 0x28, A: 0xff} // 红：悬空采样
)
