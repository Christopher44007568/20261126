package viz

import (
	"nsmigrate/audit"
	"nsmigrate/raster"
)

// Render builds the poster for one stage + audit report.
func Render(o Options) *Poster {
	cardGap := 12
	cardsTop := contentY + 142

	cardHs := make([]int, len(o.Report.Findings))
	cardsH := 0
	for i, f := range o.Report.Findings {
		cardHs[i] = cardHeight(f)
		cardsH += cardHs[i]
		if i < len(o.Report.Findings)-1 {
			cardsH += cardGap
		}
	}
	panelH := (cardsTop - contentY) + cardsH + 24
	if panelH < 760 {
		panelH = 760
	}
	winH := contentY + panelH + margin

	c := raster.New(winW, winH, palette.page)
	p := &Poster{Canvas: c, Width: winW, Height: winH}

	p.drawHeader(o)

	// Panels.
	c.FillRect(margin, contentY, sceneW, panelH, palette.panel)
	c.FrameRect(margin, contentY, sceneW, panelH, 2, palette.border)
	c.FillRect(sideX, contentY, sideW, panelH, palette.panel)
	c.FrameRect(sideX, contentY, sideW, panelH, 2, palette.border)

	p.drawScene(o, panelH)
	p.drawSide(o, cardsTop, cardHs, cardGap)

	for i := range p.Boxes {
		p.Boxes[i].Image = o.ImageName
	}
	return p
}

func (p *Poster) drawHeader(o Options) {
	c := p.Canvas
	c.FillRect(0, 0, winW, headerH, raster.Hex(0x101828))
	c.FillRect(0, headerH-6, winW, 6, accentForMode(o.Mode))

	c.Text(o.Title, margin+8, 26, 3, raster.Hex(0xF8FAFC))
	sub := o.Subtitle
	c.Text(sub, margin+10, 82, 2, raster.Hex(0xC7D2E2))

	label := modeBadge(o.Mode)
	bw := raster.TextWidth(label, 2) + 36
	bx := winW - margin - bw
	c.FillRect(bx, 30, bw, 40, accentForMode(o.Mode))
	c.Text(label, bx+18, 40, 2, raster.Hex(0x0B1220))
}

func accentForMode(mode string) raster.Color {
	switch mode {
	case "correct":
		return raster.Hex(0x34D399)
	case "string-only":
		return raster.Red
	default:
		return raster.Hex(0x60A5FA)
	}
}

func modeBadge(mode string) string {
	switch mode {
	case "correct":
		return "AFTER / CORRECT"
	case "string-only":
		return "AFTER / STRING-ONLY"
	default:
		return "BEFORE"
	}
}

func kindLabel(k string) string {
	switch k {
	case "reference":
		return "REFERENCE"
	case "material_binding":
		return "MATERIAL BINDING"
	case "camera_target":
		return "CAMERA TARGET"
	}
	return k
}

func kindColor(k string) raster.Color {
	switch k {
	case "reference":
		return palette.ref
	case "material_binding":
		return palette.mat
	case "camera_target":
		return palette.cam
	}
	return palette.muted
}

func cardHeight(f audit.Finding) int {
	bottom := 124 // three base text rows + gap
	if f.StalePath != "" {
		bottom += 24
	}
	if f.Status == audit.StatusDefault {
		bottom += 42 // default value chip
	}
	if f.Hint != "" {
		lines := raster.Wrap(f.Hint, 1, cardInnerW)
		bottom += 6 + len(lines)*12
	}
	return bottom + 12
}
