package render

import (
	"fmt"
	"io"

	"instancelab/internal/lab"
)

// WriteSharedColorError highlights the shared-map fault: every instance
// sharing the override table is filled with the error stripe pattern.
func WriteSharedColorError(w io.Writer, doc *lab.Document, fault *lab.SharedMapFault) error {
	affected := make(map[string]bool, len(fault.AffectedInstanceIDs))
	for _, id := range fault.AffectedInstanceIDs {
		affected[id] = true
	}

	s := NewSVG(w, 1280, 1560, "错误共享颜色高亮：共享覆盖表",
		fmt.Sprintf("目标 %s 的编辑穿透到 %d 个共享同一 map 的实例；品红斜纹标出全部受影响实例。",
			fault.TargetInstanceID, len(fault.UnexpectedInstanceIDs)))
	s.Header("错误共享颜色高亮 · 共享覆盖表（反例）",
		"多个实例被分配了同一个 ColorOverrides map；通过任一实例写入都会改变所有共享者的有效颜色。")

	s.Badge(48, 96, false, fmt.Sprintf("FAIL · 连带 %d 个非目标实例", len(fault.UnexpectedInstanceIDs)))
	s.Chip(330, 96, 220, fmt.Sprintf("共享 owner：%d 个", fault.OwnerCount()), ErrorPink)

	gridInstances(s, doc, func(in *lab.Instance) (fill string, stroke string, width float64, mark bool) {
		if affected[in.ID] {
			return ErrorBg, ErrorPink, 2.2, in.ID == fault.TargetInstanceID
		}
		return Panel, Line, 1, false
	})
	return s.Close()
}

// WriteWritebackError highlights the prototype-writeback fault: every
// instance inheriting the prototype default changes color together.
func WriteWritebackError(w io.Writer, doc *lab.Document, fault *lab.WritebackFault) error {
	affected := make(map[string]bool, len(fault.AffectedInstanceIDs))
	for _, id := range fault.AffectedInstanceIDs {
		affected[id] = true
	}
	s := NewSVG(w, 1280, 1560, "错误共享颜色高亮：回写原型",
		fmt.Sprintf("目标 %s 的编辑写入 Prototype.Nodes，%d 个继承默认色的实例一起变色。",
			fault.TargetInstanceID, len(fault.UnexpectedInstanceIDs)))
	s.Header("错误共享颜色高亮 · 实例编辑回写原型（反例）",
		"实例没有独立覆盖项，直接改了共享原型节点；所有未本地覆盖该节点的实例全部被连带修改。")

	s.Badge(48, 96, false, fmt.Sprintf("FAIL · 连带 %d 个非目标实例", len(fault.UnexpectedInstanceIDs)))
	s.Chip(380, 96, 260, "原型 body 填充已被改写", ErrorPink)

	gridInstances(s, doc, func(in *lab.Instance) (string, string, float64, bool) {
		if affected[in.ID] {
			return ErrorBg, ErrorPink, 2.2, in.ID == fault.TargetInstanceID
		}
		return Panel, Line, 1, false
	})
	return s.Close()
}

// WriteFixedIsolation shows the correct outcome: only the edited instance
// changes and every sibling keeps its previous fill.
func WriteFixedIsolation(w io.Writer, doc *lab.Document, report *lab.ColorIsolationReport) error {
	s := NewSVG(w, 1280, 1560, "修复后颜色隔离",
		fmt.Sprintf("目标 %s 仅在自身 map 写入覆盖；原型颜色保持 %s，其他实例零变化。",
			report.TargetInstanceID, report.PrototypeColorAfter))
	s.Header("修复后颜色隔离 · 每实例独立覆盖表（正例）",
		"SetColor 只写实例自己的 ColorOverrides map；原型只读，兄弟实例通过各自快照解析颜色。")

	s.Badge(48, 96, report.Pass, "PASS · 仅目标实例变化")
	s.Chip(300, 96, 250, fmt.Sprintf("原型颜色不变 %s", report.PrototypeColorAfter), GoodGreen)

	gridInstances(s, doc, func(in *lab.Instance) (string, string, float64, bool) {
		if in.ID == report.TargetInstanceID {
			return ErrorBg, ErrorPink, 2.2, true
		}
		return Panel, Line, 1, false
	})
	return s.Close()
}

func gridInstances(s *SVG, doc *lab.Document, style func(*lab.Instance) (fill, stroke string, width float64, mark bool)) {
	cols := 8
	cellW, cellH := 142.0, 158.0
	startX, startY := 48.0, 150.0
	for i, in := range doc.Instances() {
		col := i % cols
		row := i / cols
		x := startX + float64(col)*(cellW+8)
		y := startY + float64(row)*(cellH+10)
		fill, stroke, width, mark := style(in)
		s.Rect(x, y, cellW, cellH, fill, stroke, width, 12)
		s.Text(x+10, y+18, in.ID, 11, Ink, 800)
		if in.Hidden {
			s.Text(x+cellW/2, y+cellH/2, "HIDDEN", 12, SubInk, 800)
		} else {
			drawMiniCard(s, doc, in, x+12, y+34)
		}
		if mark {
			s.Rect(x+cellW-26, y+8, 16, 16, ErrorPink, "", 0, 4)
			s.Text(x+cellW-22.5, y+21, "T", 10, "#FFFFFF", 800)
		}
	}
}
