package main

import (
	"fmt"
	"strings"
)

const (
	graphWidth  = 1240
	graphHeight = 820
)

type graphNode struct {
	x, y, w, h int
}

func (n graphNode) left() int    { return n.x }
func (n graphNode) right() int   { return n.x + n.w }
func (n graphNode) centerY() int { return n.y + n.h/2 }

func RenderReferenceGraph(report Report) *Canvas {
	c := NewCanvas(graphWidth, graphHeight, colBg)
	drawBackdrop(c, graphWidth, graphHeight)
	c.Text(42, 28, 3, colInk, "REFERENCE / DANGLING GRAPH")
	c.Text(44, 67, 1, colMuted, "QUERY REFS -> GENERATIONS -> SHARED SLOTS; RETIREMENT QUEUE DEFERS RELEASE UNTIL REFCOUNT REACHES ZERO")
	drawGraphLegend(c, 44, 94)

	queryNodes := make(map[string]graphNode)
	for i, query := range report.Queries {
		n := graphNode{x: 44, y: 142 + i*78, w: 224, h: 60}
		queryNodes[query.ID] = n
		col := generationColor(query.Generation)
		c.FillRect(n.x, n.y, n.w, n.h, colPanel)
		c.DrawRect(n.x, n.y, n.w, n.h, 1, col)
		c.Text(n.x+10, n.y+8, 1, col, query.ID)
		c.Text(n.x+10, n.y+25, 1, colMuted, fmt.Sprintf("REF G%d  T%d -> T%d", query.Generation, query.StartTick, query.DoneTick))
		c.Text(n.x+10, n.y+42, 1, colGreen, "RELEASED AFTER READ")
	}

	genNodes := make(map[GenID]graphNode)
	for i, gen := range report.Generations {
		n := graphNode{x: 462, y: 142 + i*155, w: 236, h: 104}
		genNodes[gen.ID] = n
		col := generationColor(gen.ID)
		c.FillRect(n.x, n.y, n.w, n.h, colPanel2)
		c.DrawRect(n.x, n.y, n.w, n.h, 2, col)
		c.Text(n.x+10, n.y+8, 2, col, fmt.Sprintf("GENERATION %d", gen.ID))
		c.Text(n.x+10, n.y+34, 1, colMuted, fmt.Sprintf("BUILD T%d   RETIRE T%d", gen.BuiltAt, gen.RetiredAt))
		c.Text(n.x+10, n.y+51, 1, colMuted, fmt.Sprintf("RELEASE T%d   LAST REF T%d", gen.ReleasedAt, gen.LastRefDoneAt))
		c.Text(n.x+10, n.y+70, 1, colGreen, "RELEASE >= LAST QUERY")
		c.Text(n.x+10, n.y+86, 1, colMuted, fmt.Sprintf("ACTIVE REFS %d", gen.ActiveRefs()))
	}

	slotOwners := slotOwnership(report)
	slotNodes := make(map[int]graphNode)
	for slot := 0; slot < arenaCapacity; slot++ {
		n := graphNode{x: 902, y: 138 + slot*57, w: 296, h: 47}
		slotNodes[slot] = n
		c.FillRect(n.x, n.y, n.w, n.h, colPanel)
		owners := slotOwners[slot]
		border := colGrid
		if len(owners) > 1 {
			border = colYellow
		}
		c.DrawRect(n.x, n.y, n.w, n.h, 1, border)
		c.Text(n.x+8, n.y+6, 1, colInk, fmt.Sprintf("SLOT %d", slot))
		if len(owners) == 0 {
			c.Text(n.x+78, n.y+6, 1, colMuted, "FREE")
		}
		for j, owner := range owners {
			line := fmt.Sprintf("G%d T%d-T%d", owner.gen, owner.from, owner.to)
			c.Text(n.x+78+j*104, n.y+6, 1, generationColor(owner.gen), line)
		}
		if len(owners) > 1 {
			c.Text(n.x+8, n.y+26, 1, colYellow, "RECYCLED")
		}
	}

	for _, query := range report.Queries {
		qn := queryNodes[query.ID]
		gn := genNodes[query.Generation]
		if qn.w == 0 || gn.w == 0 {
			continue
		}
		col := generationColor(query.Generation)
		c.Line(qn.right(), qn.centerY(), gn.left(), gn.centerY(), 2, col)
		c.Circle(gn.left(), gn.centerY(), 3, col)
	}

	for _, gen := range report.Generations {
		gn := genNodes[gen.ID]
		col := generationColor(gen.ID)
		for _, slot := range gen.Slots {
			sn, ok := slotNodes[slot]
			if !ok {
				continue
			}
			c.DashedLine(gn.right(), gn.centerY(), sn.left(), sn.centerY(), 1, 5, col)
			c.Circle(sn.left(), sn.centerY(), 2, col)
		}
	}

	for _, dangling := range report.Dangling {
		qn, qok := queryNodes[dangling.Query]
		sn, sok := slotNodes[dangling.Slot]
		if !qok || !sok {
			continue
		}
		c.DashedLine(qn.right(), qn.centerY(), sn.left(), sn.centerY(), 3, 7, colRed)
	}

	drawRetirementQueue(c, report, genNodes)
	drawGraphSummary(c, report)
	return c
}

func drawGraphLegend(c *Canvas, x, y int) {
	drawLegend(c, x, y, colGen1, "G1 REF")
	drawLegend(c, x+105, y, colGen2, "G2 REF")
	drawLegend(c, x+210, y, colGen3, "G3 REF")
	drawLegend(c, x+315, y, colYellow, "RECYCLED SLOT")
	c.FillRect(x+475, y, 18, 4, colRed)
	c.Text(x+500, y+1, 1, colMuted, "DANGLING REF (NONE)")
}

type slotOwner struct {
	gen      GenID
	from, to int
}

func slotOwnership(report Report) map[int][]slotOwner {
	owners := make(map[int][]slotOwner)
	for _, gen := range report.Generations {
		for _, slot := range gen.Slots {
			owners[slot] = append(owners[slot], slotOwner{
				gen: gen.ID, from: gen.BuiltAt, to: releaseTime(gen),
			})
		}
	}
	for slot := range owners {
		if len(owners[slot]) > 1 {
			owners[slot] = owners[slot][len(owners[slot])-2:]
		}
	}
	return owners
}

func drawRetirementQueue(c *Canvas, report Report, gens map[GenID]graphNode) {
	ordered := append([]*Generation(nil), report.Generations...)
	for i := 1; i < len(ordered); i++ {
		prev := gens[ordered[i-1].ID]
		next := gens[ordered[i].ID]
		if prev.h == 0 || next.h == 0 {
			continue
		}
		x := prev.x + 10
		c.Line(x, prev.y+prev.h, x, next.y, 1, colYellow)
		c.Line(x, next.y, x-4, next.y+7, 1, colYellow)
		c.Line(x, next.y, x+4, next.y+7, 1, colYellow)
	}
	c.Text(408, 118, 1, colYellow, "RETIRE QUEUE")
}

func drawGraphSummary(c *Canvas, report Report) {
	y := 690
	c.FillRect(36, y-18, 1168, 1, colGrid)
	statusCol := colGreen
	if report.Metrics.DanglingRefs > 0 || report.Metrics.PrematureReleases > 0 {
		statusCol = colRed
	}
	c.Text(44, y, 2, statusCol, fmt.Sprintf(
		"DANGLING REFERENCES %d   PREMATURE RELEASES %d",
		report.Metrics.DanglingRefs, report.Metrics.PrematureReleases,
	))
	c.Text(44, y+36, 1, colMuted, "QUERY REFS ARE RELEASED AT DONE TICK; GENERATION SLOTS BECOME FREE ONLY AFTER THE RETIREMENT QUEUE SEES REFCOUNT ZERO.")
	c.Text(44, y+54, 1, colGreen, fmt.Sprintf(
		"SAFE REUSE %d/%d   FRAMES %d   GHOST PIXELS %d",
		report.Metrics.SafeReuseEvents, report.Metrics.ReuseEvents,
		report.Metrics.FramesRendered, report.Metrics.GhostPixels,
	))
	var checks []string
	for _, gen := range report.Generations {
		checks = append(checks, fmt.Sprintf("G%d %d>=%d", gen.ID, gen.ReleasedAt, gen.LastRefDoneAt))
	}
	c.Text(44, y+76, 1, colMuted, "RELEASE/LAST-QUERY: "+strings.Join(checks, "  "))
}
