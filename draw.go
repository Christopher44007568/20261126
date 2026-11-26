package main

// FillCircle 使用包围矩形和圆心距离填充圆。
func (im *Image) FillCircle(r Rect, c RGB) {
	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	rx := r.W / 2
	ry := r.H / 2
	if rx == 0 || ry == 0 {
		return
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			dx := (x - cx) * ry
			dy := (y - cy) * rx
			if dx*dx+dy*dy <= rx*rx*ry*ry {
				im.Set(x, y, c)
			}
		}
	}
}

// FillTriangle 用包围盒内的重心坐标测试填充三角形。
func (im *Image) FillTriangle(p [3]Point, c RGB) {
	minX := minInt(p[0].X, p[1].X, p[2].X)
	maxX := maxInt(p[0].X, p[1].X, p[2].X)
	minY := minInt(p[0].Y, p[1].Y, p[2].Y)
	maxY := maxInt(p[0].Y, p[1].Y, p[2].Y)

	area := edge(p[0], p[1], p[2])
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			pt := Point{X: x, Y: y}
			w0 := edge(p[1], p[2], pt)
			w1 := edge(p[2], p[0], pt)
			w2 := edge(p[0], p[1], pt)
			if area < 0 {
				w0, w1, w2 = -w0, -w1, -w2
			}
			if w0 >= 0 && w1 >= 0 && w2 >= 0 {
				im.Set(x, y, c)
			}
		}
	}
}

func edge(a, b, c Point) int {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func minInt(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxInt(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func (im *Image) FillShape(s Shape, c RGB) {
	switch s.Kind {
	case ShapeRect:
		im.FillRect(s.Rect, c)
	case ShapeCircle:
		im.FillCircle(s.Rect, c)
	case ShapeTriangle:
		if len(s.Points) == 3 {
			im.FillTriangle([3]Point{s.Points[0], s.Points[1], s.Points[2]}, c)
		}
	case ShapeNone:
	}
}
