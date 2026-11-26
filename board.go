package main

// BoardSpec carries the copy and data for one proof board.
type BoardSpec struct {
	Key          string
	Title        string
	Subtitle     string
	Footer       string
	FooterAccent colorRGBA
	Stage        Stage
}

// RenderBoard composites a full 1080x720 proof image.
func RenderBoard(spec BoardSpec) *Canvas {
	c := NewCanvas(boardW, boardH, colBgTop)
	c.VGradient(0, 0, boardW, boardH, colBgTop, colBgBottom)

	c.FillRoundRect(26, 24, 9, 42, 3, colAmber)
	c.DrawText(spec.Title, 52, 18, 4, colInk, 2)
	c.DrawText(spec.Subtitle, 52, 58, 2, colMuted, 1)

	drawCard(c, 0, "REFERENCE INSTANCE", spec.Stage.Live)
	drawCard(c, 1, "INDEPENDENT COPY", spec.Stage.Archive)

	c.FillRect(24, 657, boardW-48, 1, withAlpha(spec.FooterAccent, 150))
	drawFooter(c, spec.Footer, spec.FooterAccent)
	return c
}

func drawFooter(c *Canvas, text string, accent colorRGBA) {
	maxW := boardW - 48
	scale := 3
	tracking := 0
	if Measure(text, scale, tracking) > maxW {
		scale = 2
		tracking = 1
	}
	c.DrawTextFit(text, 24, 674, scale, accent, tracking, maxW)
}

func slotOK(s SlotReport) bool { return s.Error == "" }

func worstStatus(rep SceneReport) colorRGBA {
	if slotOK(rep.Material) && slotOK(rep.Geometry) && slotOK(rep.Camera) {
		return colGreen
	}
	return colRed
}

func drawCard(c *Canvas, idx int, title string, rep SceneReport) {
	x, y, w, h := cardRect(idx)
	status := worstStatus(rep)

	c.FillRoundRect(x, y, w, h, 18, colCard)
	c.StrokeRoundRect(x, y, w, h, 18, 2, colCardEdge)

	c.FillRoundRect(x+20, y+24, 6, 38, 3, status)
	c.DrawText(title, x+38, y+22, 3, colInk, 1)
	c.DrawText(rep.Name, x+39, y+54, 1, colMuted, 1)

	vp := rect{x: x + 16, y: y + 76, w: w - 32, h: 298}
	drawViewport(c, vp, rep)

	labelX := x + 20
	tagX := x + 20
	pathX := x + 42
	rightX := x + w - 12

	c.DrawText("FINAL PATH SOURCE", labelX, y+398, 1, colAmber, 1)

	rows := []struct {
		tag string
		rep SlotReport
		top int
	}{
		{"MAT", rep.Material, y + 420},
		{"GEO", rep.Geometry, y + 440},
		{"CAM", rep.Camera, y + 460},
	}
	for _, row := range rows {
		tagCol := colGreen
		if row.rep.Error != "" {
			tagCol = colRed
		}
		c.DrawText(row.tag, tagX, row.top, 1, tagCol, 0)
		c.DrawTextFit(row.rep.Source, pathX, row.top, 1, colBody, 0, rightX-pathX)
	}

	chipTop := y + 494
	c.FillRect(labelX, chipTop+1, 12, 12, status)
	chip := chipText(rep)
	chipScale := 2
	chipTrack := 0
	if Measure(chip, chipScale, chipTrack) > rightX-(labelX+20) {
		chipScale = 1
		chipTrack = 1
	}
	c.DrawTextFit(chip, labelX+20, chipTop, chipScale, status, chipTrack, rightX-(labelX+20))

	helper := "ONLY REFERENCED PATHS ARE RE-EVALUATED / INDEPENDENT COPIES STAY FIXED"
	c.DrawTextFit(helper, labelX, y+528, 1, colFaint, 0, rightX-labelX)
}

func chipText(rep SceneReport) string {
	var bad []string
	if rep.Material.Error != "" {
		bad = append(bad, "MISSING MATERIAL")
	}
	if rep.Geometry.Error != "" {
		bad = append(bad, "GEOMETRY")
	}
	if rep.Camera.Error != "" {
		bad = append(bad, "CAMERA")
	}
	if len(bad) == 0 {
		return "ALL REFERENCES RESOLVED"
	}
	out := "ERROR OVERLAY: "
	for i, b := range bad {
		if i > 0 {
			out += " + "
		}
		out += b
	}
	return out
}

func drawViewport(c *Canvas, vp rect, rep SceneReport) {
	c.FillRoundRect(vp.x, vp.y, vp.w, vp.h, 14, colViewport)

	if rep.Geometry.Error != "" {
		drawMissingPanel(c, vp, rep.Geometry)
		c.StrokeRoundRect(vp.x, vp.y, vp.w, vp.h, 14, 2, colVPEdge)
		return
	}

	drawShowcase(c, vp)
	drawViewportChrome(c, vp)
	if rep.Material.Error != "" || rep.Camera.Error != "" {
		drawAssetErrorOverlay(c, vp, rep)
	}
	c.StrokeRoundRect(vp.x, vp.y, vp.w, vp.h, 14, 2, colVPEdge)
}

func drawViewportChrome(c *Canvas, vp rect) {
	c.DrawText("GEO", vp.x+16, vp.y+14, 1, colFaint, 0)
	right := vp.x + vp.w - 16
	camLabel := Ellipsize("CAM FALLBACK CAMERA / HERO 35", 1, 0, 250)
	c.DrawTextRight(camLabel, right, vp.y+14, 1, colFaint, 0)
	c.DrawText("MAT FALLBACK SURFACE", vp.x+16, vp.y+vp.h-22, 1, colFaint, 0)
}

func drawAssetErrorOverlay(c *Canvas, vp rect, rep SceneReport) {
	drawHatch(c, vp, HexA("#FF3B5C", 46), 24)

	message := "ERROR OVERLAY / MISSING MATERIAL"
	if rep.Material.Error == "" && rep.Camera.Error != "" {
		message = "ERROR OVERLAY / MISSING CAMERA"
	}
	banner := rect{x: vp.x, y: vp.y + 34, w: vp.w, h: 42}
	prev := c.Clip
	c.Clip = vp.imgRect()
	c.FillRect(banner.x, banner.y, banner.w, banner.h, colBanner)
	c.Clip = prev
	c.DrawTextFit(message, banner.x+18, banner.y+14, 2, colInk, 0, banner.w-36)
}
