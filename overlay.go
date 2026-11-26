package hdrlab

import "fmt"

// Jet maps t in [0,1] to the classic blue-cyan-green-yellow-red colormap.
func Jet(t float64) (r, g, b float64) {
	t = Clamp01(t)
	r = Clamp01(1.5 - abs(4*t-3))
	g = Clamp01(1.5 - abs(4*t-2))
	b = Clamp01(1.5 - abs(4*t-1))
	return
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Cluster is a connected region of ghost-mask pixels.
type Cluster struct {
	Bounds Rect
	Count  int
}

// FindClusters labels 8-connected components in mask and keeps those with
// at least minSize pixels.
func FindClusters(mask []bool, w, h, minSize int) []Cluster {
	seen := make([]bool, len(mask))
	var out []Cluster
	queue := make([]int, 0, 256)
	for start := range mask {
		if !mask[start] || seen[start] {
			continue
		}
		queue = queue[:0]
		queue = append(queue, start)
		seen[start] = true
		c := Cluster{Bounds: Rect{MinX: w, MinY: h}}
		for len(queue) > 0 {
			p := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			px, py := p%w, p/w
			c.Count++
			if px < c.Bounds.MinX {
				c.Bounds.MinX = px
			}
			if px > c.Bounds.MaxX {
				c.Bounds.MaxX = px
			}
			if py < c.Bounds.MinY {
				c.Bounds.MinY = py
			}
			if py > c.Bounds.MaxY {
				c.Bounds.MaxY = py
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := px+dx, py+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					np := ny*w + nx
					if mask[np] && !seen[np] {
						seen[np] = true
						queue = append(queue, np)
					}
				}
			}
		}
		if c.Count >= minSize {
			out = append(out, c)
		}
	}
	return out
}

func drawClusterEllipses(im *Img, clusters []Cluster, r, g, b, a float64) {
	for _, c := range clusters {
		cx := (c.Bounds.MinX + c.Bounds.MaxX) / 2
		cy := (c.Bounds.MinY + c.Bounds.MaxY) / 2
		rx := (c.Bounds.MaxX-c.Bounds.MinX)/2 + 5
		ry := (c.Bounds.MaxY-c.Bounds.MinY)/2 + 5
		im.Ellipse(cx, cy, rx, ry, r, g, b, a)
		im.Ellipse(cx, cy, rx+1, ry+1, r, g, b, a*0.6)
	}
}

// ResidualScale returns the max residual, used as the heat-map full scale.
func ResidualScale(resid *Img) float64 {
	max := 1e-9
	for i := 0; i < len(resid.Pix); i += 3 {
		if resid.Pix[i] > max {
			max = resid.Pix[i]
		}
	}
	return max
}

// BuildAlignmentErrorImage renders the per-pixel alignment residual as a
// jet heat map over a darkened preview, with the ghost mask in magenta,
// a shift-vector inset, and a color scale bar.
func BuildAlignmentErrorImage(preview, resid *Img, ghost []bool, clusters []Cluster, specs []FrameSpec, shifts []Shift, scale float64) *Img {
	out := preview.Clone()
	for i := range out.Pix {
		out.Pix[i] *= 0.45
	}
	for oy := 0; oy < OutH; oy++ {
		for ox := 0; ox < OutW; ox++ {
			t := resid.Pix[(oy*OutW+ox)*3] / scale
			if t > 0.04 {
				r, g, b := Jet(t)
				a := 0.35 + 0.55*t
				if a > 0.9 {
					a = 0.9
				}
				out.Blend(ox, oy, r, g, b, a)
			}
			if ghost[oy*OutW+ox] {
				out.Blend(ox, oy, 1, 0, 1, 0.55)
			}
		}
	}
	drawClusterEllipses(out, clusters, 1, 0.25, 0.7, 0.9)

	out.DrawTextBox(4, 4, "ALIGN ERROR + GHOST MASK", 1, 1, 1)
	y := 16
	for i, s := range specs {
		if i == 1 {
			continue // reference frame has no shift
		}
		label := fmt.Sprintf("%s DX%+.1f DY%+.1f", upper(s.Name), float64(shifts[i].DX), float64(shifts[i].DY))
		out.DrawTextBox(4, y, label, 1, 0.95, 0.4)
		y += 12
	}
	drawShiftInset(out, shifts)
	drawScaleBar(out, scale)
	out.DrawTextBox(4, OutH-12, "MAGENTA=MOTION/DEGHOST", 1, 0.4, 1)
	return out
}

// drawShiftInset renders the recovered handheld shifts as scaled vectors.
func drawShiftInset(out *Img, shifts []Shift) {
	const factor = 7
	x0, y0, x1, y1 := OutW-128, 14, OutW-10, 92
	out.BlendRect(x0, y0, x1, y1, 0.03, 0.04, 0.06, 0.78)
	out.Rect(x0, y0, x1, y1, 0.5, 0.55, 0.6, 0.9)
	cx, cy := (x0+x1)/2, (y0+y1)/2-4
	out.Line(x0+4, cy, x1-4, cy, 0.35, 0.38, 0.42, 0.8)
	out.Line(cx, y0+4, cx, y1-12, 0.35, 0.38, 0.42, 0.8)
	// under: yellow, over: cyan
	out.Arrow(cx, cy, cx+shifts[0].DX*factor, cy+shifts[0].DY*factor, 1, 0.85, 0.1, 0.95)
	out.Arrow(cx, cy, cx+shifts[2].DX*factor, cy+shifts[2].DY*factor, 0.1, 0.9, 1, 0.95)
	out.DrawText(x1-52, y1-10, "SHIFT X7", 0.9, 0.9, 0.9)
}

func drawScaleBar(out *Img, scale float64) {
	x0, x1 := OutW-224, OutW-10
	y0, y1 := OutH-26, OutH-18
	for x := x0; x < x1; x++ {
		r, g, b := Jet(float64(x-x0) / float64(x1-x0-1))
		for y := y0; y < y1; y++ {
			out.Blend(x, y, r, g, b, 0.95)
		}
	}
	out.Rect(x0, y0, x1, y1, 0.7, 0.7, 0.7, 0.9)
	out.DrawTextBox(x0, y0-11, "RESIDUAL", 1, 1, 1)
	out.DrawTextBox(x0, y1+2, "0.00", 1, 1, 1)
	label := fmt.Sprintf("%.2f", scale)
	out.DrawTextBox(x1-TextWidth(label), y1+2, label, 1, 1, 1)
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
