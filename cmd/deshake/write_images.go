package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"

	"deshake/internal/render"
	"deshake/internal/stab"
	"deshake/internal/synth"
)

var (
	red    = color.RGBA{R: 220, G: 70, B: 70, A: 255}
	blue   = color.RGBA{R: 70, G: 110, B: 220, A: 255}
	green  = color.RGBA{R: 60, G: 190, B: 100, A: 255}
	orange = color.RGBA{R: 230, G: 150, B: 50, A: 255}
	purple = color.RGBA{R: 150, G: 80, B: 210, A: 255}
	teal   = color.RGBA{R: 50, G: 180, B: 180, A: 255}
)

func writeTriptyches(frames []*synth.Frame, r2, r3 []*stab.FrameResult,
	all []rowMetrics, g2, g3 stab.Rect) error {
	picks := pickFrames(all)
	for _, fi := range picks {
		f := frames[fi]
		p0 := render.DrawRGB(f.Img.Pix, f.W, f.H)
		p1 := render.DrawResult(r2[fi], true)
		p2 := render.DrawResult(r3[fi], true)

		// Yellow dashed global safe-crop overlay on each stabilized panel.
		drawGlobal(p1, g2)
		drawGlobal(p2, g3)

		m := all[fi]
		footer := []string{
			"FRAME " + strconv.Itoa(fi),
			fmt.Sprintf("HOLE 2D %.1f%% 3D %.1f%%", 100*m.Hole2D, 100*m.Hole3D),
			fmt.Sprintf("JELLY RMSE NONE %.2f 2D %.2f 3D %.2f",
				m.JellyNoneRMSE, m.Jelly2DRMSE, m.Jelly3DRMSE),
		}
		trip := render.Triptych([]*render.Canvas{p0, p1, p2},
			[]string{"1 NO STABILIZATION", "2 2D TRANSLATION", "3 3D DE-ROLL"},
			footer)
		if err := savePNG(trip, filepath.Join(outDir,
			fmt.Sprintf("triptych_f%02d.png", fi))); err != nil {
			return err
		}
	}
	return nil
}

func pickFrames(all []rowMetrics) []int {
	turn := selectTurn(all)
	picks := map[int]bool{5: true, turn: true, nFrames - 3: true}
	out := make([]int, 0, len(picks))
	for k := range picks {
		out = append(out, k)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func drawGlobal(c *render.Canvas, b stab.Rect) {
	yellow := color.RGBA{R: 245, G: 220, B: 60, A: 255}
	for x := b.X0; x < b.X1; x += 6 {
		c.Set(x, b.Y0, yellow)
		c.Set(x, b.Y1-1, yellow)
	}
	for y := b.Y0; y < b.Y1; y += 6 {
		c.Set(b.X0, y, yellow)
		c.Set(b.X1-1, y, yellow)
	}
}

func writeMarginChart(all []rowMetrics, r3 []*stab.FrameResult) error {
	n := len(all)
	margin2d := []render.Series{
		{Name: "2D TOP", Dash: true, Col: red, Y: edgeSeq(all, true, edgeTop)},
		{Name: "2D BOTTOM", Dash: true, Col: blue, Y: edgeSeq(all, true, edgeBottom)},
		{Name: "2D LEFT", Dash: true, Col: green, Y: edgeSeq(all, true, edgeLeft)},
		{Name: "2D RIGHT", Dash: true, Col: purple, Y: edgeSeq(all, true, edgeRight)},
	}
	margin3d := []render.Series{
		{Name: "3D TOP", Col: red, Y: edgeSeq(all, false, edgeTop)},
		{Name: "3D BOTTOM", Col: blue, Y: edgeSeq(all, false, edgeBottom)},
		{Name: "3D LEFT", Col: green, Y: edgeSeq(all, false, edgeLeft)},
		{Name: "3D RIGHT", Col: purple, Y: edgeSeq(all, false, edgeRight)},
	}
	hole2d := []render.Series{{Name: "2D HOLE %", Dash: true, Col: orange, Y: holeSeq(all, true)}}
	hole3d := []render.Series{{Name: "3D HOLE %", Col: teal, Y: holeSeq(all, false)}}
	turn := selectTurn(all)
	if err := savePNG(render.MarginChart(n, margin2d, margin3d, hole2d, hole3d),
		filepath.Join(outDir, "edge_margin.png")); err != nil {
		return err
	}
	return savePNG(render.BoundaryChart(r3[turn]),
		filepath.Join(outDir, "boundary_crop.png"))
}

type edgeKind int

const (
	edgeTop edgeKind = iota
	edgeBottom
	edgeLeft
	edgeRight
)

func edgeSeq(all []rowMetrics, is2d bool, kind edgeKind) []float64 {
	y := make([]float64, len(all))
	for i, m := range all {
		e := m.Edge3D
		if is2d {
			e = m.Edge2D
		}
		switch kind {
		case edgeTop:
			y[i] = e.Top
		case edgeBottom:
			y[i] = e.Bottom
		case edgeLeft:
			y[i] = e.Left
		default:
			y[i] = e.Right
		}
	}
	return y
}

func holeSeq(all []rowMetrics, is2d bool) []float64 {
	y := make([]float64, len(all))
	for i, m := range all {
		if is2d {
			y[i] = 100 * m.Hole2D
		} else {
			y[i] = 100 * m.Hole3D
		}
	}
	return y
}

func savePNG(c *render.Canvas, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.WritePNG(f)
}
