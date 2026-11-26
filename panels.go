package main

func drawMissingPanel(c *Canvas, vp rect, geo SlotReport) {
	c.FillRoundRect(vp.x, vp.y, vp.w, vp.h, 14, Hex("#2A0E19"))
	drawHatch(c, vp, HexA("#FF3B5C", 58), 26)
	c.StrokeRoundRect(vp.x+1, vp.y+1, vp.w-2, vp.h-2, 13, 4, colRed)

	ix, iy := vp.x+30, vp.y+34
	drawFileIcon(c, ix, iy, 80, colRed)

	tx := ix + 78
	c.DrawText("MISSING GEOMETRY", tx, iy+6, 3, colInk, 1)
	c.DrawText("REFERENCE NOT FOUND", tx+2, iy+40, 1, Hex("#E79AA8"), 2)

	px := vp.x + 28
	valueMax := vp.x + vp.w - 28

	pathTop := vp.y + vp.h - 98
	c.DrawText("PATH:", px, pathTop, 1, colInk, 0)
	labelW := Measure("PATH:", 1, 0) + 10
	c.DrawTextFit(geo.DisplayPath, px+labelW, pathTop, 1, colBody, 0, valueMax-(px+labelW))

	srcTop := pathTop + 22
	c.DrawText("SOURCE:", px, srcTop, 1, colInk, 0)
	srcLabelW := Measure("SOURCE:", 1, 0) + 10
	c.DrawTextFit(geo.Source, px+srcLabelW, srcTop, 1, colBody, 0, valueMax-(px+srcLabelW))

	c.DrawText("READABLE PLACEHOLDER / SCENE IS NOT EMPTY", px, vp.y+vp.h-50, 1, colInk, 0)
	c.DrawText("MATERIAL AND CAMERA SOURCES ARE SHOWN BELOW", px, vp.y+vp.h-28, 1, colMuted, 0)
}
