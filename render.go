package main

import "path/filepath"

const errorAmplify = 8

type panelKind int

const (
	panelOriginal panelKind = iota
	panelDecoded
	panelError
)

func renderTexturePanel(im *Bitmap, scale int) *Bitmap {
	out := NewBitmap(im.W*scale, im.H*scale)
	for y := 0; y < out.H; y++ {
		for x := 0; x < out.W; x++ {
			p := im.At(x/scale, y/scale)
			bg := checker(x, y)
			out.Set(x, y, compositeOver(p, bg))
		}
	}
	return out
}

func renderErrorPanel(orig, decoded *Bitmap, scale int) *Bitmap {
	out := NewBitmap(orig.W*scale, orig.H*scale)
	for y := 0; y < out.H; y++ {
		for x := 0; x < out.W; x++ {
			a := orig.At(x/scale, y/scale)
			b := decoded.At(x/scale, y/scale)
			var p Pixel
			switch {
			case (a.A == 0) != (b.A == 0):
				p = Pixel{R: 255, G: 0, B: 198, A: 255}
			case a.A == 0 && b.A == 0:
				p = darkChecker(x, y)
			default:
				diff := max4(absU8(a.R, b.R), absU8(a.G, b.G), absU8(a.B, b.B), absU8(a.A, b.A))
				p = heat(uint8(minInt(255, diff*errorAmplify)))
			}
			out.Set(x, y, p)
		}
	}
	return out
}

func renderComparison(scenarios []Scenario, decoded []*Bitmap, scale int) *Bitmap {
	rows := make([][]*Bitmap, 0, len(scenarios))
	for i, s := range scenarios {
		rows = append(rows, []*Bitmap{
			renderTexturePanel(s.Original, scale),
			renderTexturePanel(decoded[i], scale),
			renderErrorPanel(s.Original, decoded[i], scale),
		})
	}
	return composeGrid(rows, 12)
}

func renderColumn(scenarios []Scenario, decoded []*Bitmap, kind panelKind, scale int) *Bitmap {
	panels := make([]*Bitmap, 0, len(scenarios))
	for i, s := range scenarios {
		switch kind {
		case panelOriginal:
			panels = append(panels, renderTexturePanel(s.Original, scale))
		case panelDecoded:
			panels = append(panels, renderTexturePanel(decoded[i], scale))
		case panelError:
			panels = append(panels, renderErrorPanel(s.Original, decoded[i], scale))
		}
	}
	return composeGrid([][]*Bitmap{panels}, 12)
}

func composeGrid(rows [][]*Bitmap, gap int) *Bitmap {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return NewBitmap(1, 1)
	}
	cellW, cellH := rows[0][0].W, rows[0][0].H
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
		for _, im := range row {
			if im.W != cellW || im.H != cellH {
				panic("render grid cells must have equal dimensions")
			}
		}
	}
	w := cols*cellW + (cols-1)*gap
	h := len(rows)*cellH + (len(rows)-1)*gap
	out := NewBitmap(w, h)
	for i := range out.Pix {
		out.Pix[i] = Pixel{R: 24, G: 26, B: 32, A: 255}
	}
	for row, images := range rows {
		for col, im := range images {
			ox := col * (cellW + gap)
			oy := row * (cellH + gap)
			for y := 0; y < cellH; y++ {
				copy(out.Pix[(oy+y)*out.W+ox:(oy+y)*out.W+ox+cellW],
					im.Pix[y*cellW:(y+1)*cellW])
			}
		}
	}
	return out
}

func checker(x, y int) Pixel {
	if ((x/8)+(y/8))&1 == 0 {
		return Pixel{R: 66, G: 68, B: 76, A: 255}
	}
	return Pixel{R: 92, G: 95, B: 104, A: 255}
}

func darkChecker(x, y int) Pixel {
	if ((x/8)+(y/8))&1 == 0 {
		return Pixel{R: 17, G: 19, B: 24, A: 255}
	}
	return Pixel{R: 28, G: 31, B: 38, A: 255}
}

func compositeOver(fg, bg Pixel) Pixel {
	if fg.A == 255 {
		return fg
	}
	alpha := float64(fg.A) / 255
	blend := func(f, b uint8) uint8 {
		return uint8(float64(f)*alpha + float64(b)*(1-alpha) + 0.5)
	}
	return Pixel{R: blend(fg.R, bg.R), G: blend(fg.G, bg.G), B: blend(fg.B, bg.B), A: 255}
}

func heat(v uint8) Pixel {
	t := float64(v) / 255
	stops := [][3]float64{
		{8, 12, 24},
		{20, 72, 210},
		{0, 220, 220},
		{255, 214, 32},
		{255, 48, 32},
		{255, 255, 255},
	}
	scaled := t * float64(len(stops)-1)
	i := int(scaled)
	if i >= len(stops)-1 {
		i = len(stops) - 2
	}
	f := scaled - float64(i)
	return Pixel{
		R: uint8(stops[i][0] + (stops[i+1][0]-stops[i][0])*f),
		G: uint8(stops[i][1] + (stops[i+1][1]-stops[i][1])*f),
		B: uint8(stops[i][2] + (stops[i+1][2]-stops[i][2])*f),
		A: 255,
	}
}

func max4(a, b, c, d int) int {
	v := a
	if b > v {
		v = b
	}
	if c > v {
		v = c
	}
	if d > v {
		v = d
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func writeRenderedArtifacts(dir string, scenarios []Scenario, decoded []*Bitmap) ([]string, error) {
	exports := []struct {
		name  string
		image *Bitmap
	}{
		{"original.png", renderColumn(scenarios, decoded, panelOriginal, 8)},
		{"decoded.png", renderColumn(scenarios, decoded, panelDecoded, 8)},
		{"error_x8.png", renderColumn(scenarios, decoded, panelError, 8)},
		{"overview.png", renderComparison(scenarios, decoded, 6)},
	}
	paths := make([]string, 0, len(exports))
	for _, e := range exports {
		path := filepath.Join(dir, e.name)
		if err := writePNG(path, e.image); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	for i, s := range scenarios {
		path := filepath.Join(dir, s.ID+"_compare.png")
		one := renderComparison([]Scenario{scenarios[i]}, decoded[i:i+1], 8)
		if err := writePNG(path, one); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
