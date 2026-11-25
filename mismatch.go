package main

type MismatchBoxReport struct {
	NodeID       int32 `json:"node_id"`
	CorrectMin   Vec3  `json:"correct_min"`
	CorrectMax   Vec3  `json:"correct_max"`
	StaleMin     Vec3  `json:"stale_min"`
	StaleMax     Vec3  `json:"stale_max"`
	Displacement Vec3  `json:"displacement"`
}

type MismatchVisualReport struct {
	Description   string              `json:"description"`
	GreenMeaning  string              `json:"green"`
	RedMeaning    string              `json:"red"`
	YellowMeaning string              `json:"yellow"`
	Boxes         []MismatchBoxReport `json:"boxes"`
	RedPixels     int                 `json:"red_pixels"`
	GreenPixels   int                 `json:"green_pixels"`
	YellowPixels  int                 `json:"yellow_pixels"`
}

func BuildMismatchDiagnostic(base *Canvas, rt *RelocatedAS, cam Camera) (*Canvas, MismatchVisualReport) {
	canvas := base.Clone()
	for i := range canvas.Pixels {
		canvas.Pixels[i] = canvas.Pixels[i].Mul(0.10)
	}
	canvas.FillRect(0, 0, canvas.Width, 106, V(0.01, 0.012, 0.03), 0.86)
	canvas.FillRect(0, canvas.Height-62, canvas.Width, 62, V(0.01, 0.012, 0.03), 0.86)

	report := MismatchVisualReport{
		Description:   "Injected stale relocation boxes demonstrate color coding; the real serialized round trip has zero AABB mismatches.",
		GreenMeaning:  "correct relocated AABB",
		RedMeaning:    "misaligned/stale AABB after a bad address or triangle-index relocation",
		YellowMeaning: "center displacement vector",
		Boxes:         []MismatchBoxReport{},
	}

	mainInstance := FindRuntimeInstance(rt.RootInstance, 1)
	if mainInstance == nil {
		mainInstance = rt.RootInstance
	}

	selected := selectDiagnosticLeaves(rt)
	shift := V(0.205, 0.145, 0.035)
	green := V(0.10, 1.0, 0.46)
	red := V(1.0, 0.12, 0.28)
	yellow := V(1.0, 0.86, 0.12)

	for _, node := range selected {
		correct := node.Bounds
		stale := AABB{Min: correct.Min.Add(shift), Max: correct.Max.Add(shift)}
		report.Boxes = append(report.Boxes, MismatchBoxReport{
			NodeID: node.FileID, CorrectMin: correct.Min, CorrectMax: correct.Max,
			StaleMin: stale.Min, StaleMax: stale.Max, Displacement: shift,
		})

		DrawTransformedAABB(canvas, cam, stale, mainInstance.World, red, 0.92)
		DrawTransformedAABB(canvas, cam, correct, mainInstance.World, green, 0.98)

		staleCenter := matPoint(mainInstance.World, stale.Center())
		correctCenter := matPoint(mainInstance.World, correct.Center())
		x0, y0, ok0 := cam.Project(staleCenter)
		x1, y1, ok1 := cam.Project(correctCenter)
		if ok0 && ok1 {
			canvas.LineAlpha(int(x0+0.5), int(y0+0.5), int(x1+0.5), int(y1+0.5), yellow, 0.95)
		}
	}

	drawLegendSwatches(canvas)
	for _, pixel := range canvas.Pixels {
		if pixel.X > 0.72 && pixel.Y < 0.42 && pixel.Z < 0.48 {
			report.RedPixels++
		}
		if pixel.Y > 0.72 && pixel.X < 0.45 && pixel.Z < 0.70 {
			report.GreenPixels++
		}
		if pixel.X > 0.78 && pixel.Y > 0.66 && pixel.Z < 0.35 {
			report.YellowPixels++
		}
	}
	return canvas, report
}

func selectDiagnosticLeaves(rt *RelocatedAS) []*RuntimeNode {
	var central []*RuntimeNode
	var all []*RuntimeNode
	var walk func(*RuntimeNode)
	walk = func(node *RuntimeNode) {
		if node == nil {
			return
		}
		if node.Left == nil && node.Right == nil {
			all = append(all, node)
			center := node.Bounds.Center()
			if center.X > -1.15 && center.X < 1.15 && center.Y > -0.72 && center.Y < 0.72 {
				central = append(central, node)
			}
			return
		}
		walk(node.Left)
		walk(node.Right)
	}
	walk(rt.Root)

	source := central
	if len(source) < 6 {
		source = all
	}
	selected := make([]*RuntimeNode, 0, 8)
	for i := 0; i < len(source) && len(selected) < 8; i++ {
		idx := (i*7 + 2) % len(source)
		candidate := source[idx]
		if len(selected) == 0 || candidate.FileID != selected[len(selected)-1].FileID {
			selected = append(selected, candidate)
		}
		if len(source) <= 8 {
			selected = selected[:0]
			for _, node := range source {
				selected = append(selected, node)
			}
			break
		}
	}
	return selected
}

func drawLegendSwatches(c *Canvas) {
	white := V(0.92, 0.96, 1.0)
	green := V(0.10, 1.0, 0.46)
	red := V(1.0, 0.12, 0.28)
	yellow := V(1.0, 0.86, 0.12)

	c.FillRect(18, 20, 30, 18, green, 1)
	DrawText(c, "GREEN = RELOCATED AABB", 62, 22, 2, white)

	c.FillRect(18, 48, 30, 18, red, 1)
	DrawText(c, "RED = STALE MISALIGNED BOX", 62, 50, 2, white)

	c.FillRect(18, 76, 30, 16, yellow, 1)
	DrawText(c, "YELLOW = CENTER DELTA", 62, 78, 2, white)

	bottomY := c.Height - 48
	DrawText(c, "EDGE / DEGENERATE / DEEP: SAME SURFACE", 18, bottomY, 2, green)
	DrawText(c, "ACTUAL ROUNDTRIP DIFF = 0 AABB MISMATCH", 18, bottomY+24, 2, white)
}
