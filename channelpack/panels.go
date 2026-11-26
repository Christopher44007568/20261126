package main

import "image"

const (
	tileSize = 248
	gap      = 16
	margin   = 20
)

type slotInfo struct {
	slot     string
	semantic string
}

func decompositionSpec(layout string) [][]slotInfo {
	switch layout {
	case "separate":
		return [][]slotInfo{
			{{"R", "ALBEDO R"}, {"G", "ALBEDO G"}, {"B", "ALBEDO B"}, {"A", "ONE 255"}},
			{{"R", "NORMAL X"}, {"G", "NORMAL Y"}, {"B", "NORMAL Z"}, {"A", "ONE 255"}},
			{{"R", "ROUGHNESS"}, {"G", "ROUGHNESS"}, {"B", "ROUGHNESS"}, {"A", "ONE 255"}},
			{{"R", "OCCLUSION"}, {"G", "OCCLUSION"}, {"B", "OCCLUSION"}, {"A", "ONE 255"}},
		}
	case "packed":
		return [][]slotInfo{
			{{"R", "ALBEDO R"}, {"G", "ALBEDO G"}, {"B", "ALBEDO B"}, {"A", "OCCLUSION"}},
			{{"R", "NORMAL X"}, {"G", "NORMAL Y"}, {"B", "NORMAL Z"}, {"A", "ROUGHNESS"}},
		}
	default:
		return [][]slotInfo{
			{{"R", "ALBEDO R"}, {"G", "ALBEDO G"}, {"B", "ALBEDO B"}, {"A", "ROUGH4|AO4"}},
			{{"R", "NORMAL X"}, {"G", "NORMAL Y"}, {"B", "NORMAL Z"}, {"A", "UNUSED 255"}},
		}
	}
}

func buildDecomposition(layout string, texs []*Texture, w, h int) *image.NRGBA {
	spec := decompositionSpec(layout)
	rowPitch := 306
	width := margin*2 + tileSize*4 + gap*3
	height := 30 + rowPitch*len(spec) + 14
	img := newCanvas(width, height)
	drawCenteredText(img, 12, "CHANNEL DECOMPOSITION - "+upper(layout), inkColor)

	for row, infos := range spec {
		baseY := 32 + row*rowPitch
		drawText(img, margin, baseY, "TEXTURE "+itoa(row+1), inkColor)
		tileY := baseY + 16
		for col, info := range infos {
			x := margin + col*(tileSize+gap)
			drawGrayTile(img, x, tileY, tileSize, tileSize, w, h, texs[row].Pix, col, 4)
			frameRect(img, x, tileY, tileSize, tileSize, frameColor)
			label := info.slot + "  " + info.semantic
			drawText(img, x+(tileSize-textWidth(label))/2, tileY+tileSize+10, label, inkColor)
		}
	}
	return img
}

func buildErrorMap(title string, ref, got *Maps, gain int) *image.NRGBA {
	rowPitch := 290
	width := margin*2 + tileSize*3 + gap*2
	height := 30 + rowPitch*3 + 4
	img := newCanvas(width, height)
	drawCenteredText(img, 12, title, inkColor)

	labels := [][]string{
		{"ALBEDO R", "ALBEDO G", "ALBEDO B"},
		{"NORMAL X", "NORMAL Y", "NORMAL Z"},
		{"ROUGHNESS", "OCCLUSION", ""},
	}
	for row := 0; row < 3; row++ {
		tileY := 32 + row*rowPitch
		for col := 0; col < 3; col++ {
			x := margin + col*(tileSize+gap)
			label := labels[row][col]
			if label == "" {
				continue
			}
			switch {
			case row == 0:
				drawDiffTile(img, x, tileY, tileSize, tileSize, ref.W, ref.H,
					ref.Albedo, got.Albedo, col, 3, gain)
			case row == 1:
				drawDiffTile(img, x, tileY, tileSize, tileSize, ref.W, ref.H,
					ref.Normal, got.Normal, col, 3, gain)
			case col == 0:
				drawDiffTile(img, x, tileY, tileSize, tileSize, ref.W, ref.H,
					ref.Roughness, got.Roughness, 0, 1, gain)
			default:
				drawDiffTile(img, x, tileY, tileSize, tileSize, ref.W, ref.H,
					ref.Occlusion, got.Occlusion, 0, 1, gain)
			}
			frameRect(img, x, tileY, tileSize, tileSize, frameColor)
			drawText(img, x+(tileSize-textWidth(label))/2, tileY+tileSize+10, label, inkColor)
		}
	}
	return img
}

func buildComparison(spheres []*image.NRGBA) *image.NRGBA {
	size := spheres[0].Bounds().Dx()
	bar := 8
	width := size*len(spheres) + bar*(len(spheres)-1)
	img := newCanvas(width, size)
	for i, s := range spheres {
		x := i * (size + bar)
		for y := 0; y < size; y++ {
			for xx := 0; xx < size; xx++ {
				img.Set(x+xx, y, s.NRGBAAt(xx, y))
			}
		}
		if i > 0 {
			fillRect(img, x-bar, 0, bar, size, whiteBar)
		}
	}
	return img
}

func textureToImage(t *Texture, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, t.Pix)
	return img
}

func planarToImage(pix []byte, w, h, channels int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		if channels == 3 {
			img.Pix[i*4] = pix[i*3]
			img.Pix[i*4+1] = pix[i*3+1]
			img.Pix[i*4+2] = pix[i*3+2]
		} else {
			v := pix[i]
			img.Pix[i*4] = v
			img.Pix[i*4+1] = v
			img.Pix[i*4+2] = v
		}
		img.Pix[i*4+3] = 255
	}
	return img
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
