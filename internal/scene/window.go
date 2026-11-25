package scene

func drawWindow(p *Painter) {
	const ox0, oy0, ox1, oy1 = 1054, 52, 1686, 434
	const gx0, gy0, gx1, gy1 = 1088, 82, 1652, 404

	p.Rect(ox0-16, oy0-14, ox1+16, oy1+18, RGB{R: 0.205, G: 0.150, B: 0.096})
	p.Rect(ox0, oy0, ox1, oy1, RGB{R: 0.055, G: 0.038, B: 0.026})

	p.GradientV(gx0, gy0, gx1, 245,
		RGB{R: 2.10, G: 2.75, B: 3.80},
		RGB{R: 5.20, G: 4.65, B: 3.45},
	)
	p.GradientV(gx0, 230, gx1, gy1,
		RGB{R: 3.20, G: 3.25, B: 2.85},
		RGB{R: 1.15, G: 1.25, B: 1.18},
	)
	p.RadialGlow(1458, 132, 190, RGB{R: 1.0, G: 0.82, B: 0.55}, 1.15)
	p.Circle(1458, 132, 34, RGB{R: 8.5, G: 7.4, B: 5.2})

	drawCity(p, gx0, gy0, gx1, gy1)
	drawBranches(p)

	p.AlphaRect(gx0+18, gy0+8, gx0+72, gy1-12, RGB{R: 0.75, G: 0.88, B: 1.0}, 0.10)
	p.AlphaRect(gx0+92, gy0+8, gx0+112, gy1-12, RGB{R: 0.95, G: 0.98, B: 1.0}, 0.08)

	frame := RGB{R: 0.115, G: 0.079, B: 0.050}
	frameHi := RGB{R: 0.285, G: 0.215, B: 0.135}
	p.Rect(gx0-14, gy0-12, gx0+10, gy1+12, frame)
	p.Rect(gx1-10, gy0-12, gx1+14, gy1+12, frame)
	p.Rect(gx0-14, gy0-12, gx1+14, gy0+12, frame)
	p.Rect(gx0-14, gy1-12, gx1+14, gy1+12, frame)
	for _, mx := range []int{1276, 1466} {
		p.Rect(mx-8, gy0, mx+8, gy1, frame)
		p.Rect(mx-8, gy0, mx-5, gy1, frameHi)
	}
	p.Rect(gx0, 238, gx1, 254, frame)
	p.Rect(gx0, 238, gx1, 242, frameHi)
	p.Rect(1362, 258, 1374, 292, RGB{R: 0.34, G: 0.25, B: 0.12})
	p.Rect(1374, 266, 1408, 276, RGB{R: 0.42, G: 0.31, B: 0.15})

	p.GradientV(1030, 426, 1710, 452,
		RGB{R: 0.31, G: 0.225, B: 0.145},
		RGB{R: 0.145, G: 0.095, B: 0.055},
	)
	p.Rect(1016, 448, 1724, 462, RGB{R: 0.105, G: 0.067, B: 0.038})
	drawSillObjects(p)
}

func drawCity(p *Painter, x0, _, x1, y1 int) {
	buildings := []struct {
		x, w, top int
		c         RGB
	}{
		{1100, 110, 238, RGB{R: 0.72, G: 0.82, B: 0.92}},
		{1202, 92, 205, RGB{R: 0.92, G: 0.88, B: 0.76}},
		{1490, 118, 224, RGB{R: 0.66, G: 0.76, B: 0.86}},
		{1596, 72, 258, RGB{R: 0.86, G: 0.82, B: 0.70}},
	}
	for bi, b := range buildings {
		p.Rect(b.x, b.top, b.x+b.w, y1, b.c)
		p.Rect(b.x, b.top, b.x+b.w, b.top+8, scaleRGB(b.c, 0.72))
		for wy := b.top + 20; wy < y1-12; wy += 28 {
			for wx := b.x + 14; wx < b.x+b.w-12; wx += 25 {
				lit := hash01(wx/7, wy/9+bi*31) > 0.58
				if lit {
					p.Rect(wx, wy, wx+10, wy+14, RGB{R: 2.4, G: 2.0, B: 1.25})
				} else {
					p.Rect(wx, wy, wx+10, wy+14, RGB{R: 0.20, G: 0.30, B: 0.42})
				}
			}
		}
	}
	_ = x0
	_ = x1
}

func drawBranches(p *Painter) {
	branch := RGB{R: 0.055, G: 0.075, B: 0.045}
	p.Line(1650, 404, 1568, 286, 8, branch)
	p.Line(1568, 286, 1510, 214, 5, branch)
	p.Line(1574, 296, 1628, 218, 5, branch)
	p.Line(1510, 214, 1462, 166, 3, branch)
	p.Line(1518, 224, 1488, 150, 3, branch)
	p.Line(1628, 218, 1642, 152, 3, branch)
	for i := 0; i < 34; i++ {
		x := 1450 + int(hash01(i, 4)*210)
		y := 145 + int(hash01(i, 9)*170)
		r := 5 + int(hash01(i, 13)*8)
		p.Circle(x, y, r, RGB{R: 0.075, G: 0.16 + 0.08*hash01(i, 2), B: 0.070})
	}
}

func drawSillObjects(p *Painter) {
	p.Rect(1128, 386, 1188, 430, RGB{R: 0.34, G: 0.12, B: 0.07})
	drawPlant(p, 1158, 386, 0.42)
	p.Rect(1518, 392, 1574, 430, RGB{R: 0.70, G: 0.68, B: 0.58})
	p.Rect(1526, 374, 1566, 397, RGB{R: 0.82, G: 0.80, B: 0.70})
	p.Rect(1566, 382, 1584, 402, RGB{R: 0.70, G: 0.68, B: 0.58})
}
