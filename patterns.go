package main

func drawMaterialPattern(im *Image, node *Node, mat Material) {
	r := node.Shape.Rect
	switch mat.Shader {
	case "sky":
		im.BlendRect(Rect{X: 0, Y: 205, W: FrameW, H: 55}, mat.Accent, 55)
	case "sky_night":
		drawStars(im)
		im.BlendRect(Rect{X: 0, Y: 210, W: FrameW, H: 50}, mat.Accent, 45)

	case "sun":
		im.FillCircle(insetRect(r, 10), mat.Accent)
	case "moon":
		im.FillCircle(Rect{X: r.X + 18, Y: r.Y + 18, W: 12, H: 12}, mat.Accent)
		im.FillCircle(Rect{X: r.X + 42, Y: r.Y + 38, W: 8, H: 8}, mat.Accent)
		im.BlendCircle(r, mat.Glow, 28)

	case "mountain_snow":
		drawTriangleSnowCap(im, node.Shape.Points, mat.Accent)

	case "ground":
		for x := r.X + 12; x < r.X+r.W; x += 27 {
			y := 270 + int(deterministicHash(x, 91)%28)
			im.FillRect(Rect{X: x, Y: y, W: 2, H: 7}, mat.Accent)
			im.Set(x-2, y+2, mat.Accent)
			im.Set(x+3, y+2, mat.Accent)
		}
	case "snow_ground", "ground_night":
		for x := r.X + 18; x < r.X+r.W; x += 34 {
			y := 270 + int(deterministicHash(x, 17)%35)
			im.FillCircle(Rect{X: x, Y: y, W: 18, H: 5}, mat.Accent)
		}

	case "bark":
		for x := r.X + 4; x < r.X+r.W; x += 7 {
			im.FillRect(Rect{X: x, Y: r.Y + 3, W: 2, H: r.H - 6}, mat.Accent)
		}

	case "foliage", "foliage_night":
		drawCanopyDots(im, r, mat.Accent, 4, 19)
		drawCanopyDots(im, insetRect(r, 8), mat.Glow, 3, 41)
	case "snow_crown":
		drawCanopyDots(im, r, mat.Accent, 3, 23)
		for _, blob := range []Rect{
			{X: r.X + 10, Y: r.Y + 8, W: 24, H: 12},
			{X: r.X + 38, Y: r.Y + 2, W: 30, H: 14},
			{X: r.X + 70, Y: r.Y + 14, W: 22, H: 11},
		} {
			im.FillCircle(blob, mat.Accent)
		}

	case "planks", "planks_cold", "planks_night":
		for y := r.Y + 18; y < r.Y+r.H; y += 20 {
			im.FillRect(Rect{X: r.X + 2, Y: y, W: r.W - 4, H: 2}, mat.Accent)
			offset := 24
			if (y/20)%2 == 0 {
				offset = 58
			}
			im.FillRect(Rect{X: r.X + offset, Y: y - 18, W: 2, H: 18}, mat.Accent)
			im.FillRect(Rect{X: r.X + r.W - offset, Y: y - 18, W: 2, H: 18}, mat.Accent)
		}

	case "roof_tiles", "roof_tiles_night":
		drawTriangleRows(im, node.Shape.Points, mat.Accent)
	case "roof_snow":
		drawTriangleRows(im, node.Shape.Points, mat.Accent)
		drawTriangleSnowCap(im, node.Shape.Points, mat.Accent)

	case "door", "door_night":
		im.FrameRect(insetRect(r, 5), 2, mat.Accent)
		im.FillCircle(Rect{X: r.X + r.W - 14, Y: r.Y + r.H/2 - 3, W: 7, H: 7}, mat.Accent)
		if mat.Shader == "door_night" {
			im.BlendCircle(Rect{X: r.X + r.W - 20, Y: r.Y + r.H/2 - 10, W: 20, H: 20}, mat.Glow, 35)
		}

	case "glass", "frosted_glass":
		im.drawWindowFrame(r, mat.Accent)
		im.BlendRect(Rect{X: r.X + 6, Y: r.Y + 5, W: r.W / 3, H: 3}, mat.Glow, 190)
		if mat.Shader == "frosted_glass" {
			drawCanopyDots(im, r, mat.Glow, 2, 7)
		}
	case "lit_window":
		im.BlendRect(Rect{X: r.X - 8, Y: r.Y - 8, W: r.W + 16, H: r.H + 16}, mat.Glow, 38)
		im.drawWindowFrame(r, mat.Accent)
		im.BlendRect(Rect{X: r.X + 7, Y: r.Y + 6, W: r.W / 4, H: 4}, hexRGB("#FFF7D1"), 210)
	}
}

func (im *Image) drawWindowFrame(r Rect, c RGB) {
	im.FillRect(Rect{X: r.X + r.W/2 - 2, Y: r.Y, W: 4, H: r.H}, c)
	im.FillRect(Rect{X: r.X, Y: r.Y + r.H/2 - 2, W: r.W, H: 4}, c)
	im.FrameRect(r, 2, c)
}

func drawStars(im *Image) {
	star := hexRGB("#FFF3B0")
	for y := 18; y < 205; y += 24 {
		for x := 18; x < FrameW; x += 31 {
			h := deterministicHash(x, y)
			if h%5 != 0 {
				continue
			}
			im.Set(x, y, star)
			im.Set(x+1, y, star)
			im.Set(x-1, y, star)
			im.Set(x, y+1, star)
			im.Set(x, y-1, star)
		}
	}
}

func insetRect(r Rect, d int) Rect {
	return Rect{X: r.X + d, Y: r.Y + d, W: maxInt(1, r.W-2*d), H: maxInt(1, r.H-2*d)}
}

func insideCircle(r Rect, x, y int) bool {
	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	dx := float64(x-cx) / float64(r.W/2)
	dy := float64(y-cy) / float64(r.H/2)
	return dx*dx+dy*dy <= 1.0
}

func drawCanopyDots(im *Image, r Rect, c RGB, size, salt int) {
	step := size + 7
	for y := r.Y + 6; y < r.Y+r.H-4; y += step {
		for x := r.X + 6; x < r.X+r.W-4; x += step {
			xx := x + int(deterministicHash(x, y+salt)%7) - 3
			yy := y + int(deterministicHash(x+salt, y)%7) - 3
			if insideCircle(r, xx, yy) {
				im.FillRect(Rect{X: xx, Y: yy, W: size, H: size}, c)
			}
		}
	}
}

func pointInTriangle(p [3]Point, q Point) bool {
	a := edge(p[0], p[1], q)
	b := edge(p[1], p[2], q)
	c := edge(p[2], p[0], q)
	return (a >= 0 && b >= 0 && c >= 0) || (a <= 0 && b <= 0 && c <= 0)
}

func drawTriangleRows(im *Image, points []Point, c RGB) {
	if len(points) != 3 {
		return
	}
	tri := [3]Point{points[0], points[1], points[2]}
	minY := minInt(points[0].Y, points[1].Y, points[2].Y)
	maxY := maxInt(points[0].Y, points[1].Y, points[2].Y)
	minX := minInt(points[0].X, points[1].X, points[2].X)
	maxX := maxInt(points[0].X, points[1].X, points[2].X)
	for y := minY + 18; y < maxY; y += 18 {
		for x := minX; x <= maxX; x++ {
			if pointInTriangle(tri, Point{X: x, Y: y}) {
				im.Set(x, y, c)
				im.Set(x, y+1, c)
			}
		}
	}
}

func drawTriangleSnowCap(im *Image, points []Point, c RGB) {
	if len(points) != 3 {
		return
	}
	apex := points[1]
	left := points[0]
	right := points[2]
	if points[0].Y < apex.Y {
		apex = points[0]
		left = points[1]
		right = points[2]
	}
	const t = 0.34
	lx := apex.X + int(float64(left.X-apex.X)*t)
	ly := apex.Y + int(float64(left.Y-apex.Y)*t)
	rx := apex.X + int(float64(right.X-apex.X)*t)
	ry := apex.Y + int(float64(right.Y-apex.Y)*t)
	im.FillTriangle([3]Point{apex, {X: lx, Y: ly}, {X: rx, Y: ry}}, c)
}

func (im *Image) BlendCircle(r Rect, c RGB, alpha uint8) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if insideCircle(r, x, y) {
				im.Blend(x, y, c, alpha)
			}
		}
	}
}
