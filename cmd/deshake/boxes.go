package main

import "deshake/internal/stab"

func extractBoxes(rs []*stab.FrameResult) []stab.Rect {
	out := make([]stab.Rect, len(rs))
	for i, r := range rs {
		out[i] = r.Box
	}
	return out
}

// intersectBoxes returns the largest rectangle contained in every per-frame
// valid crop. It is the fixed safe crop that never shows a hole or a wavy edge.
func intersectBoxes(boxes []stab.Rect) stab.Rect {
	if len(boxes) == 0 {
		return stab.Rect{}
	}
	g := boxes[0]
	for _, b := range boxes[1:] {
		if b.X0 > g.X0 {
			g.X0 = b.X0
		}
		if b.Y0 > g.Y0 {
			g.Y0 = b.Y0
		}
		if b.X1 < g.X1 {
			g.X1 = b.X1
		}
		if b.Y1 < g.Y1 {
			g.Y1 = b.Y1
		}
	}
	if g.X1 <= g.X0 || g.Y1 <= g.Y0 {
		return stab.Rect{}
	}
	return g
}
