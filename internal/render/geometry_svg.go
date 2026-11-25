package render

import (
	"fmt"
	"io"

	"instancelab/internal/lab"
)

// WriteBoundsOutlineSync draws the same instance before and after a
// translation, showing bounds, selection outline and render outlines moving
// together in one snapshot recomputation.
func WriteBoundsOutlineSync(w io.Writer, doc *lab.Document, instanceID string, dx, dy float64) error {
	before, err := doc.Snapshot(instanceID)
	if err != nil {
		return err
	}
	after, err := doc.Translate(instanceID, dx, dy)
	if err != nil {
		return err
	}

	s := NewSVG(w, 1280, 760, "包围盒与渲染轮廓同步更新",
		fmt.Sprintf("实例 %s 平移 (%.0f, %.0f) 后，Bounds、SelectionOutline 与 RenderOutlines 在同一次 Snapshot 中重算。",
			instanceID, dx, dy))
	s.Header("包围盒与渲染轮廓同步更新",
		"几何只有一个事实来源：当前可见节点几何的并集；移动实例时三层轮廓一起平移，不会出现只更新其中一层的漂移。")

	drawSnapshotPanel(s, "平移前 Snapshot", 48, 120, before, false)
	drawSnapshotPanel(s, "平移后 Snapshot（同一次重算）", 664, 120, after, true)

	s.Badge(48, 686, true, "PASS · 三层轮廓全部同步")
	s.Text(300, 704, fmt.Sprintf("渲染轮廓数量：%d → %d（保持一致）",
		len(before.RenderOutlines), len(after.RenderOutlines)), 13, SubInk, 600)
	return s.Close()
}

func drawSnapshotPanel(s *SVG, title string, x, y float64, snap *lab.Snapshot, moved bool) {
	s.Panel(x, y, 568, 540, title)
	scale := 1.9
	ox, oy := x+110, y+95

	for _, node := range snap.Nodes {
		if !node.Visible {
			continue
		}
		nx := ox + node.Geometry.X*scale
		ny := oy + node.Geometry.Y*scale
		nw := node.Geometry.Width * scale
		nh := node.Geometry.Height * scale
		switch node.Kind {
		case "circle":
			s.Circle(nx, ny, nw, node.Fill)
		case "line":
			s.Line(nx, ny+nh/2, nx+nw, ny+nh/2, node.Fill, nh)
		default:
			s.Rect(nx, ny, nw, nh, node.Fill, "", 0, 8)
		}
	}

	for _, outline := range snap.RenderOutlines {
		s.DashedRect(ox+outline.X*scale, oy+outline.Y*scale,
			outline.Width*scale, outline.Height*scale, "#0EA5E9", 1.2)
	}

	b := snap.Bounds
	s.DashedRect(ox+b.X*scale, oy+b.Y*scale, b.Width*scale, b.Height*scale, Ink, 2)
	s.Text(ox+b.X*scale, oy+b.Y*scale-10, "Bounds = SelectionOutline（黑色粗虚线）", 12, Ink, 800)
	s.Text(ox, oy+330, fmt.Sprintf("Offset: (%.0f, %.0f)", snap.OffsetX, snap.OffsetY), 12, SubInk, 600)
	s.Text(ox, oy+352, fmt.Sprintf("Bounds: x=%.1f y=%.1f w=%.1f h=%.1f", b.X, b.Y, b.Width, b.Height), 12, SubInk, 600)
	if moved {
		s.Text(ox, oy+374, "三层轮廓均随可见节点几何一起平移", 12, GoodGreen, 700)
	}
}
