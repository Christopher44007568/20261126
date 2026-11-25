package main

import (
	"image"
	"math"
)

type ErrorGrid struct {
	Spec    CheckerSpec
	ByID    map[int]PointResidual
	MaxErr  float64
	RMSE    float64
	ScalePx float64
}

func NewErrorGrid(spec CheckerSpec, residuals []PointResidual, scalePx float64) ErrorGrid {
	g := ErrorGrid{Spec: spec, ByID: map[int]PointResidual{}, ScalePx: scalePx}
	for _, r := range residuals {
		g.ByID[r.ID] = r
		g.MaxErr = math.Max(g.MaxErr, r.Error)
	}
	g.RMSE = RMSEResidual(residuals)
	if g.ScalePx <= 0 {
		g.ScalePx = math.Max(1, g.MaxErr)
	}
	return g
}

func RenderResidualHeatmap(src *image.RGBA, g ErrorGrid) *image.RGBA {
	var img *image.RGBA
	if src != nil {
		img = dimImage(src, 0.34)
	} else {
		img = newCanvas(g.Spec.CornersX*80, g.Spec.CornersY*80, colorDark)
	}
	nx := g.Spec.CornersX
	for j := 0; j < g.Spec.CornersY-1; j++ {
		for i := 0; i < nx-1; i++ {
			ids := [4]int{j*nx + i, j*nx + i + 1, (j+1)*nx + i + 1, (j+1)*nx + i}
			rs := [4]PointResidual{}
			ok := true
			for k, id := range ids {
				r, exists := g.ByID[id]
				if !exists {
					ok = false
					break
				}
				rs[k] = r
			}
			if ok {
				ps := [4]Vec2{rs[0].Got, rs[1].Got, rs[2].Got, rs[3].Got}
				es := [4]float64{rs[0].Error, rs[1].Error, rs[2].Error, rs[3].Error}
				drawHeatQuad(img, ps, es, g.ScalePx)
			}
		}
	}
	residuals := make([]PointResidual, 0, len(g.ByID))
	for _, r := range g.ByID {
		residuals = append(residuals, r)
	}
	DrawReprojectionErrors(img, residuals)
	drawLegend(img, g.ScalePx, g.MaxErr, g.RMSE)
	return img
}

func drawHeatQuad(img *image.RGBA, ps [4]Vec2, es [4]float64, scale float64) {
	drawTriangleHeat(img, ps[0], ps[1], ps[2], es[0], es[1], es[2], scale)
	drawTriangleHeat(img, ps[0], ps[2], ps[3], es[0], es[2], es[3], scale)
}

func drawTriangleHeat(img *image.RGBA, a, b, c Vec2, ea, eb, ec, scale float64) {
	minX := int(math.Floor(math.Min(a.X, math.Min(b.X, c.X))))
	maxX := int(math.Ceil(math.Max(a.X, math.Max(b.X, c.X))))
	minY := int(math.Floor(math.Min(a.Y, math.Min(b.Y, c.Y))))
	maxY := int(math.Ceil(math.Max(a.Y, math.Max(b.Y, c.Y))))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			wa, wb, wc, ok := barycentric(p, a, b, c)
			if !ok {
				continue
			}
			err := wa*ea + wb*eb + wc*ec
			col := heatColor(err / scale)
			blendPixel(img, x, y, col, 0.76)
		}
	}
}

func barycentric(p, a, b, c Vec2) (float64, float64, float64, bool) {
	v0 := Vec2{X: b.X - a.X, Y: b.Y - a.Y}
	v1 := Vec2{X: c.X - a.X, Y: c.Y - a.Y}
	v2 := Vec2{X: p.X - a.X, Y: p.Y - a.Y}
	d00 := v0.X*v0.X + v0.Y*v0.Y
	d01 := v0.X*v1.X + v0.Y*v1.Y
	d11 := v1.X*v1.X + v1.Y*v1.Y
	d20 := v2.X*v0.X + v2.Y*v0.Y
	d21 := v2.X*v1.X + v2.Y*v1.Y
	den := d00*d11 - d01*d01
	if math.Abs(den) < 1e-12 {
		return 0, 0, 0, false
	}
	wb := (d11*d20 - d01*d21) / den
	wc := (d00*d21 - d01*d20) / den
	wa := 1 - wb - wc
	const eps = -1e-4
	if wa < eps || wb < eps || wc < eps {
		return 0, 0, 0, false
	}
	return wa, wb, wc, true
}

func drawLegend(img *image.RGBA, scale, maxErr, rmse float64) {
	b := img.Bounds()
	lx := b.Dx() - 54
	y0, h, w := 70, 240, 18
	xRight := lx + w
	drawLine(img, Vec2{X: float64(lx - 8), Y: float64(y0 - 8)}, Vec2{X: float64(xRight + 8), Y: float64(y0 - 8)}, 2, colorWhite)
	drawLine(img, Vec2{X: float64(lx - 8), Y: float64(y0 + h + 8)}, Vec2{X: float64(xRight + 8), Y: float64(y0 + h + 8)}, 2, colorWhite)
	for y := y0; y < y0+h; y++ {
		col := heatColor(1 - float64(y-y0)/float64(h-1))
		for px := lx; px < xRight; px++ {
			img.SetRGBA(px, y, col)
		}
	}
	drawLine(img, Vec2{X: float64(lx), Y: float64(y0)}, Vec2{X: float64(xRight), Y: float64(y0)}, 1.4, colorWhite)
	drawLine(img, Vec2{X: float64(lx), Y: float64(y0 + h)}, Vec2{X: float64(xRight), Y: float64(y0 + h)}, 1.4, colorWhite)
	drawLine(img, Vec2{X: float64(lx), Y: float64(y0)}, Vec2{X: float64(lx), Y: float64(y0 + h)}, 1.4, colorWhite)
	drawLine(img, Vec2{X: float64(xRight), Y: float64(y0)}, Vec2{X: float64(xRight), Y: float64(y0 + h)}, 1.4, colorWhite)
	for k := 0; k <= 4; k++ {
		y := y0 + h*k/4
		drawLine(img, Vec2{X: float64(xRight), Y: float64(y)}, Vec2{X: float64(xRight + 8), Y: float64(y)}, 1.5, colorWhite)
	}
	_ = maxErr
	_ = rmse
}
