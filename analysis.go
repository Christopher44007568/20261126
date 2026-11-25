package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// Stats summarises one shutter path across the whole frame sequence.
type Stats struct {
	Shutter       Shutter
	MaxShear      float64 // largest top-vs-bottom edge deviation, pixels
	MaxRowStep    float64 // largest |edge[y+1]-edge[y]|, pixels
	MaxBorderStep float64 // largest adjacent-row step at segment borders
	RestDeviation float64 // max edge deviation from the straight rest pose
	BentFrames    int     // frames whose shear exceeds one pixel
}

// frameShear is the horizontal deviation between the rod's top and bottom
// edges. Zero means the rod is rendered perfectly straight.
func frameShear(edge []float64) float64 {
	return math.Abs(edge[len(edge)-1] - edge[0])
}

// adjacentStep returns the largest timing-induced edge jump between
// neighbouring rows overall, and inside the feather zones around segment
// borders. A well-behaved segmented path has no discontinuity: every jump
// stays on the bounded slope of the linear cross-fade.
func adjacentStep(p Params, edge []float64) (maxStep, borderStep float64, borderRows []int) {
	half := float64(p.BlockRows) / 2
	for y := 0; y+1 < len(edge); y++ {
		d := math.Abs(edge[y+1] - edge[y])
		if d > maxStep {
			maxStep = d
		}
		local := math.Mod(float64(y)+0.5, float64(p.BlockRows))
		inFeather := math.Abs(local-half) <= p.FeatherRows
		if inFeather {
			borderRows = append(borderRows, y)
			if d > borderStep {
				borderStep = d
			}
		}
	}
	return maxStep, borderStep, borderRows
}

// deviationMatrix returns each edge sample's signed deviation from its frame
// mean and the largest absolute value across every shutter path, so all
// heatmaps share one colour scale and stay directly comparable.
func deviationMatrix(p Params, edges [][]float64, globalMax float64) ([][]float64, float64) {
	mat := make([][]float64, p.Frames)
	max := globalMax
	for f, edge := range edges {
		mean := 0.0
		for _, v := range edge {
			mean += v
		}
		mean /= float64(len(edge))
		mat[f] = make([]float64, len(edge))
		for y, v := range edge {
			d := v - mean
			mat[f][y] = d
			if math.Abs(d) > max {
				max = math.Abs(d)
			}
		}
	}
	return mat, max
}

// shearHeatmap renders signed edge deviation from the frame mean using a
// diverging blue -> white -> red scale; columns are frames, rows are sensor
// rows. A fixed shared scale keeps the three shutter paths comparable.
func shearHeatmap(p Params, mat [][]float64, scale int, globalMax float64, path string) error {
	w := p.Frames * scale
	h := p.Height
	im := newImage(w, h)
	if globalMax == 0 {
		globalMax = 1
	}
	for f, col := range mat {
		for y, d := range col {
			r, g, b := divergingColor(d / globalMax)
			for dx := 0; dx < scale; dx++ {
				im.set(f*scale+dx, y, r, g, b)
			}
		}
	}
	return im.writePPM(path)
}

// divergingColor maps -1 (deep blue) through 0 (white) to +1 (deep red).
func divergingColor(d float64) (r, g, b uint8) {
	d = math.Max(-1, math.Min(1, d))
	a := math.Abs(d)
	if d >= 0 {
		// white at 0, deep red at +1
		return 255, uint8(255 * (1 - a)), uint8(255 * (1 - a))
	}
	// white at 0, deep blue at -1
	return uint8(255 * (1 - a)), uint8(255 * (1 - a)), 255
}

// edgeTrajectory plots the rod edge position (canvas y) of three
// representative rows against frame number (canvas x): top (red), middle
// (amber), bottom (green). One panel per shutter path, rest pose marked by a
// dashed reference line.
func edgeTrajectory(p Params, all map[Shutter][][]float64, path string) error {
	const (
		pad    = 12
		stepX  = 24
		panelH = 220
	)
	panelW := p.Frames*stepX + 2*pad
	im := newImage(panelW, panelH*len(all))
	restEdge := p.RodX - p.RodHalf
	// Edge values span the whole pan travel, so normalise against the full
	// observed range rather than the sensor width.
	minEdge, maxEdge := restEdge, restEdge
	for _, edges := range all {
		for _, edge := range edges {
			for _, v := range edge {
				if v < minEdge {
					minEdge = v
				}
				if v > maxEdge {
					maxEdge = v
				}
			}
		}
	}
	span := maxEdge - minEdge
	mapY := func(base int, v float64) int {
		return base + pad + int(((v-minEdge)/span)*float64(panelH-2*pad))
	}
	idx := 0
	for _, s := range []Shutter{Rolling, Segmented, Global} {
		edges := all[s]
		y0 := idx * panelH
		// Rest-pose reference line.
		restY := mapY(y0, restEdge)
		for x := pad; x < panelW-pad; x += 4 {
			im.set(x, restY, 90, 90, 90)
		}
		rows := []struct {
			row int
			col [3]uint8
		}{
			{0, [3]uint8{220, 60, 60}},
			{p.Height / 2, [3]uint8{230, 170, 40}},
			{p.Height - 1, [3]uint8{60, 170, 90}},
		}
		if s == Global {
			// All rows share one instant, so the three traces coincide;
			// draw them as one bold white curve instead of overlapping dots.
			rows = []struct {
				row int
				col [3]uint8
			}{
				{0, [3]uint8{235, 235, 235}},
				{p.Height / 2, [3]uint8{235, 235, 235}},
				{p.Height - 1, [3]uint8{235, 235, 235}},
			}
		}
		for _, rc := range rows {
			prevX, prevY := -1, 0
			for f, edge := range edges {
				x := pad + f*stepX
				yy := mapY(y0, edge[rc.row])
				im.set(x, yy, rc.col[0], rc.col[1], rc.col[2])
				if prevX >= 0 {
					drawLine(im, prevX, prevY, x, yy, rc.col)
				}
				prevX, prevY = x, yy
			}
		}
		for f := 0; f <= p.Frames; f++ {
			x := pad + f*stepX
			for y := y0; y < y0+panelH; y++ {
				if x < im.Width && y < im.Height {
					im.set(x, y, 120, 120, 120)
				}
			}
		}
		idx++
	}
	return im.writePPM(path)
}

func drawLine(im *Image, x0, y0, x1, y1 int, c [3]uint8) {
	dx := abs(float64(x1 - x0))
	dy := -abs(float64(y1 - y0))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && x0 < im.Width && y0 >= 0 && y0 < im.Height {
			im.set(x0, y0, c[0], c[1], c[2])
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// writeEdgesCSV dumps the full H x F edge matrix for external inspection.
func writeEdgesCSV(p Params, edges [][]float64, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for y := 0; y < p.Height; y++ {
		for frame := 0; frame < p.Frames; frame++ {
			sep := ","
			if frame == p.Frames-1 {
				sep = "\n"
			}
			fmt.Fprintf(f, "%.4f%s", edges[frame][y], sep)
		}
	}
	return nil
}

func joinArtifacts(dir, name string) string {
	return filepath.Join(dir, name)
}
