package main

import (
	"image"
	"image/color"
)

var (
	stateMapped   = color.RGBA{R: 0xf4, G: 0xf4, B: 0xf4, A: 0xff} // 白：已映射
	stateCold     = color.RGBA{R: 0x33, G: 0x33, B: 0x3b, A: 0xff} // 深灰：从未加载
	stateEvicted  = color.RGBA{R: 0xd6, G: 0x28, B: 0x28, A: 0xff} // 红：曾被淘汰
	stateFree     = color.RGBA{R: 0x16, G: 0x16, B: 0x1c, A: 0xff} // 空槽底色
	labelInk      = color.RGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff}
	labelInkLight = color.RGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
)

// drawStateGrid 绘制虚拟页状态色块与物理池属主条。
func drawStateGrid(t *Table) *image.RGBA {
	grid := PagesPerSide * StateBlock
	stripH := StateBlock + 16
	img := image.NewRGBA(image.Rect(0, 0, grid, grid+stripH))
	fill(img, img.Rect, color.RGBA{R: 0x0b, G: 0x0b, B: 0x10, A: 0xff})

	for y := 0; y < PagesPerSide; y++ {
		for x := 0; x < PagesPerSide; x++ {
			vp := VP{X: x, Y: y}
			rect := image.Rect(x*StateBlock, y*StateBlock,
				(x+1)*StateBlock, (y+1)*StateBlock)
			drawStateCell(img, rect, t, vp)
		}
	}

	drawPhysicalStrip(img, t, image.Pt(0, grid+8))
	return img
}

func drawStateCell(img *image.RGBA, rect image.Rectangle, t *Table, vp VP) {
	slot, mapped := t.Mapped(vp)

	// 内部填充：已映射用该页基色，否则用暗灰，让“是否驻留”一眼可辨。
	inner := rect.Inset(3)
	if mapped {
		fill(img, inner, pageBaseColor(vp))
	} else {
		fill(img, inner, color.RGBA{R: 0x26, G: 0x26, B: 0x2e, A: 0xff})
	}

	// 边框表达页表状态。
	border := stateCold
	if mapped {
		border = stateMapped
	} else if t.EvictedOnce(vp) {
		border = stateEvicted
	}
	drawBorder(img, rect, border)

	// 角落小色块：已映射显示物理槽号，曾淘汰显示红色标记。
	if mapped {
		tag := image.Rect(rect.Min.X+3, rect.Min.Y+3,
			rect.Min.X+11, rect.Min.Y+11)
		fill(img, tag, labelInkLight)
		drawDigit(img, slot, rect.Min.X+5, rect.Min.Y+5, labelInk)
	} else if t.EvictedOnce(vp) {
		tag := image.Rect(rect.Min.X+3, rect.Min.Y+3,
			rect.Min.X+9, rect.Min.Y+9)
		fill(img, tag, stateEvicted)
	}
}

func drawPhysicalStrip(img *image.RGBA, t *Table, origin image.Point) {
	for slot := 0; slot < PhysicalPages; slot++ {
		rect := image.Rect(origin.X+slot*StateBlock, origin.Y,
			origin.X+(slot+1)*StateBlock, origin.Y+StateBlock)
		vp, ok := t.Owner(slot)
		if !ok {
			fill(img, rect, stateFree)
			drawBorder(img, rect, stateCold)
			continue
		}
		fill(img, rect, pageBaseColor(vp))
		drawBorder(img, rect, stateMapped)
		drawDigit(img, slot, rect.Min.X+2, rect.Min.Y+2, labelInk)
		drawVP(img, vp, rect.Min.X+10, rect.Min.Y+18)
	}
}

func drawVP(img *image.RGBA, vp VP, x, y int) {
	drawDigit(img, vp.X, x, y, labelInk)
	drawDigit(img, vp.Y, x+8, y, labelInk)
}

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	for py := r.Min.Y; py < r.Max.Y; py++ {
		for px := r.Min.X; px < r.Max.X; px++ {
			img.Set(px, py, c)
		}
	}
}

func drawBorder(img *image.RGBA, r image.Rectangle, c color.Color) {
	for px := r.Min.X; px < r.Max.X; px++ {
		img.Set(px, r.Min.Y, c)
		img.Set(px, r.Max.Y-1, c)
	}
	for py := r.Min.Y; py < r.Max.Y; py++ {
		img.Set(py, r.Min.X, c)
		img.Set(py, r.Max.X-1, c)
	}
}
