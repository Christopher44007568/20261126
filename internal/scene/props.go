package scene

import "math"

func drawPlant(p *Painter, cx, baseY int, scale float64) {
	p.Rect(cx-int(34*scale), baseY, cx+int(34*scale), baseY+int(48*scale), RGB{R: 0.20, G: 0.095, B: 0.045})
	p.Rect(cx-int(40*scale), baseY-8, cx+int(40*scale), baseY+8, RGB{R: 0.28, G: 0.14, B: 0.065})
	stem := RGB{R: 0.055, G: 0.16, B: 0.060}
	for i := -5; i <= 5; i++ {
		endX := cx + int(float64(i)*22*scale)
		endY := baseY - int((95+math.Abs(float64(i))*13)*scale)
		p.Line(cx, baseY-4, endX, endY, maxInt(2, int(4*scale)), stem)
		leaf := RGB{R: 0.055 + 0.018*float32(hash01(i+20, 3)), G: 0.22 + 0.08*float32(hash01(i, 7)), B: 0.075}
		rx := int(float64(18+hash01(i, 11)*10) * scale)
		ry := int(float64(34+hash01(i, 13)*16) * scale)
		drawEllipse(p, endX, endY, rx, ry, float64(i)*0.18, leaf)
	}
}

func drawEllipse(p *Painter, cx, cy, rx, ry int, rotation float64, c RGB) {
	cosR, sinR := math.Cos(rotation), math.Sin(rotation)
	for y := cy - ry - 4; y <= cy+ry+4; y++ {
		for x := cx - rx - 4; x <= cx+rx+4; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			ux := dx*cosR + dy*sinR
			uy := -dx*sinR + dy*cosR
			if (ux*ux)/(float64(rx*rx))+(uy*uy)/(float64(ry*ry)) <= 1 {
				p.H.Set(x, y, c)
			}
		}
	}
}

func drawCurtains(p *Painter) {
	drawCurtain(p, 982, 42, 1088, 438)
	drawCurtain(p, 1650, 42, 1772, 438)
	p.Rect(965, 34, 1790, 58, RGB{R: 0.16, G: 0.105, B: 0.065})
	p.Rect(980, 54, 1775, 66, RGB{R: 0.31, G: 0.22, B: 0.13})
}

func drawCurtain(p *Painter, x0, y0, x1, y1 int) {
	width := x1 - x0
	for x := x0; x < x1; x++ {
		fold := 0.5 + 0.5*math.Sin(float64(x-x0)/float64(width)*math.Pi*8)
		shade := 0.075 + 0.105*fold
		c := RGB{R: float32(shade * 1.18), G: float32(shade * 0.48), B: float32(shade * 0.38)}
		p.Rect(x, y0, x+1, y1, c)
	}
	for y := y0; y < y1; y += 18 {
		p.AlphaRect(x0, y, x1, y+2, RGB{R: 0.02, G: 0.008, B: 0.006}, 0.18)
	}
	p.AlphaRect(x0+width/2-8, y0+190, x0+width/2+18, y0+238, RGB{R: 0.34, G: 0.22, B: 0.12}, 0.85)
}

func drawForegroundTable(p *Painter) {
	p.GradientV(0, 438, WorldWidth, 492,
		RGB{R: 0.105, G: 0.052, B: 0.025},
		RGB{R: 0.055, G: 0.026, B: 0.014},
	)
	p.Rect(0, 438, WorldWidth, 446, RGB{R: 0.22, G: 0.125, B: 0.060})
	p.Rect(1180, 455, 1265, 540, RGB{R: 0.040, G: 0.020, B: 0.012})
	p.Rect(1510, 455, 1595, 540, RGB{R: 0.040, G: 0.020, B: 0.012})
	p.Rect(250, 410, 330, 448, RGB{R: 0.18, G: 0.20, B: 0.20})
	p.Rect(260, 382, 320, 414, RGB{R: 0.72, G: 0.70, B: 0.62})
	p.Rect(267, 388, 313, 405, RGB{R: 0.10, G: 0.12, B: 0.14})
}
