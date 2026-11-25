package main

import "math"

func (c *Canvas) plotThick(x, y, radius int, color Vec3, alpha float64) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius+radius {
				c.Blend(x+dx, y+dy, color, alpha)
			}
		}
	}
}

func (c *Canvas) DrawLine(x0, y0, x1, y1, radius int, color Vec3, alpha float64) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.plotThick(x0, y0, radius, color, alpha)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

var boxEdges = [12][2]int{
	{0, 1}, {1, 3}, {3, 2}, {2, 0},
	{4, 5}, {5, 7}, {7, 6}, {6, 4},
	{0, 4}, {1, 5}, {2, 6}, {3, 7},
}

func aabbCorners(b AABB) [8]Vec3 {
	return [8]Vec3{
		V(b.Min.X, b.Min.Y, b.Min.Z),
		V(b.Max.X, b.Min.Y, b.Min.Z),
		V(b.Min.X, b.Max.Y, b.Min.Z),
		V(b.Max.X, b.Max.Y, b.Min.Z),
		V(b.Min.X, b.Min.Y, b.Max.Z),
		V(b.Max.X, b.Min.Y, b.Max.Z),
		V(b.Min.X, b.Max.Y, b.Max.Z),
		V(b.Max.X, b.Max.Y, b.Max.Z),
	}
}

func DrawAABBOverlay(canvas *Canvas, cam Camera, bounds AABB, color Vec3) {
	corners := aabbCorners(bounds)
	var points [8]struct {
		x, y float64
		ok   bool
	}
	for i, p := range corners {
		x, y, _, ok := cam.Project(p)
		points[i] = struct {
			x, y float64
			ok   bool
		}{x, y, ok}
	}

	black := V(0.02, 0.02, 0.03)
	for _, edge := range boxEdges {
		a, b := points[edge[0]], points[edge[1]]
		if !a.ok || !b.ok {
			continue
		}
		x0, y0 := int(a.x+0.5), int(a.y+0.5)
		x1, y1 := int(b.x+0.5), int(b.y+0.5)
		canvas.DrawLine(x0, y0, x1, y1, 3, black, 0.85)
		canvas.DrawLine(x0, y0, x1, y1, 1, color, 1)
	}
}

func DrawLegend(canvas *Canvas, frame int) {
	amber := hexColor(0xffb000)
	magenta := hexColor(0xff2bd6)
	white := V(0.96, 0.98, 1.0)
	dark := V(0.04, 0.07, 0.12)

	panelX, panelY := 18, 18
	panelW, panelH := 430, 104
	if canvas.Width < 520 {
		panelW = canvas.Width - 36
	}
	canvas.FillRect(panelX, panelY, panelW, panelH, dark, 0.72)

	scale := 2
	x := panelX + 14
	y := panelY + 12
	canvas.DrawText(x, y, scale, "BLAS BUILT ONCE", white)
	canvas.DrawText(x+238, y, scale, "POSE", white)
	canvas.DrawText(x+292, y, scale, frameLabel(frame), amber)

	y += 25
	canvas.FillRect(x, y+2, 12, 12, amber, 1)
	canvas.DrawText(x+22, y, scale, "STALE TLAS BOX / LEFT ARM", white)

	y += 25
	canvas.FillRect(x, y+2, 12, 12, magenta, 1)
	canvas.DrawText(x+22, y, scale, "MISSED UPDATE / RIGHT LEG", white)
}

func frameLabel(frame int) string {
	switch frame {
	case 0:
		return "1/4"
	case 1:
		return "2/4"
	case 2:
		return "3/4"
	default:
		return "4/4"
	}
}
