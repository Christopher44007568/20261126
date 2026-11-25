package main

// drawDiagonalHatch 在子树正文区画斜纹，强调这里没有真实内容。
func drawDiagonalHatch(im *Image, rc Rect, c Color) {
	top := rc.Y + 16
	for k := -rc.H; k < rc.W; k += 10 {
		for t := 0; t < rc.H; t++ {
			x := rc.X + k + t
			y := top + t
			if x >= rc.X && x < rc.X+rc.W && y >= top && y < rc.Y+rc.H {
				im.Set(x, y, c)
			}
		}
	}
}

// drawFullContent 根据缓存 payload 绘制确定性彩色单元网格与标识。
func drawFullContent(im *Image, rc Rect, p *Payload) {
	if p == nil {
		// 防御性回退：状态宣称已加载却无缓存时，绝不能显示旧像素。
		drawUnloaded(im, rc)
		return
	}
	gridTop := rc.Y + 22
	gridLeft := rc.X + 10
	cols, rows := 16, len(p.Rows)
	cellW := (rc.W - 20) / cols
	cellH := (rc.H - 52) / rows

	for row, line := range p.Rows {
		for col := 0; col < cols && col < len(line); col++ {
			c := payloadColor(line[col])
			x := gridLeft + col*cellW
			y := gridTop + row*cellH
			im.SolidRect(x+1, y+1, cellW-2, cellH-2, Color{c.R, c.G, c.B, 210})
		}
	}

	tag := p.SubtreeID + " SEED " + itoa(int(p.Seed))
	tw := TextWidth(tag, 1)
	tagY := rc.Y + rc.H - 22
	im.SolidRect(rc.X+(rc.W-tw)/2-4, tagY-3, tw+8, 14, RGB(0x0c, 0x0e, 0x14))
	im.DrawText(rc.X+(rc.W-tw)/2, tagY, 1, tag, RGB(0xd1, 0xfa, 0xe5))
}
