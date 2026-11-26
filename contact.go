package main

// Blit 将 src 原样复制到 dst 的 (x,y)。所有变体帧尺寸一致。
func (dst *Image) Blit(src *Image, x, y int) {
	for sy := 0; sy < src.H; sy++ {
		for sx := 0; sx < src.W; sx++ {
			dx, dy := x+sx, y+sy
			if !dst.InBounds(dx, dy) {
				continue
			}
			si := src.offset(sx, sy)
			di := dst.offset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
}

// BuildContactSheet 生成夏/冬/夜三变体横向 PPM 拼图。
func BuildContactSheet(frames map[Variant]*Image) *Image {
	const pad = 20
	const panelY = 82
	width := pad*4 + FrameW*3
	height := 532
	sheet := NewImage(width, height)
	bg := hexRGB("#0F172A")
	panelBorder := hexRGB("#CBD5E1")
	muted := hexRGB("#94A3B8")
	white := hexRGB("#F8FAFC")
	sheet.Fill(bg)

	sheet.DrawCenteredText(width/2, 24, 3, "ASSET VARIANT CONTACT SHEET", white)
	sheet.DrawCenteredText(width/2, 58, 1, "ONE ASSET TREE - THREE RESOLVED BINDING SETS", muted)

	order := AllVariants()
	accents := map[Variant]RGB{
		VariantSummer: hexRGB("#F2C94C"),
		VariantWinter: hexRGB("#7DD3FC"),
		VariantNight:  hexRGB("#A78BFA"),
	}
	for i, variant := range order {
		x := pad + i*(FrameW+pad)
		frame := frames[variant]
		sheet.Blit(frame, x, panelY)
		sheet.FrameRect(Rect{X: x - 2, Y: panelY - 2, W: FrameW + 4, H: FrameH + 4}, 2, panelBorder)
		sheet.FrameRect(Rect{X: x - 5, Y: panelY - 5, W: FrameW + 10, H: FrameH + 10}, 1, accents[variant])

		label := variant.Label()
		sheet.DrawCenteredText(x+FrameW/2, panelY+FrameH+18, 3, label, accents[variant])
		hint := "VARIANT=" + string(variant) + " / SLOT SURFACE BOUND"
		sheet.DrawCenteredText(x+FrameW/2, panelY+FrameH+50, 1, hint, muted)
	}

	footer := "SELECTION PROPAGATES PARENT TO CHILD; RESTORE REBINDS AND MATCHES EVERY RGB BYTE"
	sheet.DrawCenteredText(width/2, height-24, 2, footer, white)
	return sheet
}

// AddPathCaption 在传播演示帧顶部和底部加入可读路径条。
func AddPathCaption(im *Image, resolution Resolution) {
	dark := RGB{R: 8, G: 12, B: 24}
	amber := hexRGB("#FFB000")
	cyan := hexRGB("#67E8F9")
	white := hexRGB("#F8FAFC")

	im.BlendRect(Rect{X: 10, Y: 10, W: FrameW - 20, H: 42}, dark, 225)
	im.FrameRect(Rect{X: 10, Y: 10, W: FrameW - 20, H: 42}, 1, amber)
	im.DrawText(18, 18, 2, "PARENT=WINTER  BANK=NIGHT", white)

	pathLine := "TARGET: " + resolution.TargetPath
	im.BlendRect(Rect{X: 10, Y: FrameH - 48, W: FrameW - 20, H: 38}, dark, 225)
	im.FrameRect(Rect{X: 10, Y: FrameH - 48, W: FrameW - 20, H: 38}, 1, cyan)
	im.DrawText(18, FrameH-40, 1, pathLine, cyan)
	sourceLine := "LEFT + RIGHT INHERIT NIGHT FROM WINDOWBANK"
	im.DrawText(18, FrameH-24, 1, sourceLine, white)
}

// AddCurrentPathCaption 标注恢复完成后的当前选择路径。
func AddCurrentPathCaption(im *Image, resolution Resolution) {
	dark := RGB{R: 8, G: 12, B: 24}
	green := hexRGB("#6EE7B7")
	cyan := hexRGB("#67E8F9")
	white := hexRGB("#F8FAFC")

	im.BlendRect(Rect{X: 10, Y: 10, W: FrameW - 20, H: 42}, dark, 225)
	im.FrameRect(Rect{X: 10, Y: 10, W: FrameW - 20, H: 42}, 1, green)
	im.DrawText(18, 18, 2, "CURRENT="+resolution.Variant.Label(), white)

	im.BlendRect(Rect{X: 10, Y: FrameH - 48, W: FrameW - 20, H: 38}, dark, 225)
	im.FrameRect(Rect{X: 10, Y: FrameH - 48, W: FrameW - 20, H: 38}, 1, cyan)
	im.DrawText(18, FrameH-40, 1, "TARGET: "+resolution.TargetPath, cyan)
	im.DrawText(18, FrameH-24, 1, "SOURCE: "+resolution.SourcePath, white)
}
