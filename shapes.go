package main

import "math"

func ellipseHit(px, py, cx, cy, rx, ry float64) bool {
	if rx <= 0 || ry <= 0 {
		return false
	}
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	return dx*dx+dy*dy <= 1
}

// FillEllipse fills an axis aligned ellipse with 2x2 supersampling.
func (c *Canvas) FillEllipse(cx, cy, rx, ry int, col colorRGBA) {
	const ss = 2
	for py := cy - ry - 1; py <= cy+ry+1; py++ {
		for px := cx - rx - 1; px <= cx+rx+1; px++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					if ellipseHit(fx, fy, float64(cx), float64(cy), float64(rx), float64(ry)) {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			cc := col
			cc.A = uint8(uint32(col.A) * uint32(hit) / (ss * ss))
			c.set(px, py, cc)
		}
	}
}

// StrokeEllipse draws an elliptical outline of the given thickness.
func (c *Canvas) StrokeEllipse(cx, cy, rx, ry, thick int, col colorRGBA) {
	const ss = 2
	rxo := float64(rx) + float64(thick)/2
	ryo := float64(ry) + float64(thick)/2
	rxi := math.Max(1, float64(rx)-float64(thick)/2)
	ryi := math.Max(1, float64(ry)-float64(thick)/2)
	for py := cy - int(ryo) - 2; py <= cy+int(ryo)+2; py++ {
		for px := cx - int(rxo) - 2; px <= cx+int(rxo)+2; px++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					outer := ellipseHit(fx, fy, float64(cx), float64(cy), rxo, ryo)
					inner := ellipseHit(fx, fy, float64(cx), float64(cy), rxi, ryi)
					if outer && !inner {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			cc := col
			cc.A = uint8(uint32(col.A) * uint32(hit) / (ss * ss))
			c.set(px, py, cc)
		}
	}
}

// Ring fills the annulus between outerR and innerR.
func (c *Canvas) Ring(cx, cy, outerR, innerR int, col colorRGBA) {
	const ss = 2
	for py := cy - outerR - 1; py <= cy+outerR+1; py++ {
		for px := cx - outerR - 1; px <= cx+outerR+1; px++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					d := (fx-float64(cx))*(fx-float64(cx)) + (fy-float64(cy))*(fy-float64(cy))
					if d <= float64(outerR*outerR) && d >= float64(innerR*innerR) {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			cc := col
			cc.A = uint8(uint32(col.A) * uint32(hit) / (ss * ss))
			c.set(px, py, cc)
		}
	}
}

// FillCircle fills a solid anti-aliased circle.
func (c *Canvas) FillCircle(cx, cy, r int, col colorRGBA) {
	c.Ring(cx, cy, r, 0, col)
}

// Line draws a thickness-wide anti-aliased line between two points.
func (c *Canvas) Line(x0, y0, x1, y1, thick int, col colorRGBA) {
	half := float64(thick) / 2
	minX := minInt(x0, x1) - thick - 1
	maxX := maxInt(x0, x1) + thick + 1
	minY := minInt(y0, y1) - thick - 1
	maxY := maxInt(y0, y1) + thick + 1
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	len2 := dx*dx + dy*dy
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			fx := float64(px) + 0.5
			fy := float64(py) + 0.5
			t := 0.0
			if len2 > 0 {
				t = ((fx-float64(x0))*dx + (fy-float64(y0))*dy) / len2
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
			}
			qx := float64(x0) + t*dx
			qy := float64(y0) + t*dy
			d := math.Hypot(fx-qx, fy-qy)
			if d > half+0.75 {
				continue
			}
			a := 1.0
			if d > half-0.75 {
				a = (half + 0.75 - d) / 1.5
			}
			cc := col
			cc.A = uint8(float64(col.A) * a)
			c.set(px, py, cc)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
