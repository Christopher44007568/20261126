package main

import (
	"image"
	"image/color"
	"math"
)

var (
	colorLime   = color.RGBA{R: 74, G: 255, B: 126, A: 255}
	colorCyan   = color.RGBA{R: 64, G: 224, B: 255, A: 255}
	colorRed    = color.RGBA{R: 255, G: 64, B: 70, A: 255}
	colorYellow = color.RGBA{R: 255, G: 220, B: 64, A: 255}
	colorWhite  = color.RGBA{R: 245, G: 248, B: 255, A: 255}
	colorDark   = color.RGBA{R: 18, G: 22, B: 30, A: 255}
)

func cloneRGBA(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)
	return dst
}

func (d Detection) gridMap() *polyMap {
	var coords, points []Vec2
	for _, p := range d.Points {
		if p.Found {
			coords = append(coords, Vec2{X: float64(p.I), Y: float64(p.J)})
			points = append(points, p.Pos)
		}
	}
	pm, _ := fitPolyMap(coords, points)
	return pm
}

func DrawDetectionGrid(img *image.RGBA, d Detection) {
	nx := d.Spec.CornersX
	for _, p := range d.Points {
		if !p.Found {
			continue
		}
		if p.I < nx-1 {
			right := d.Points[p.ID+1]
			if right.Found {
				drawLine(img, p.Pos, right.Pos, 2.2, colorLime)
			}
		}
		if p.J < d.Spec.CornersY-1 {
			down := d.Points[p.ID+nx]
			if down.Found {
				drawLine(img, p.Pos, down.Pos, 2.2, colorLime)
			}
		}
	}
	pm := d.gridMap()
	for _, p := range d.Points {
		if p.Found {
			strokeCircle(img, p.Pos, 5.2, 1.4, colorDark)
			fillCircle(img, p.Pos, 3.4, colorLime)
			continue
		}
		pred := Vec2{X: -100, Y: -100}
		if pm != nil {
			pred = pm.Map(Vec2{X: float64(p.I), Y: float64(p.J)})
		}
		if pred.X >= 0 && pred.Y >= 0 && pred.X < float64(d.Width) && pred.Y < float64(d.Height) {
			strokeCircle(img, pred, 10, 2.4, colorRed)
			drawPlus(img, pred, 5.5, 2, colorRed)
		}
	}
}

func DrawReprojectionErrors(img *image.RGBA, residuals []PointResidual) {
	for _, r := range residuals {
		drawLine(img, r.Want, r.Got, 2.4, colorRed)
		drawCross(img, r.Want, 6.5, 2, colorYellow)
		strokeCircle(img, r.Got, 5.5, 1.6, colorDark)
		fillCircle(img, r.Got, 3.2, colorCyan)
	}
}

func UndistortImage(src *image.RGBA, cam Camera) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	grayBG := color.RGBA{R: 22, G: 26, B: 34, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			xn := (float64(x) + 0.5 - cam.Cx) / cam.Fx
			yn := (float64(y) + 0.5 - cam.Cy) / cam.Fy
			xd, yd := cam.distort(xn, yn)
			sx := cam.Fx*xd + cam.Cx - 0.5
			sy := cam.Fy*yd + cam.Cy - 0.5
			if sx < 0 || sy < 0 || sx > float64(w-1) || sy > float64(h-1) {
				dst.SetRGBA(x, y, grayBG)
				continue
			}
			r, g, bl := sampleRGBA(src, sx, sy)
			dst.SetRGBA(x, y, color.RGBA{R: clampUint8(r), G: clampUint8(g), B: clampUint8(bl), A: 255})
		}
	}
	return dst
}

func sampleRGBA(img *image.RGBA, x, y float64) (float64, float64, float64) {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx, ty := x-float64(x0), y-float64(y0)
	x0 = clampInt(x0, 0, img.Bounds().Dx()-1)
	y0 = clampInt(y0, 0, img.Bounds().Dy()-1)
	x1 := clampInt(x0+1, 0, img.Bounds().Dx()-1)
	y1 := clampInt(y0+1, 0, img.Bounds().Dy()-1)
	at := func(xi, yi int) (float64, float64, float64) {
		i := img.PixOffset(xi, yi)
		return float64(img.Pix[i]), float64(img.Pix[i+1]), float64(img.Pix[i+2])
	}
	r00, g00, b00 := at(x0, y0)
	r10, g10, b10 := at(x1, y0)
	r01, g01, b01 := at(x0, y1)
	r11, g11, b11 := at(x1, y1)
	r := lerp(lerp(r00, r10, tx), lerp(r01, r11, tx), ty)
	g := lerp(lerp(g00, g10, tx), lerp(g01, g11, tx), ty)
	b := lerp(lerp(b00, b10, tx), lerp(b01, b11, tx), ty)
	return r, g, b
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func UndistortPixel(p Vec2, cam Camera) Vec2 {
	xn := (p.X - cam.Cx) / cam.Fx
	yn := (p.Y - cam.Cy) / cam.Fy
	x, y := cam.UndistortNormalized(xn, yn)
	return Vec2{X: cam.Fx*x + cam.Cx, Y: cam.Fy*y + cam.Cy}
}

func DrawCorrectedGrid(img *image.RGBA, d Detection, cam Camera, result *CalibrationResult, frame int) {
	nx := d.Spec.CornersX
	und := make(map[int]Vec2)
	for _, p := range d.Points {
		if p.Found {
			und[p.ID] = UndistortPixel(p.Pos, cam)
		}
	}
	for _, p := range d.Points {
		q, ok := und[p.ID]
		if !ok {
			continue
		}
		if p.I < nx-1 {
			if r, ok2 := und[p.ID+1]; ok2 {
				drawLine(img, q, r, 2.1, colorCyan)
			}
		}
		if p.J < d.Spec.CornersY-1 {
			if dn, ok2 := und[p.ID+nx]; ok2 {
				drawLine(img, q, dn, 2.1, colorCyan)
			}
		}
		fillCircle(img, q, 3, colorWhite)
	}
	if result != nil && frame < len(result.Poses) {
		ideal := result.Camera
		ideal.K1, ideal.K2, ideal.K3, ideal.P1, ideal.P2 = 0, 0, 0, 0, 0
		var grid [][]Vec2
		for j := 0; j < d.Spec.CornersY; j++ {
			var row []Vec2
			for i := 0; i < nx; i++ {
				obj := Vec3{X: float64(i) * d.Spec.Square, Y: float64(j) * d.Spec.Square}
				row = append(row, ideal.Project(obj, result.Poses[frame]))
			}
			grid = append(grid, row)
		}
		for j := 0; j < d.Spec.CornersY; j++ {
			for i := 0; i < nx; i++ {
				if i+1 < nx {
					drawLine(img, grid[j][i], grid[j][i+1], 1.2, colorYellow)
				}
				if j+1 < d.Spec.CornersY {
					drawLine(img, grid[j][i], grid[j+1][i], 1.2, colorYellow)
				}
			}
		}
	}
}

func RenderRectifiedGrid(src *image.RGBA, cam Camera, pose Pose, spec CheckerSpec) *image.RGBA {
	squarePx := 80.0
	w := (spec.CornersX + 1) * int(squarePx)
	h := (spec.CornersY + 1) * int(squarePx)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 22, G: 26, B: 34, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			obj := Vec3{
				X: (float64(x)/squarePx - 1) * spec.Square,
				Y: (float64(y)/squarePx - 1) * spec.Square,
			}
			srcPos := cam.Project(obj, pose)
			if srcPos.X < 0 || srcPos.Y < 0 || srcPos.X > float64(src.Bounds().Dx()-1) || srcPos.Y > float64(src.Bounds().Dy()-1) {
				dst.SetRGBA(x, y, bg)
				continue
			}
			r, g, b := sampleRGBA(src, srcPos.X, srcPos.Y)
			dst.SetRGBA(x, y, color.RGBA{R: clampUint8(r), G: clampUint8(g), B: clampUint8(b), A: 255})
		}
	}
	toXY := func(i, j int) Vec2 {
		return Vec2{X: float64(i+1) * squarePx, Y: float64(j+1) * squarePx}
	}
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			p := toXY(i, j)
			if i+1 < spec.CornersX {
				drawLine(dst, p, toXY(i+1, j), 2, colorCyan)
			}
			if j+1 < spec.CornersY {
				drawLine(dst, p, toXY(i, j+1), 2, colorCyan)
			}
			fillCircle(dst, p, 3, colorYellow)
		}
	}
	return dst
}

func dimImage(img *image.RGBA, factor float64) *image.RGBA {
	out := cloneRGBA(img)
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i] = uint8(float64(out.Pix[i]) * factor)
		out.Pix[i+1] = uint8(float64(out.Pix[i+1]) * factor)
		out.Pix[i+2] = uint8(float64(out.Pix[i+2]) * factor)
	}
	return out
}
