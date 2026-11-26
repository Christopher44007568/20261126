package viz

import (
	"fmt"

	"nsmigrate/audit"
	"nsmigrate/raster"
	"nsmigrate/scene"
)

func (p *Poster) drawSide(o Options, cardsTop int, hs []int, gap int) {
	c := p.Canvas

	c.Text("LINK INTEGRITY LEDGER", sideX+24, contentY+20, 3, palette.ink)
	c.FillRect(sideX+20, contentY+58, sideW-40, 2, palette.border)

	bx, by, bw, bh := sideX+16, contentY+66, sideW-32, 64
	pass := o.Report.InvariantOK
	if pass {
		c.FillRect(bx, by, bw, bh, palette.okBg)
		c.FrameRect(bx, by, bw, bh, 2, palette.ok)
		msg := "PASS - in-scope reference, material binding and camera target resolve"
		c.TextWrapped(msg, bx+16, by+13, 2, bw-32, palette.ok)
	} else {
		c.FillRect(bx, by, bw, bh, palette.badBg)
		c.FrameRect(bx, by, bw, bh, 2, palette.bad)
		msg := "FAIL - dangling arcs; each card quotes the concrete OLD namespace path"
		c.TextWrapped(msg, bx+16, by+13, 2, bw-32, palette.bad)
	}

	y := cardsTop
	for i, f := range o.Report.Findings {
		p.drawCard(cardX, y, cardW, f)
		y += hs[i] + gap
	}
}

func statusText(s audit.Status) string {
	switch s {
	case audit.StatusOK:
		return "OK"
	case audit.StatusBroken:
		return "BROKEN"
	case audit.StatusDefault:
		return "DEFAULT"
	}
	return string(s)
}

func (p *Poster) drawCard(x, y, w int, f audit.Finding) {
	c := p.Canvas
	h := cardHeight(f)
	ok := f.Status == audit.StatusOK

	c.FillRect(x, y, w, h, palette.panel)
	c.FrameRect(x, y, w, h, 1, palette.border)
	stripe := kindColor(string(f.Kind))
	if !ok {
		stripe = palette.bad
	}
	c.FillRect(x, y, 6, h, stripe)

	// ID badge.
	badgeX, badgeY, badgeW, badgeH := x+16, y+14, 42, 28
	c.FillRect(badgeX, badgeY, badgeW, badgeH, kindColor(string(f.Kind)))
	idW := raster.TextWidth(f.LinkID, 2)
	c.Text(f.LinkID, badgeX+(badgeW-idW)/2, badgeY+6, 2, raster.Hex(0xFFFFFF))

	c.Text(kindLabel(string(f.Kind)), x+68, y+20, 2, palette.ink)

	// Status pill (red-framed for broken / defaulted).
	pText := statusText(f.Status)
	pillW := raster.TextWidth(pText, 2) + 28
	pillH := 28
	pillX := x + w - 14 - pillW
	pillY := y + 14
	switch f.Status {
	case audit.StatusOK:
		c.FillRect(pillX, pillY, pillW, pillH, palette.okBg)
		c.FrameRect(pillX, pillY, pillW, pillH, 2, palette.ok)
		c.Text(pText, pillX+14, pillY+6, 2, palette.ok)
	case audit.StatusBroken:
		c.FillRect(pillX, pillY, pillW, pillH, palette.badBg)
		p.redBox(pillX, pillY, pillW, pillH, Box{
			Category: "broken_link", Part: "status_pill", LinkID: f.LinkID,
			Target: f.Target, OldPath: f.OldPath, Label: pText,
		}, false)
		c.Text(pText, pillX+14, pillY+6, 2, palette.bad)
	case audit.StatusDefault:
		c.FillRect(pillX, pillY, pillW, pillH, palette.warnBg)
		p.redBox(pillX, pillY, pillW, pillH, Box{
			Category: "retained_default", Part: "status_pill", LinkID: f.LinkID,
			Target: f.Target, OldPath: f.OldPath, Label: pText,
		}, false)
		c.Text(pText, pillX+14, pillY+6, 2, raster.Hex(0x9A6700))
	}

	c.FillRect(x+14, y+48, w-28, 1, palette.border)

	tx := x + 16
	c.Text(f.OwnerPath, tx, y+56, 2, palette.ink)

	oldX := c.Text("old: ", tx, y+80, 2, palette.faint)
	c.Text(f.OldPath, oldX, y+80, 2, raster.Hex(0x9A3412))

	newPrefix := "now: "
	newCol := palette.ok
	if !ok {
		newCol = palette.bad
	}
	newX := c.Text(newPrefix, tx, y+104, 2, palette.faint)
	c.Text(f.Target, newX, y+104, 2, newCol)

	cur := y + 128
	if f.StalePath != "" {
		staleX := c.Text("stale: ", tx, cur, 2, palette.bad)
		c.Text(f.StalePath, staleX, cur, 2, palette.bad)
		cur += 24
	}

	if f.Status == audit.StatusDefault {
		hex := fmt.Sprintf("#%02X%02X%02X", scene.DefaultMaterialColor.R,
			scene.DefaultMaterialColor.G, scene.DefaultMaterialColor.B)
		label := "DEFAULT " + scene.DefaultMaterialName + " " + hex
		tw := raster.TextWidth(label, 2)
		chipH := 32
		chipW := 12 + 20 + 10 + tw + 16
		if chipW > cardInnerW {
			chipW = cardInnerW
		}
		c.FillRect(tx, cur, chipW, chipH, raster.Hex(0xF1F4F9))
		c.FillRect(tx+12, cur+7, 18, 18, cvt(scene.DefaultMaterialColor))
		c.FrameRect(tx+12, cur+7, 18, 18, 1, palette.muted)
		c.Text(label, tx+42, cur+8, 2, palette.ink)
		p.redBox(tx, cur, chipW, chipH, Box{
			Category: "retained_default", Part: "value_chip", LinkID: f.LinkID,
			Target: f.Target, OldPath: f.OldPath, Label: label,
		}, false)
		cur += 40
	}

	if f.Hint != "" {
		c.TextWrapped(f.Hint, tx, cur+4, 1, cardInnerW, palette.muted)
	}
}
