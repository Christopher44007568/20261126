package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
)

const (
	canvasW = 960
	canvasH = 600
)

var materialWorldQuad = []Vec2{
	{X: -1.24, Y: -0.82},
	{X: 1.24, Y: -0.82},
	{X: 1.24, Y: 0.82},
	{X: -1.24, Y: 0.82},
}

func projectQuad(matrix Mat3, quad []Vec2) ([]Vec2, bool) {
	projected, ok := matrix.TransformPoints(quad)
	if !ok {
		return nil, false
	}
	return projected, true
}

func quadCenter(quad []Vec2) Vec2 {
	var center Vec2
	for _, p := range quad {
		center = center.add(p)
	}
	return center.scale(1 / float64(len(quad)))
}

func quadBounds(quad []Vec2) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range quad {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	return
}

func pointInAxisQuad(p Vec2, quad []Vec2) bool {
	if len(quad) != 4 {
		return false
	}
	minX, minY, maxX, maxY := quadBounds(quad)
	return p.X >= minX && p.X <= maxX && p.Y >= minY && p.Y <= maxY
}

func fillConvexPolygon(dst *image.RGBA, quad []Vec2, c color.RGBA) {
	if len(quad) < 3 {
		return
	}
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, p := range quad {
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	startY := int(math.Max(0, math.Floor(minY)))
	endY := int(math.Min(float64(dst.Bounds().Dy()-1), math.Ceil(maxY)))
	for y := startY; y <= endY; y++ {
		sampleY := float64(y) + 0.5
		var xs []float64
		for i := range quad {
			a, b := quad[i], quad[(i+1)%len(quad)]
			if (a.Y <= sampleY && b.Y > sampleY) || (b.Y <= sampleY && a.Y > sampleY) {
				t := (sampleY - a.Y) / (b.Y - a.Y)
				xs = append(xs, a.X+t*(b.X-a.X))
			}
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			x0 := int(math.Max(0, math.Ceil(xs[i]-0.5)))
			x1 := int(math.Min(float64(dst.Bounds().Dx()-1), math.Floor(xs[i+1]-0.5)))
			for x := x0; x <= x1; x++ {
				blendPixel(dst, x, y, c)
			}
		}
	}
}

func drawQuadOutline(dst *image.RGBA, quad []Vec2, thickness int, c color.RGBA) {
	for i := range quad {
		drawLine(dst, quad[i], quad[(i+1)%len(quad)], thickness, c)
	}
}

func drawDashedQuad(dst *image.RGBA, quad []Vec2, thickness int, c color.RGBA) {
	for i := range quad {
		drawDashedLine(dst, quad[i], quad[(i+1)%len(quad)], thickness, 13, 8, c)
	}
}

func background(dst *image.RGBA) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		row := mixColor(
			color.RGBA{R: 7, G: 15, B: 27, A: 255},
			color.RGBA{R: 16, G: 31, B: 48, A: 255},
			t,
		)
		for x := 0; x < w; x++ {
			dst.SetRGBA(x, y, row)
		}
	}
	for x := 40; x < w; x += 48 {
		drawLine(dst, Vec2{X: float64(x), Y: 0}, Vec2{X: float64(x), Y: float64(h)}, 1, color.RGBA{R: 36, G: 57, B: 79, A: 90})
	}
	for y := 40; y < h; y += 48 {
		drawLine(dst, Vec2{X: 0, Y: float64(y)}, Vec2{X: float64(w), Y: float64(y)}, 1, color.RGBA{R: 36, G: 57, B: 79, A: 90})
	}
	drawRect(dst, 10, 10, w-20, h-20, 2, color.RGBA{R: 43, G: 65, B: 91, A: 180})
}

func rasterizeTexturedQuad(dst *image.RGBA, camera Mat3, worldQuad []Vec2, tex *image.RGBA) error {
	inv, ok := camera.Inverse()
	if !ok {
		return fmt.Errorf("camera matrix is singular")
	}
	projected, ok := projectQuad(camera, worldQuad)
	if !ok {
		return fmt.Errorf("camera projection is invalid")
	}

	shadow := make([]Vec2, len(projected))
	for i, p := range projected {
		shadow[i] = Vec2{X: p.X + 20, Y: p.Y + 28}
	}
	fillConvexPolygon(dst, shadow, color.RGBA{R: 0, G: 0, B: 0, A: 125})

	minX, minY, maxX, maxY := quadBounds(projected)
	x0 := int(math.Max(0, math.Floor(minX)))
	y0 := int(math.Max(0, math.Floor(minY)))
	x1 := int(math.Min(float64(dst.Bounds().Dx()-1), math.Ceil(maxX)))
	y1 := int(math.Min(float64(dst.Bounds().Dy()-1), math.Ceil(maxY)))
	worldMinX, worldMinY, worldMaxX, worldMaxY := quadBounds(worldQuad)

	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			world, valid := inv.Transform(Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5})
			if !valid || !pointInAxisQuad(world, worldQuad) {
				continue
			}
			u := (world.X - worldMinX) / (worldMaxX - worldMinX)
			// The camera negates world y so the screen y axis points down.
			// Flip v as well to keep the texture upright on screen.
			v := 1 - (world.Y-worldMinY)/(worldMaxY-worldMinY)
			c := sampleTexture(tex, u, v)
			edge := math.Min(math.Min(u, 1-u), math.Min(v, 1-v))
			if edge < 0.018 {
				c = mixColor(c, color.RGBA{R: 5, G: 12, B: 22, A: 255}, 0.45)
			}
			blendPixel(dst, x, y, c)
		}
	}
	return nil
}

func drawBindingHUD(dst *image.RGBA, binding RenderBinding) {
	accent := colGreen
	status := "FRESH"
	line := fmt.Sprintf("CAM V%d  TEX V%d  DESC V%d", binding.Snapshot.CameraVersion, binding.Snapshot.TextureVersion, binding.Snapshot.DescriptorVersion)
	if binding.Snapshot.Stale {
		accent = colAmber
		status = "STALE"
	}
	fillRect(dst, 24, 22, 430, 58, colPanel)
	fillRect(dst, 24, 22, 6, 58, accent)
	drawRect(dst, 24, 22, 430, 58, 1, colPanelEdge)
	fillRect(dst, 44, 35, 96, 27, accent)
	drawText(dst, 55, 42, status, 2, colInk)
	drawText(dst, 154, 38, line, 2, colWhite)

	hash := binding.Texture.Hash
	if len(hash) > 12 {
		hash = hash[:12]
	}
	drawText(dst, 760, 27, "TEXTURE HASH", 1, colMuted)
	drawText(dst, 760, 41, hash, 2, colCyan)
	drawText(dst, 760, 61, fmt.Sprintf("MATRIX %s", binding.Snapshot.CameraLabel), 1, colMuted)

	if binding.Snapshot.Stale {
		flags := strings.ToUpper(strings.ReplaceAll(binding.Snapshot.Invalidations.String(), "_", " "))
		fillRect(dst, 24, canvasH-58, 620, 34, color.RGBA{R: 52, G: 37, B: 13, A: 224})
		fillRect(dst, 24, canvasH-58, 5, 34, colAmber)
		drawText(dst, 41, canvasH-49, "PINNED SNAPSHOT  "+flags, 2, colAmber)
	}
}

func RenderFrame(binding RenderBinding) (*image.RGBA, error) {
	dst := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	background(dst)
	projected, ok := projectQuad(binding.Camera.Matrix, materialWorldQuad)
	if !ok {
		return nil, fmt.Errorf("material %s projection failed", binding.MaterialID)
	}
	if err := rasterizeTexturedQuad(dst, binding.Camera.Matrix, materialWorldQuad, binding.Pixels); err != nil {
		return nil, err
	}
	accent := colCyan
	if binding.Snapshot.Stale {
		accent = colAmber
	}
	drawQuadOutline(dst, projected, 3, accent)
	center, _ := binding.Camera.Matrix.Transform(Vec2{})
	fillCircle(dst, int(math.Round(center.X)), int(math.Round(center.Y)), 4, colWhite)
	drawCircle(dst, int(math.Round(center.X)), int(math.Round(center.Y)), 12, 2, accent)
	drawBindingHUD(dst, binding)
	return dst, nil
}
