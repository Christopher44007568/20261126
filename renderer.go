package main

// 三种加载状态对应的叠加色：未加载=红，加载中=琥珀，完整=绿。
var statusColor = map[Status]Color{
	StatusUnloaded: RGB(0xef, 0x44, 0x44),
	StatusLoading:  RGB(0xf5, 0x9e, 0x0b),
	StatusLoaded:   RGB(0x34, 0xd3, 0x99),
}

const overlayAlpha = 36 // 约 14% 不透明度的状态色叠加

// Renderer 无像素缓存：每次 Render 都返回一张全新画布，
// 已卸载区域只能画出占位内容，物理上无法复用旧像素。
type Renderer struct {
	loader *Loader
}

func NewRenderer(l *Loader) *Renderer { return &Renderer{loader: l} }

// Render 遍历场景树，为每个叶子子树按当前加载状态绘制一帧。
func (r *Renderer) Render(root *Node, title string) *Image {
	im := NewImage(FrameW, FrameH)
	im.SolidRect(0, 0, FrameW, FrameH, RGB(0x10, 0x12, 0x18))
	drawHeader(im, title)
	for _, leaf := range root.Leaves() {
		r.drawSubtree(im, leaf)
	}
	return im
}

func drawHeader(im *Image, title string) {
	im.DrawText(Margin, 4, 1, title, RGB(0xe5, 0xe7, 0xeb))
}

func (r *Renderer) drawSubtree(im *Image, n *Node) {
	rc := n.Rect
	im.SolidRect(rc.X, rc.Y, rc.W, rc.H, RGB(0x1b, 0x1e, 0x28))
	status := r.loader.Status(n.ID)

	switch status {
	case StatusLoaded:
		drawFullContent(im, rc, r.loader.Cached(n.ID))
	case StatusLoading:
		drawLoading(im, rc)
	case StatusUnloaded:
		drawUnloaded(im, rc)
	}

	// 状态色：整子树半透明叠加 + 醒目描边。
	sc := statusColor[status]
	im.FillRect(rc.X, rc.Y, rc.W, rc.H, Color{sc.R, sc.G, sc.B, overlayAlpha})
	im.Frame(rc.X, rc.Y, rc.W, rc.H, sc)
	im.Frame(rc.X+1, rc.Y+1, rc.W-2, rc.H-2, Color{sc.R, sc.G, sc.B, 120})
	drawStatusBar(im, rc, n, status, sc)
}

func drawStatusBar(im *Image, rc Rect, n *Node, status Status, sc Color) {
	im.SolidRect(rc.X, rc.Y, rc.W, 16, RGB(0x0c, 0x0e, 0x14))
	im.FillRect(rc.X, rc.Y, 4, 16, sc)
	im.DrawText(rc.X+9, rc.Y+4, 1, n.ID+" "+status.String(), RGB(0xf3, 0xf4, 0xf6))
}

// drawUnloaded 明确绘制“内容尚未加载”的占位画面。
func drawUnloaded(im *Image, rc Rect) {
	drawDiagonalHatch(im, rc, RGB(0x3a, 0x20, 0x24))
	line := "NOT LOADED"
	w := TextWidth(line, 2)
	im.DrawText(rc.X+(rc.W-w)/2, rc.Y+70, 2, line, RGB(0xf8, 0x71, 0x71))
	hint := "WAITING FETCH"
	hw := TextWidth(hint, 1)
	im.DrawText(rc.X+(rc.W-hw)/2, rc.Y+100, 1, hint, RGB(0xca, 0xa3, 0xa3))
}

// drawLoading 绘制加载中占位：扫描线、进度槽与脉冲提示。
func drawLoading(im *Image, rc Rect) {
	for y := rc.Y + 16; y < rc.Y+rc.H; y += 4 {
		im.SolidRect(rc.X, y, rc.W, 1, Color{0xf5, 0x9e, 0x0b, 40})
	}
	line := "LOADING"
	w := TextWidth(line, 2)
	im.DrawText(rc.X+(rc.W-w)/2, rc.Y+70, 2, line, RGB(0xfd, 0xba, 0x74))

	barX, barY, barW, barH := rc.X+24, rc.Y+110, rc.W-48, 12
	im.SolidRect(barX, barY, barW, barH, RGB(0x2a, 0x24, 0x16))
	im.Frame(barX, barY, barW, barH, RGB(0x9a, 0x6a, 0x1a))
	im.SolidRect(barX+2, barY+2, (barW-4)*40/100, barH-4, RGB(0xf5, 0x9e, 0x0b))
}
