package render

import (
	"fmt"
	"io"

	"instancelab/internal/lab"
)

// WritePrototypeDiagram renders the shared prototype with its five nodes,
// default colors and local-coordinate geometry.
func WritePrototypeDiagram(w io.Writer, proto *lab.Prototype) error {
	s := NewSVG(w, 1280, 820, "原型图：Beacon Card", "一个原型包含五个节点和默认颜色；上百个实例只引用原型 ID，不复制节点树。")
	s.Header("原型图 · Beacon Card（proto-beacon-card）",
		"实例通过稳定 ID 引用该原型；颜色与可见性差异保存在实例各自的 map 中，原型本身保持只读。")

	s.Panel(48, 120, 560, 640, "原型节点（本地坐标）")
	scale := 2.2
	ox, oy := 140.0, 210.0
	for _, node := range proto.Nodes {
		x := ox + node.Geometry.X*scale
		y := oy + node.Geometry.Y*scale
		w := node.Geometry.Width * scale
		h := node.Geometry.Height * scale
		switch node.Kind {
		case "circle":
			s.Circle(x, y, w, node.Fill)
		case "line":
			s.Line(x, y+h/2, x+w, y+h/2, node.Fill, h)
		default:
			s.Rect(x, y, w, h, node.Fill, "", 0, 10)
		}
		s.Text(x, y+h+16, node.ID, 11, SubInk, 700)
	}
	s.DashedRect(ox, oy, 180*scale, 140*scale, Ink, 1.5)
	s.Text(ox, oy-12, "原型包围盒 Bounds（节点几何并集）", 12, Ink, 700)

	s.Panel(648, 120, 584, 640, "默认颜色表（原型唯一事实来源）")
	y := 190.0
	for _, node := range proto.Nodes {
		s.Rect(680, y, 18, 18, node.Fill, Line, 1, 4)
		s.Text(712, y+14, fmt.Sprintf("%s · %s", node.ID, node.Label), 13, Ink, 700)
		s.Text(940, y+14, node.Fill, 13, SubInk, 600)
		y += 48
	}
	s.Text(680, y+30, "实例无本地覆盖时回落到此表；本地覆盖仅写入实例自身 map。", 12, SubInk, 500)
	s.Text(680, y+54, "实例结构：ID + PrototypeID + Offset + ColorOverrides + HiddenNodes", 12, SubInk, 500)
	return s.Close()
}

// WriteInstancesOverview renders all instances as a compact grid, flagging
// local color overrides and hidden cards/nodes.
func WriteInstancesOverview(w io.Writer, doc *lab.Document) error {
	inst := doc.Instances()
	s := NewSVG(w, 1280, 1560, "实例总览图",
		fmt.Sprintf("%d 个实例共享一个原型；标记本地颜色覆盖、整卡隐藏和单节点隐藏。", len(inst)))
	s.Header(fmt.Sprintf("实例总览 · %d 个实例引用同一原型", len(inst)),
		"每张卡片是一个实例的快照；边框为品红表示有本地颜色覆盖，斜纹填充表示隐藏，角标 H 表示单节点隐藏。")

	cols := 8
	cellW, cellH := 142.0, 158.0
	startX, startY := 48.0, 130.0
	meta := doc.Meta()

	s.Chip(48, 96, 190, fmt.Sprintf("本地颜色覆盖 %d 项", meta.ColorOverrides), ErrorPink)
	s.Chip(250, 96, 170, fmt.Sprintf("整卡隐藏 %d 个", meta.HiddenInstances), Line)
	s.Chip(432, 96, 180, fmt.Sprintf("单节点隐藏 %d 项", meta.HiddenNodes), Amber)

	for i, in := range inst {
		col := i % cols
		row := i / cols
		x := startX + float64(col)*(cellW+8)
		y := startY + float64(row)*(cellH+10)

		hasColor := len(in.ColorOverrides) > 0
		stroke := Line
		strokeW := 1.0
		if hasColor {
			stroke = ErrorPink
			strokeW = 2.2
		}
		s.Rect(x, y, cellW, cellH, Panel, stroke, strokeW, 12)

		if in.Hidden {
			s.Rect(x+1, y+1, cellW-2, cellH-2, "url(#hidden-grid)", Line, 1, 12)
			s.Text(x+cellW/2, y+cellH/2, "HIDDEN", 12, SubInk, 800)
		} else {
			drawMiniCard(s, doc, in, x+12, y+34)
		}
		s.Text(x+10, y+18, in.ID, 11, Ink, 800)
		if hasColor {
			s.Rect(x+cellW-22, y+9, 12, 12, ErrorPink, "", 0, 3)
		}
	}
	return s.Close()
}

func drawMiniCard(s *SVG, doc *lab.Document, in *lab.Instance, x, y float64) {
	snap, err := doc.Snapshot(in.ID)
	if err != nil {
		return
	}
	scale := 0.62
	for _, node := range snap.Nodes {
		if !node.Visible {
			continue
		}
		// Mini cards render in local prototype coordinates; the instance
		// document offset is irrelevant inside a fixed grid cell.
		nx := x + node.Geometry.X*scale - snap.OffsetX*scale
		ny := y + node.Geometry.Y*scale - snap.OffsetY*scale
		nw := node.Geometry.Width * scale
		nh := node.Geometry.Height * scale
		switch node.Kind {
		case "circle":
			s.Circle(nx, ny, nw, node.Fill)
		case "line":
			s.Line(nx, ny+nh/2, nx+nw, ny+nh/2, node.Fill, nh)
		default:
			s.Rect(nx, ny, nw, nh, node.Fill, "", 0, 5)
		}
	}
}
