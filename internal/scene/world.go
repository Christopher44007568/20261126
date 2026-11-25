package scene

type ROI struct {
	X0, Y0, X1, Y1 int
}

var (
	WindowROI = ROI{X0: 164, Y0: 42, X1: 446, Y1: 202}
	ShadowROI = ROI{X0: 18, Y0: 78, X1: 142, Y1: 224}
	DarkROI   = ROI{X0: 55, Y0: 70, X1: 330, Y1: 225}
)

func RenderWorld() *HDR {
	world := NewHDR(WorldWidth, WorldHeight)
	p := &Painter{H: world}

	p.GradientV(0, 0, WorldWidth, 405,
		RGB{R: 0.055, G: 0.052, B: 0.050},
		RGB{R: 0.145, G: 0.116, B: 0.088},
	)
	addWallGrain(p)

	p.GradientV(0, 390, WorldWidth, WorldHeight,
		RGB{R: 0.145, G: 0.079, B: 0.041},
		RGB{R: 0.038, G: 0.021, B: 0.015},
	)
	drawFloorBoards(p)
	p.Rect(0, 382, WorldWidth, 397, RGB{R: 0.20, G: 0.155, B: 0.105})
	p.Rect(0, 397, WorldWidth, 405, RGB{R: 0.055, G: 0.040, B: 0.030})

	drawDoorway(p)
	drawBookshelf(p)
	drawWallPictures(p)
	drawPlant(p, 850, 382, 1.15)
	drawSideCabinet(p)

	drawWindow(p)
	drawCurtains(p)
	drawForegroundTable(p)

	return world
}

func addWallGrain(p *Painter) {
	for y := 0; y < 390; y += 3 {
		for x := 0; x < WorldWidth; x += 3 {
			n := hash01(x/3, y/3)
			if n > 0.72 {
				c := p.H.At(x, y)
				delta := (n - 0.72) * 0.055
				p.Rect(x, y, x+3, y+3, RGB{c.R + delta, c.G + delta*0.9, c.B + delta*0.7})
			}
		}
	}
}

func drawFloorBoards(p *Painter) {
	for x := 0; x < WorldWidth; x += 120 {
		p.Rect(x, 392, x+3, WorldHeight, RGB{R: 0.025, G: 0.014, B: 0.010})
		offset := 30 + (x/120%3)*45
		for y := 420 + offset%60; y < WorldHeight; y += 58 {
			p.Rect(x+4, y, x+118, y+2, RGB{R: 0.055, G: 0.028, B: 0.016})
		}
	}
}

func drawDoorway(p *Painter) {
	p.Rect(118, 92, 382, 402, RGB{R: 0.19, G: 0.145, B: 0.095})
	p.GradientV(140, 112, 360, 390,
		RGB{R: 0.006, G: 0.009, B: 0.016},
		RGB{R: 0.020, G: 0.026, B: 0.040},
	)
	p.Rect(158, 145, 245, 388, RGB{R: 0.025, G: 0.018, B: 0.014})
	p.Rect(260, 170, 342, 388, RGB{R: 0.018, G: 0.020, B: 0.026})
	p.RadialGlow(315, 235, 70, RGB{R: 0.16, G: 0.20, B: 0.32}, 0.35)
	p.Circle(232, 270, 5, RGB{R: 0.42, G: 0.31, B: 0.12})
	p.Rect(105, 398, 395, 410, RGB{R: 0.12, G: 0.085, B: 0.055})
}

func drawBookshelf(p *Painter) {
	p.Rect(470, 132, 760, 398, RGB{R: 0.105, G: 0.060, B: 0.032})
	p.Rect(486, 150, 744, 382, RGB{R: 0.035, G: 0.021, B: 0.014})
	bookColors := []RGB{
		{R: 0.34, G: 0.075, B: 0.055},
		{R: 0.07, G: 0.16, B: 0.24},
		{R: 0.30, G: 0.22, B: 0.07},
		{R: 0.10, G: 0.22, B: 0.13},
		{R: 0.24, G: 0.12, B: 0.25},
		{R: 0.36, G: 0.31, B: 0.22},
	}
	shelfYs := []int{178, 232, 286, 340}
	for row, sy := range shelfYs {
		x := 496
		i := 0
		for x < 735 {
			width := 10 + int(hash01(int(x), row)*16)
			height := 34 + int(hash01(int(x*3), row+9)*18)
			c := bookColors[(i+row)%len(bookColors)]
			p.Rect(x, sy-height, x+width, sy, c)
			p.Rect(x+2, sy-height+4, x+4, sy-4, scaleRGB(c, 1.35))
			x += width + 3
			i++
		}
		p.Rect(486, sy, 744, sy+9, RGB{R: 0.16, G: 0.095, B: 0.050})
	}
	p.Rect(458, 394, 772, 407, RGB{R: 0.075, G: 0.043, B: 0.025})
}

func drawWallPictures(p *Painter) {
	frames := []ROI{
		{X0: 42, Y0: 118, X1: 92, Y1: 178},
		{X0: 805, Y0: 105, X1: 890, Y1: 165},
		{X0: 900, Y0: 145, X1: 980, Y1: 205},
	}
	for i, f := range frames {
		p.Rect(f.X0-7, f.Y0-7, f.X1+7, f.Y1+7, RGB{R: 0.24, G: 0.17, B: 0.09})
		p.GradientV(f.X0, f.Y0, f.X1, f.Y1,
			RGB{R: 0.10 + float32(i)*0.025, G: 0.15, B: 0.18},
			RGB{R: 0.28, G: 0.20, B: 0.12},
		)
		p.Circle((f.X0+f.X1)/2+8, f.Y0+22, 10, RGB{R: 0.42, G: 0.31, B: 0.16})
		p.Line(f.X0+8, f.Y1-12, f.X1-10, f.Y0+28, 4, RGB{R: 0.08, G: 0.16, B: 0.10})
	}
}

func drawSideCabinet(p *Painter) {
	p.Rect(790, 292, 1018, 398, RGB{R: 0.115, G: 0.067, B: 0.038})
	p.Rect(806, 310, 898, 382, RGB{R: 0.075, G: 0.043, B: 0.026})
	p.Rect(910, 310, 1002, 382, RGB{R: 0.075, G: 0.043, B: 0.026})
	p.Circle(889, 346, 4, RGB{R: 0.34, G: 0.25, B: 0.11})
	p.Circle(919, 346, 4, RGB{R: 0.34, G: 0.25, B: 0.11})
	p.Rect(778, 396, 1030, 410, RGB{R: 0.060, G: 0.034, B: 0.021})
}

func hash01(x, y int) float32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float32(h%10000) / 10000
}

func scaleRGB(c RGB, s float32) RGB {
	return RGB{R: c.R * s, G: c.G * s, B: c.B * s}
}
