package main

import (
	"fmt"
	"sort"
)

const (
	diagramWidth  = 1200
	diagramHeight = 680
)

func RenderReuseTimeline(report Report) *Canvas {
	c := NewCanvas(diagramWidth, diagramHeight, colBg)
	drawBackdrop(c, diagramWidth, diagramHeight)
	c.Text(48, 34, 3, colInk, "ACCELERATION STRUCTURE REUSE TIMELINE")
	c.Text(50, 72, 1, colMuted, "THREE FRAMES / TWO ASYNC LIGHT-PATH SEQUENCES PER FRAME / ONE 8-SLOT SHARED ARENA")

	maxTick := reportMaxTick(report)
	if maxTick < 8 {
		maxTick = 8
	}
	left, right := 190, 1130
	tickX := func(t int) int {
		return left + int(float64(t)/float64(maxTick)*float64(right-left))
	}

	legendY := 98
	drawLegend(c, 190, legendY, colGen1, "GEN 1")
	drawLegend(c, 310, legendY, colGen2, "GEN 2")
	drawLegend(c, 430, legendY, colGen3, "GEN 3")
	drawLegend(c, 550, legendY, colYellow, "REUSE WAIT")
	drawLegend(c, 700, legendY, colGreen, "QUERY A")
	drawLegend(c, 820, legendY, colBlue, "QUERY B")

	slotTop, slotRowH := 152, 33
	c.Text(48, slotTop-24, 1, colMuted, "SHARED BUFFER SLOTS")
	for slot := 0; slot < arenaCapacity; slot++ {
		y := slotTop + slot*slotRowH
		c.FillRect(48, y, 128, 25, colPanel)
		c.Text(56, y+6, 1, colMuted, fmt.Sprintf("SLOT %d", slot))
		c.FillRect(left, y+8, right-left, 9, RGB{21, 32, 55})
	}
	for t := 0; t <= maxTick; t++ {
		x := tickX(t)
		c.Line(x, slotTop-4, x, slotTop+arenaCapacity*slotRowH+4, 1, colGrid)
		c.Text(x-5, slotTop+arenaCapacity*slotRowH+16, 1, colMuted, fmt.Sprintf("T%d", t))
	}
	for _, gen := range report.Generations {
		col := generationColor(gen.ID)
		for _, slot := range gen.Slots {
			if slot < 0 || slot >= arenaCapacity {
				continue
			}
			y := slotTop + slot*slotRowH + 8
			x0 := tickX(gen.BuiltAt)
			x1 := tickX(releaseTime(gen))
			if x1 <= x0 {
				x1 = x0 + 3
			}
			c.FillRect(x0, y, x1-x0, 9, col)
			c.Text(x0+5, y-14, 1, col, fmt.Sprintf("G%d", gen.ID))
		}
	}

	highlightReuse(c, report, tickX, slotTop, slotRowH)

	queryTop := 486
	c.Text(48, queryTop-24, 1, colMuted, "LIGHT-PATH QUERY REFS")
	for i, query := range report.Queries {
		y := queryTop + i*23
		col := colGreen
		if query.Sequence == "B" {
			col = colBlue
		}
		c.Text(48, y+4, 1, colMuted, query.ID)
		c.FillRect(left, y+4, right-left, 11, RGB{21, 32, 55})
		x0 := tickX(query.StartTick)
		x1 := tickX(query.DoneTick)
		if x1 <= x0 {
			x1 = x0 + 3
		}
		c.FillRect(x0, y+4, x1-x0, 11, col)
		c.Text(x1+6, y+4, 1, col, fmt.Sprintf("DONE T%d", query.DoneTick))
	}

	drawMilestone(c, "G1 LAST REF", 4, tickX(4), slotTop+arenaCapacity*slotRowH+30, colYellow)
	drawMilestone(c, "G2 LAST REF", 5, tickX(5), slotTop+arenaCapacity*slotRowH+30, colYellow)
	drawMilestone(c, "G3 LAST REF", 7, tickX(7), slotTop+arenaCapacity*slotRowH+30, colYellow)

	bottom := 640
	c.FillRect(40, bottom-8, 1120, 1, colGrid)
	c.Text(48, bottom, 1, colGreen, fmt.Sprintf(
		"SAFE REUSE %d/%d   DANGLING %d   PREMATURE RELEASES %d   GHOST PIXELS %d",
		report.Metrics.SafeReuseEvents, report.Metrics.ReuseEvents,
		report.Metrics.DanglingRefs, report.Metrics.PrematureReleases,
		report.Metrics.GhostPixels,
	))
	return c
}

func drawBackdrop(c *Canvas, w, h int) {
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		col := RGB{
			R: byte(7 + 5*t),
			G: byte(12 + 8*t),
			B: byte(24 + 14*t),
		}
		c.FillRect(0, y, w, 1, col)
	}
	for x := 20; x < w; x += 40 {
		c.Line(x, 0, x-90, h, 1, RGB{12, 22, 40})
	}
}

func drawLegend(c *Canvas, x, y int, col RGB, label string) {
	c.FillRect(x, y, 16, 10, col)
	c.Text(x+23, y+1, 1, colMuted, label)
}

func generationColor(id GenID) RGB {
	switch id {
	case 1:
		return colGen1
	case 2:
		return colGen2
	case 3:
		return colGen3
	default:
		return colInk
	}
}

func releaseTime(gen *Generation) int {
	if gen.ReleasedAt != noTick {
		return gen.ReleasedAt
	}
	return reportMaxGenerationTick(gen) + 1
}

func reportMaxGenerationTick(gen *Generation) int {
	maxTick := gen.BuiltAt
	if gen.RetiredAt > maxTick {
		maxTick = gen.RetiredAt
	}
	if gen.ReleasedAt > maxTick {
		maxTick = gen.ReleasedAt
	}
	if gen.LastRefDoneAt > maxTick {
		maxTick = gen.LastRefDoneAt
	}
	return maxTick
}

func reportMaxTick(report Report) int {
	maxTick := 0
	for _, gen := range report.Generations {
		if t := reportMaxGenerationTick(gen); t > maxTick {
			maxTick = t
		}
	}
	for _, query := range report.Queries {
		if query.DoneTick > maxTick {
			maxTick = query.DoneTick
		}
	}
	for _, event := range report.Events {
		if event.Tick > maxTick {
			maxTick = event.Tick
		}
	}
	return maxTick
}

func highlightReuse(c *Canvas, report Report, tickX func(int) int, slotTop, slotRowH int) {
	if len(report.Reuses) == 0 {
		return
	}
	minWait, maxReuse := report.Reuses[0].WaitStart, report.Reuses[0].ReusedAt
	slots := make(map[int]bool)
	for _, reuse := range report.Reuses {
		if reuse.WaitStart < minWait {
			minWait = reuse.WaitStart
		}
		if reuse.ReusedAt > maxReuse {
			maxReuse = reuse.ReusedAt
		}
		slots[reuse.Slot] = true
	}
	indexes := make([]int, 0, len(slots))
	for slot := range slots {
		indexes = append(indexes, slot)
	}
	sort.Ints(indexes)
	if len(indexes) == 0 {
		return
	}
	y0 := slotTop + indexes[0]*slotRowH - 3
	y1 := slotTop + indexes[len(indexes)-1]*slotRowH + 28
	x0 := tickX(minWait)
	x1 := tickX(maxReuse)
	c.BlendRect(x0, y0, x1-x0, y1-y0, colYellow, 0.16)
	c.DrawRect(x0, y0, x1-x0, y1-y0, 2, colYellow)
	if x1-x0 < 120 {
		c.Text(x0-8, y0-18, 1, colYellow, "WAIT")
	} else {
		c.Text(x0+8, y0-18, 1, colYellow, "REUSE WAIT")
	}
	c.Line(x1, y0-24, x1, y1+2, 2, colYellow)
	c.Text(x1+7, y0-25, 1, colYellow, fmt.Sprintf("REUSED T%d", maxReuse))
}

func drawMilestone(c *Canvas, label string, tick, x, y int, col RGB) {
	c.Line(x, y-8, x, y+12, 2, col)
	c.Text(x+5, y, 1, col, fmt.Sprintf("%s T%d", label, tick))
}
