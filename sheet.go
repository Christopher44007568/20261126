package hdrlab

// BuildComparisonSheet lays out the three brackets on top and the three
// fusion results below, with caption bars, on a dark background.
func BuildComparisonSheet(brackets []*Img, direct, aligned, preview *Img) *Img {
	const (
		tileW  = 390
		tileH  = 248
		labelH = 26
		cols   = 3
		rows   = 2
	)
	sheet := NewImg(cols*tileW, rows*tileH)
	sheet.FillRect(0, 0, sheet.W, sheet.H, 0.055, 0.07, 0.10)
	topLabels := []string{
		"1 UNDER -2EV: HIGHLIGHT DETAIL",
		"2 NORMAL 0EV: ALIGN REFERENCE",
		"3 OVER +2EV: SHADOW DETAIL",
	}
	botLabels := []string{
		"DIRECT AVERAGE: CLIP + GHOST",
		"ALIGNED AVERAGE: STILL CLIPS",
		"HDR PREVIEW: WEIGHTED DEGHOST",
	}
	bot := []*Img{direct, aligned, preview}
	for c := 0; c < cols; c++ {
		ox := c * tileW
		sheet.DrawTextBox(ox+8, 9, topLabels[c], 0.92, 0.94, 0.97)
		blit(sheet, brackets[c], ox+3, labelH)
		sheet.Rect(ox+3, labelH, ox+3+SceneW, labelH+SceneH, 0.4, 0.45, 0.5, 0.8)
		by := tileH
		sheet.DrawTextBox(ox+8, by+9, botLabels[c], 0.92, 0.94, 0.97)
		bx := ox + (tileW-OutW)/2
		blit(sheet, bot[c], bx, by+labelH+5)
		sheet.Rect(bx, by+labelH+5, bx+OutW, by+labelH+5+OutH, 0.4, 0.45, 0.5, 0.8)
	}
	return sheet
}

func blit(dst, src *Img, ox, oy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			dx, dy := ox+x, oy+y
			if dx < 0 || dy < 0 || dx >= dst.W || dy >= dst.H {
				continue
			}
			r, g, b := src.Get(x, y)
			dst.Set(dx, dy, r, g, b)
		}
	}
}
