package viz

import (
	"math"

	"nsmigrate/audit"
	"nsmigrate/raster"
	"nsmigrate/scene"
)

func (p *Poster) drawScene(o Options, panelH int) {
	c := p.Canvas
	px, py, pw := margin, contentY, sceneW

	c.Text("TOP-DOWN SCENE MAP", px+24, py+20, 3, palette.ink)
	c.Text("geometry unchanged by migration - only namespace paths are re-rooted",
		px+26, py+58, 1, palette.muted)

	mx0, mx1 := px+24, px+pw-24
	legendY := py + panelH - 92
	my0, my1 := py+74, legendY-16

	const wMinX, wMaxX = -6.6, 6.6
	const wMinY, wMaxY = -4.9, 5.2
	t := tform{
		scale: math.Min(float64(mx1-mx0)/(wMaxX-wMinX), float64(my1-my0)/(wMaxY-wMinY)),
		cx:    float64(mx0+mx1) / 2,
		cy:    float64(my0+my1) / 2,
		wcx:   0,
		wcy:   (wMinY + wMaxY) / 2,
	}

	c.FillRect(mx0, my0, mx1-mx0, my1-my0, palette.floor)
	for gx := int(math.Ceil(wMinX)); gx <= int(math.Floor(wMaxX)); gx++ {
		x := t.X(float64(gx))
		col := palette.grid
		if gx == 0 {
			col = raster.Hex(0xD6DFEB)
		}
		c.Line(x, my0, x, my1, col)
	}
	for gy := int(math.Ceil(wMinY)); gy <= int(math.Floor(wMaxY)); gy++ {
		y := t.Y(float64(gy))
		col := palette.grid
		if gy == 0 {
			col = raster.Hex(0xD6DFEB)
		}
		c.Line(mx0, y, mx1, y, col)
	}
	c.FrameRect(mx0, my0, mx1-mx0, my1-my0, 1, palette.border)

	rects := map[string]rect{}
	var union rect
	first := true
	for _, pr := range o.Stage.Prims {
		r := primRect(t, pr)
		rects[pr.ID] = r
		if pr.Shape == scene.ShapeGroup {
			continue
		}
		if first {
			union, first = r, false
		} else {
			union = unionUnion(union, r)
		}
	}
	bound := rect{x: union.x - 26, y: union.y - 26, w: union.w + 52, h: union.h + 64}
	p.drawBoundary(o, bound)

	matByOwner := map[string]audit.Finding{}
	for _, f := range o.Report.FindingsFor(scene.KindMaterialBinding) {
		matByOwner[f.OwnerID] = f
	}

	p.drawLinks(o, rects)

	for _, pr := range o.Stage.Prims {
		if pr.Shape == scene.ShapeGroup {
			continue
		}
		fill := cvt(pr.Color)
		if f, ok := matByOwner[pr.ID]; ok {
			if f.Status == audit.StatusOK {
				if tp := o.Stage.PrimByID(f.ResolvedPrimID); tp != nil {
					fill = cvt(tp.Color)
				}
			} else {
				fill = cvt(scene.DefaultMaterialColor)
			}
		}
		p.drawPrimShape(pr, rects[pr.ID], fill)
	}

	p.annotate(o, rects, matByOwner)
	p.drawLegend(px+24, legendY, pw-48)
}

func unionUnion(a, b rect) rect {
	x0 := min(a.x, b.x)
	y0 := min(a.y, b.y)
	x1 := max(a.x+a.w, b.x+b.w)
	y1 := max(a.y+a.h, b.y+b.h)
	return rect{x: x0, y: y0, w: x1 - x0, h: y1 - y0}
}

func (p *Poster) drawBoundary(o Options, b rect) {
	c := p.Canvas
	col := raster.Hex(0x4F46E5)
	label := o.Mapping.OldRoot
	if o.Mode == "correct" {
		col = raster.Hex(0x1D4ED8)
		label = o.Mapping.NewRoot
	}
	if o.Mode == "string-only" {
		col = raster.Red
		label = "PARTIAL STRING REPLACE - CHILDREN STALE AT " + o.Mapping.OldRoot
	}
	c.DashedLine(b.x, b.y, b.x+b.w, b.y, 14, 9, 3, col)
	c.DashedLine(b.x, b.y+b.h, b.x+b.w, b.y+b.h, 14, 9, 3, col)
	c.DashedLine(b.x, b.y, b.x, b.y+b.h, 14, 9, 3, col)
	c.DashedLine(b.x+b.w, b.y, b.x+b.w, b.y+b.h, 14, 9, 3, col)

	tw := raster.TextWidth(label, 2)
	chipW, chipH := tw+24, 28
	chipX, chipY := b.x+18, b.y-15
	c.FillRect(chipX, chipY, chipW, chipH, col)
	c.Text(label, chipX+12, chipY+5, 2, raster.Hex(0xFFFFFF))
}

func (p *Poster) drawLinks(o Options, rects map[string]rect) {
	c := p.Canvas
	for _, f := range o.Report.Findings {
		owner := o.Stage.PrimByID(f.OwnerID)
		if owner == nil {
			continue
		}
		or := rects[owner.ID]
		ocx, ocy := or.cx(), or.cy()

		if f.Status == audit.StatusOK {
			tp := o.Stage.PrimByID(f.ResolvedPrimID)
			if tp == nil {
				continue
			}
			tr := rects[tp.ID]
			tcx, tcy := tr.cx(), tr.cy()
			dx, dy := float64(tcx-ocx), float64(tcy-ocy)
			dist := math.Hypot(dx, dy)
			ux, uy := dx/dist, dy/dist
			rad := float64(max(tr.w, tr.h)/2 + 8)
			tipX := tcx - int(ux*rad)
			tipY := tcy - int(uy*rad)
			col := kindColor(string(f.Kind))
			if f.Kind == scene.KindReference {
				c.DashedLine(ocx, ocy, tipX, tipY, 12, 8, 3, col)
			} else {
				c.LineWidth(ocx, ocy, tipX, tipY, 3, col)
			}
			ang := math.Atan2(float64(tipY-ocy), float64(tipX-ocx))
			for _, da := range []float64{0.45, -0.45} {
				a := ang + math.Pi + da
				bx := tipX + int(13*math.Cos(a))
				by := tipY + int(13*math.Sin(a))
				c.LineWidth(tipX, tipY, bx, by, 3, col)
			}
			mx, my := (ocx+tipX)/2, (ocy+tipY)/2
			iw := raster.TextWidth(f.LinkID, 2)
			c.FillRect(mx-iw/2-5, my-11, iw+10, 22, raster.Hex(0xFFFFFF))
			c.FrameRect(mx-iw/2-5, my-11, iw+10, 22, 1, palette.border)
			c.Text(f.LinkID, mx-iw/2, my-8, 2, palette.ink)
			continue
		}

		// Unresolved: red dashed stub to a broken plug at the box corner.
		plugX, plugY := or.x+or.w-2, or.y-14
		c.DashedLine(or.cx(), or.y+4, plugX-6, plugY+8, 6, 5, 3, raster.Red)
		c.Disc(plugX, plugY, 11, raster.Hex(0xFFFFFF))
		c.Circle(plugX, plugY, 8, 3, raster.Red)
		c.LineWidth(plugX-4, plugY-4, plugX+4, plugY+4, 3, raster.Red)
		c.LineWidth(plugX+4, plugY-4, plugX-4, plugY+4, 3, raster.Red)
		lw := raster.TextWidth(f.LinkID, 2)
		c.FillRect(plugX-lw/2-5, plugY-34, lw+10, 20, raster.Hex(0xFFFFFF))
		c.Text(f.LinkID, plugX-lw/2, plugY-31, 2, palette.bad)
	}
}

func (p *Poster) annotate(o Options, rects map[string]rect, matByOwner map[string]audit.Finding) {
	for _, f := range o.Report.Findings {
		if f.Status == audit.StatusOK {
			continue
		}
		owner := o.Stage.PrimByID(f.OwnerID)
		if owner == nil {
			continue
		}
		r := rects[owner.ID]
		bx, by, bw, bh := r.x-10, r.y-10, r.w+20, r.h+20
		category := "broken_link"
		if f.Status == audit.StatusDefault {
			category = "retained_default"
		}
		p.redBox(bx, by, bw, bh, Box{
			Category: category, Part: "object", LinkID: f.LinkID,
			Target: f.Target, OldPath: f.OldPath, Label: owner.Tag,
		}, true)

		if mf, isMat := matByOwner[owner.ID]; isMat && mf.Status == audit.StatusDefault {
			chipW, chipH := 52, 26
			cx := bx + bw - chipW + 6
			cy := by - chipH/2
			p.Canvas.FillRect(cx, cy, chipW, chipH, palette.warnBg)
			p.redBox(cx, cy, chipW, chipH, Box{
				Category: "retained_default", Part: "scene_badge", LinkID: f.LinkID,
				Target: f.Target, OldPath: f.OldPath, Label: "DEFAULT",
			}, false)
			p.Canvas.Text("DEF", cx+8, cy+5, 2, raster.Hex(0x9A6700))
		}
	}
}

func (p *Poster) drawLegend(x, y, w int) {
	c := p.Canvas
	c.FillRect(x, y, w, 76, raster.Hex(0xFFFFFF))
	c.FrameRect(x, y, w, 76, 1, palette.border)

	row1Y := y + 18
	cx := x + 24
	items := []struct {
		label string
		col   raster.Color
		kind  scene.LinkKind
	}{
		{"reference", palette.ref, scene.KindReference},
		{"material binding", palette.mat, scene.KindMaterialBinding},
		{"camera target", palette.cam, scene.KindCameraTarget},
	}
	for _, it := range items {
		if it.kind == scene.KindReference {
			c.DashedLine(cx, row1Y+8, cx+44, row1Y+8, 10, 7, 3, it.col)
		} else {
			c.LineWidth(cx, row1Y+8, cx+44, row1Y+8, 3, it.col)
		}
		c.Text(it.label, cx+54, row1Y, 2, palette.ink)
		cx += 54 + raster.TextWidth(it.label, 2) + 34
	}

	row2Y := y + 48
	c.FrameRect(x+24, row2Y, 34, 20, 4, raster.Red)
	c.Text("red box = dangling link object or retained default value", x+70, row2Y+1, 2, palette.ink)
}
