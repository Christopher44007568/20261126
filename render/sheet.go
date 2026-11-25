package render

import (
	"fmt"
	"math"
	"strings"

	"animationtiming/raster"
	"animationtiming/timeline"
)

const (
	sheetOuterX = 14
	sheetGap    = 12
	sheetTop    = 104
	cardW       = 320
	cardH       = 336
)

// SheetDimensions returns the PNG size for a sheet specification.
func SheetDimensions(spec timeline.SheetSpec) (int, int) {
	rows := (spec.FrameCount + spec.Cols - 1) / spec.Cols
	w := sheetOuterX*2 + spec.Cols*cardW + (spec.Cols-1)*sheetGap
	h := sheetTop + rows*cardH + (rows-1)*sheetGap + 22
	return w, h
}

// RenderSheet draws one complete frame contact sheet.
func RenderSheet(scene *timeline.Scene, result timeline.SheetResult) *raster.Canvas {
	spec := result.Spec
	w, h := SheetDimensions(spec)
	c := raster.New(w, h, paper)
	drawSheetHeader(c, result, w)

	for _, frame := range result.Frames {
		col := frame.Index % spec.Cols
		row := frame.Index / spec.Cols
		x := sheetOuterX + col*(cardW+sheetGap)
		y := sheetTop + row*(cardH+sheetGap)
		drawFrameCard(c, scene, result, frame, x, y)
	}
	return c
}

func drawSheetHeader(c *raster.Canvas, result timeline.SheetResult, width int) {
	spec := result.Spec
	c.FillRect(0, 0, width, 94, raster.Hex(0x101826))
	c.FillRect(0, 94, width, 4, blue)
	c.Text(strings.ToUpper(spec.Title), 24, 16, 3, white)
	c.Text(spec.Blurb, 25, 52, 2, raster.Hex(0xb8c4d6))

	var summary string
	if spec.ReadMode == timeline.ModeLinear {
		reached, exact := 0, 0
		for _, k := range result.KeyCoverage {
			if k.ShownInFrame || k.SweptByInterpolation {
				reached++
			}
			if k.ShownInFrame {
				exact++
			}
		}
		summary = fmt.Sprintf("LINEAR BRACKETS REACH %d/%d KEYS - EXACT-KEY FRAMES %d - POSES NEVER POP",
			reached, len(result.KeyCoverage), exact)
	} else {
		shown := 0
		for _, k := range result.KeyCoverage {
			if k.ShownInFrame {
				shown++
			}
		}
		skipped := strings.Join(result.SkippedSignatureKeys, ", ")
		if skipped == "" {
			skipped = "NONE"
		}
		summary = fmt.Sprintf("KEYS SHOWN %d/%d - SKIPPED SIGNATURE: %s",
			shown, len(result.KeyCoverage), skipped)
	}
	summary = strings.ToUpper(summary)
	col := green
	if len(result.SkippedSignatureKeys) > 0 {
		col = gold
	}
	c.Text(summary, 25, 76, 2, col)
}

func drawFrameCard(c *raster.Canvas, scene *timeline.Scene, result timeline.SheetResult, frame timeline.Frame, x, y int) {
	primary, _ := frame.Node(scene.PrimaryID)
	bg, accent, border, badge := statusColors(primary.Status)

	c.FillRect(x, y, cardW, cardH, card)
	stageFill := raster.Hex(0xf1f8f4)
	if primary.Status == timeline.StatusBefore {
		stageFill = amberPale
	} else if primary.Status == timeline.StatusAfter {
		stageFill = redPale
	}
	c.FillRect(x+3, y+66, cardW-6, 148, stageFill)

	c.Text(fmt.Sprintf("#%02d %.3fS", frame.Index, frame.PlaybackTime), x+12, y+12, 3, ink)
	badgeW := raster.TextWidth(badge, 1) + 14
	c.FillRect(x+cardW-badgeW-10, y+12, badgeW, 20, bg)
	c.Text(badge, x+cardW-badgeW-3, y+18, 1, accent)

	clock := fmt.Sprintf("OUT %.3f -> SRC %.3f", frame.PlaybackTime, frame.SourceTime)
	c.Text(clock, x+13, y+47, 1, softInk)

	drawStaticPill(c, x+14, y+74)
	drawStaticPill(c, x+cardW-70, y+74)

	buddy, _ := frame.Node("buddy")
	drawBeacon(c, x+43, y+207)
	drawHero(c, x+158, y+210, primary.Pose)
	drawBuddy(c, x+277, y+207, buddy.Pose)

	drawFrameTimeline(c, scene, result, frame, x, y)
	c.FrameRect(x, y, cardW, cardH, 3, border)
}

func drawStaticPill(c *raster.Canvas, x, y int) {
	c.FillRect(x, y, 56, 17, raster.Hex(0xe4eaf2))
	c.Text("STATIC", x+8, y+5, 1, softInk)
}

func drawFrameTimeline(c *raster.Canvas, scene *timeline.Scene, result timeline.SheetResult, frame timeline.Frame, x, y int) {
	heroNode, _ := scene.NodeByID(scene.PrimaryID)
	samples := heroNode.Samples
	first := samples[0].Time
	last := samples[len(samples)-1].Time
	trackL, trackR := x+38, x+284
	trackY := y + 240
	timeX := func(t float64) int {
		if last == first {
			return (trackL + trackR) / 2
		}
		v := (t - first) / (last - first)
		v = math.Max(0, math.Min(1, v))
		return trackL + int(math.Round(v*float64(trackR-trackL)))
	}

	c.LineWidth(x+17, trackY, trackL, trackY, 5, amber)
	c.LineWidth(trackL, trackY, trackR, trackY, 5, raster.Hex(0x9ed8bf))
	c.LineWidth(trackR, trackY, x+303, trackY, 5, red)
	c.FillRect(trackL-1, trackY-8, 2, 16, softInk)
	c.FillRect(trackR-1, trackY-8, 2, 16, softInk)

	c.Text(frameAction(frame, result), x+11, y+222, 1, ink)

	for i, sample := range samples {
		mx := timeX(sample.Time)
		cov := result.KeyCoverage[i]
		if sample.Signature {
			c.Circle(mx, trackY, 8, 2, gold)
		}
		switch {
		case cov.Skipped && sample.Signature:
			diamond(c, mx, trackY, 7, red)
			c.LineWidth(mx-3, trackY-3, mx+3, trackY+3, 2, white)
			c.LineWidth(mx+3, trackY-3, mx-3, trackY+3, 2, white)
		case cov.ShownInFrame:
			diamond(c, mx, trackY, 6, green)
		case cov.SweptByInterpolation:
			diamond(c, mx, trackY, 6, blue)
		default:
			diamond(c, mx, trackY, 4, raster.Hex(0x7c8aa0))
		}
	}

	playX := timeX(frame.SourceTime)
	c.LineWidth(playX, trackY-9, playX, trackY+12, 2, ink)
	diamond(c, playX, trackY, 4, white)
	c.Circle(playX, trackY, 5, 1, blueDark)

	c.Text(fmt.Sprintf("FIRST %.2f", first), x+12, y+255, 1, softInk)
	lastLabel := fmt.Sprintf("LAST %.2f", last)
	c.TextRight(lastLabel, x+cardW-12, y+255, 1, softInk)

	c.FillRect(x+10, y+278, cardW-20, 1, grayLine)
	c.Text("STATIC NODES OWN NO TIME SAMPLES", x+12, y+289, 1, softInk)
	c.Text("BUDDY=BIND POSE / BEACON=BIND POSE", x+12, y+307, 1, softInk)
}

func frameAction(frame timeline.Frame, result timeline.SheetResult) string {
	primary, _ := frame.Node("hero")
	sampleName := func(i int) string {
		if i < 0 {
			return "?"
		}
		// Names are duplicated in coverage and sample order.
		for _, k := range result.KeyCoverage {
			if k.Index == i {
				return k.Name
			}
		}
		return "?"
	}
	switch primary.Status {
	case timeline.StatusBefore:
		return strings.ToUpper(fmt.Sprintf("< BEFORE FIRST - CLAMP K%d %s", primary.ShownKey, sampleName(primary.ShownKey)))
	case timeline.StatusAfter:
		return strings.ToUpper(fmt.Sprintf("AFTER LAST > - CLAMP K%d %s", primary.ShownKey, sampleName(primary.ShownKey)))
	default:
		if result.Spec.ReadMode == timeline.ModeLinear && primary.ShownKey < 0 {
			return strings.ToUpper(fmt.Sprintf("LERP K%d-K%d A%.2f", primary.Lo, primary.Hi, primary.Alpha))
		}
		return strings.ToUpper(fmt.Sprintf("HOLD K%d %s", primary.ShownKey, sampleName(primary.ShownKey)))
	}
}
