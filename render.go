package main

import (
	"fmt"
	"strings"
)

const (
	canvasW = 1040
	canvasH = 760
)

var (
	colBackground = Color3{R: 0.035, G: 0.05, B: 0.10}
	colWhite      = Color3{R: 0.96, G: 0.97, B: 1}
	colMuted      = Color3{R: 0.62, G: 0.68, B: 0.80}
	colInk        = Color3{R: 0.045, G: 0.07, B: 0.14}
	colShadow     = Color3{R: 0.015, G: 0.02, B: 0.05}
	colLine       = Color3{R: 0.16, G: 0.22, B: 0.34}
)

type panelView struct {
	Letter   string
	Name     string
	Path     string
	Child    string
	ChildTag string
}

func RenderScene(scene *Scene) *Canvas {
	c := NewCanvas(canvasW, canvasH, colBackground)
	drawBackdrop(c)
	c.Text(36, 24, "USDA-STYLE LAYER STACK", colWhite, 4)
	c.Text(38, 64, "RANK: "+strengthLadder(scene), colMuted, 2)
	c.Text(38, 88, "LOAD ORDER IS IGNORED FOR OPINIONS; ONLY DECLARED LAYERSTRENGTH WINS", colLine, 1)

	views := []panelView{
		{Letter: "A", Name: "PANEL A", Path: "/STAGE/PANELA", Child: "Knob", ChildTag: "KNOB"},
		{Letter: "B", Name: "PANEL B", Path: "/STAGE/PANELB"},
		{Letter: "C", Name: "PANEL C", Path: "/STAGE/PANELC", Child: "Badge", ChildTag: "BADGE"},
		{Letter: "D", Name: "PANEL D", Path: "/STAGE/PANELD", Child: "Tag", ChildTag: "TAG"},
	}
	positions := [][2]int{{36, 112}, {530, 112}, {36, 346}, {530, 346}}
	for i, view := range views {
		path := "/Stage/Panel" + view.Letter
		comp := scene.Attribute(path, "inputs:displayColor")
		childPath := ""
		if view.Child != "" {
			childPath = path + "/" + view.Child
		}
		drawPanelCard(c, positions[i][0], positions[i][1], 474, 220, view, comp, childPath, scene)
	}

	drawStackProbe(c, 36, 584, 968, 116, scene)
	drawFooter(c, scene)
	return c
}

func drawBackdrop(c *Canvas) {
	for y := 0; y < c.H; y += 32 {
		for x := 0; x < c.W; x += 32 {
			c.FillRect(x, y, 2, 2, colLine)
		}
	}
	c.FillRect(36, 101, 968, 1, colLine)
}

func strengthLadder(scene *Scene) string {
	var parts []string
	for _, layer := range scene.Layers {
		parts = append(parts, fmt.Sprintf("%s S%d", strings.ToUpper(layer.ID), layer.Strength))
	}
	return strings.Join(parts, " > ")
}

func drawPanelCard(c *Canvas, x, y, w, h int, view panelView, comp *AttributeComposition, childPath string, scene *Scene) {
	winner := comp.Winner
	cardColor := winner.Color
	c.FillRoundedRect(x+7, y+8, w, h, 20, colShadow)
	c.FillRoundedRect(x, y, w, h, 20, cardColor)
	c.Text(x+24, y+27, view.Letter, colInk, 5)
	c.Text(x+25, y+26, view.Letter, colWhite, 5)
	c.Text(x+76, y+30, view.Name, colInk, 3)
	c.Text(x+76, y+63, view.Path, colInk, 2)

	if childPath != "" && scene.Winner(childPath, "inputs:displayColor") != nil {
		childWinner := scene.Winner(childPath, "inputs:displayColor")
		cx, cy := x+w-92, y+67
		c.FillCircle(cx+3, cy+4, 29, colShadow)
		c.FillCircle(cx, cy, 29, childWinner.Color)
		c.Text(cx-6, cy-8, string(view.ChildTag[0]), colInk, 3)
		c.TextRight(x+w-24, y+108, view.ChildTag+" -> "+strings.ToUpper(childWinner.LayerID)+" S"+fmt.Sprint(childWinner.Strength), colInk, 1)
		c.TextRight(x+w-24, y+121, strings.ToUpper(childPath), colInk, 1)
	}

	drawOpinionPill(c, x+22, y+h-72, w-44, 54, comp)
}

func drawOpinionPill(c *Canvas, x, y, w, h int, comp *AttributeComposition) {
	c.FillRoundedRect(x, y, w, h, 11, colInk)
	winner := comp.Winner
	c.Text(x+14, y+11, fmt.Sprintf("WINNER: %s (S%d)", strings.ToUpper(winner.LayerID), winner.Strength), colWhite, 2)
	stack := opinionLadder(comp)
	scale := 2
	if c.TextWidth(stack, scale) > w-28 {
		scale = 1
	}
	c.Text(x+14, y+35, stack, colMuted, scale)
}

func opinionLadder(comp *AttributeComposition) string {
	var parts []string
	for _, opinion := range comp.Opinions {
		parts = append(parts, fmt.Sprintf("%s S%d", strings.ToUpper(opinion.LayerID), opinion.Strength))
	}
	return strings.Join(parts, " > ")
}

func drawStackProbe(c *Canvas, x, y, w, h int, scene *Scene) {
	comp := scene.Attribute("/Stage/StackProbe", "inputs:displayColor")
	winner := comp.Winner
	c.FillRoundedRect(x+7, y+8, w, h, 20, colShadow)
	c.FillRoundedRect(x, y, w, h, 20, winner.Color)
	c.Text(x+20, y+13, "ALL 4 LAYERS AUTHOR THIS COLOR: /STAGE/STACKPROBE", colWhite, 2)
	c.Text(x+20, y+34, fmt.Sprintf("STRONGEST OPINION SOURCE: %s S%d - SAME WINNER IN EVERY LOAD ORDER",
		strings.ToUpper(winner.LayerID), winner.Strength), colInk, 1)

	wrap := w - 40
	swatchW := (wrap - 3*10) / 4
	for i, opinion := range comp.Opinions {
		sx := x + 20 + i*(swatchW+10)
		sy := y + 52
		c.FillRoundedRect(sx, sy, swatchW, 49, 8, colInk)
		c.FillRoundedRect(sx+8, sy+10, 27, 27, 5, opinion.Color)
		c.Text(sx+43, sy+8, strings.ToUpper(opinion.LayerID)+" S"+fmt.Sprint(opinion.Strength), colWhite, 1)
		status := "LOSES"
		if i == 0 {
			status = "WINNER"
		}
		c.Text(sx+43, sy+23, status, colMuted, 1)
		c.Text(sx+43, sy+35, fmt.Sprintf("%s:%d", shortFile(opinion.File), opinion.Line), colMuted, 1)
	}
}

func shortFile(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func drawFooter(c *Canvas, scene *Scene) {
	c.FillRect(36, 708, 968, 1, colLine)
	c.Text(36, 716, "24 LOAD-ORDER PERMUTATIONS -> IDENTICAL SCENE + BYTE-IDENTICAL PPM", colWhite, 2)
	c.Text(36, 736, "LOCAL OVER KEEPS CHILD PATHS: /STAGE/PANELA/KNOB  /STAGE/PANELD/TAG  /STAGE/PANELC/BADGE", colMuted, 1)
	winner := scene.Winner("/Stage/StackProbe", "inputs:displayColor")
	c.Text(36, 748, fmt.Sprintf("SOURCE: %s:%d  |  THE 40-STRENGTH OPINION CANNOT BE OVERRIDDEN BY A LATER-LOADED WEAK LAYER",
		winner.File, winner.Line), colMuted, 1)
}
