package main

type Camera struct {
	Min    Vec2
	Max    Vec2
	Width  int
	Height int
}

func DefaultCamera() Camera {
	return Camera{
		Min:    Vec2{X: -2.65, Y: -1.58},
		Max:    Vec2{X: 1.55, Y: 1.20},
		Width:  360,
		Height: 240,
	}
}

func (c Camera) PointAt(x, y int) Vec2 {
	fx := (float64(x) + 0.5) / float64(c.Width)
	fy := (float64(y) + 0.5) / float64(c.Height)
	return Vec2{
		X: c.Min.X + fx*(c.Max.X-c.Min.X),
		Y: c.Max.Y - fy*(c.Max.Y-c.Min.Y),
	}
}

type Coverage struct {
	Width    int
	Height   int
	Hit      []bool
	Triangle []int
	Count    int
}

func NewCoverage(width, height int) *Coverage {
	return &Coverage{
		Width:    width,
		Height:   height,
		Hit:      make([]bool, width*height),
		Triangle: make([]int, width*height),
	}
}

func (c *Coverage) Set(x, y int, triangle int, hit bool) {
	index := y*c.Width + x
	c.Hit[index] = hit
	c.Triangle[index] = triangle
	if hit {
		c.Count++
	}
}

func (c *Coverage) At(x, y int) (int, bool) {
	index := y*c.Width + x
	return c.Triangle[index], c.Hit[index]
}

type HitFunc func(point Vec2) (int, bool)

func RenderMesh(mesh Mesh, camera Camera, hit HitFunc) (*Image, *Coverage) {
	img := NewImage(camera.Width, camera.Height)
	coverage := NewCoverage(camera.Width, camera.Height)
	for y := 0; y < camera.Height; y++ {
		for x := 0; x < camera.Width; x++ {
			point := camera.PointAt(x, y)
			img.Set(x, y, backgroundAt(y, camera.Height))
			triangle, ok := hit(point)
			coverage.Set(x, y, triangle, ok)
			if ok {
				img.Set(x, y, colorForTriangle(mesh, triangle))
			}
		}
	}
	return img, coverage
}

func RenderBVH(mesh Mesh, tree *BVH, camera Camera) (*Image, *Coverage) {
	return RenderMesh(mesh, camera, func(point Vec2) (int, bool) {
		return tree.IntersectPoint(mesh, point)
	})
}

func RenderBrute(mesh Mesh, camera Camera) (*Image, *Coverage) {
	return RenderMesh(mesh, camera, func(point Vec2) (int, bool) {
		return BruteIntersect(mesh, point)
	})
}

func backgroundAt(y, height int) RGB {
	t := float64(y) / float64(height-1)
	return RGB{
		R: uint8(15 + 15*t),
		G: uint8(20 + 18*t),
		B: uint8(31 + 23*t),
	}
}

type CompareStats struct {
	GroundTruthPixels  int
	ModePixels         int
	MissedPixels       int
	FalseHitPixels     int
	BackgroundFalseHit int
	WrongTriangle      int
	EmptyBackground    bool
}

func CompareCoverage(truth, mode *Coverage) CompareStats {
	stats := CompareStats{
		GroundTruthPixels: truth.Count,
		ModePixels:        mode.Count,
		EmptyBackground:   mode.Count == 0,
	}
	for y := 0; y < truth.Height; y++ {
		for x := 0; x < truth.Width; x++ {
			truthTri, truthHit := truth.At(x, y)
			modeTri, modeHit := mode.At(x, y)
			switch {
			case truthHit && !modeHit:
				stats.MissedPixels++
			case !truthHit && modeHit:
				stats.FalseHitPixels++
				stats.BackgroundFalseHit++
			case truthHit && modeHit && truthTri != modeTri:
				stats.FalseHitPixels++
				stats.WrongTriangle++
			}
		}
	}
	return stats
}

var (
	missColor        = RGB{R: 255, G: 58, B: 58}
	falseHitColor    = RGB{R: 38, G: 170, B: 255}
	legendBackground = RGB{R: 8, G: 11, B: 18}
)

func AnnotateComparison(base *Image, truth, mode *Coverage) {
	for y := 0; y < truth.Height; y++ {
		for x := 0; x < truth.Width; x++ {
			truthTri, truthHit := truth.At(x, y)
			modeTri, modeHit := mode.At(x, y)
			switch {
			case truthHit && !modeHit:
				base.Set(x, y, missColor)
			case (!truthHit && modeHit) || (truthHit && modeHit && truthTri != modeTri):
				base.Set(x, y, falseHitColor)
			}
		}
	}
	drawLegend(base)
}

func drawLegend(img *Image) {
	const x = 8
	const size = 7
	drawSwatch(img, x, 8, size, missColor)
	drawSwatch(img, x, 20, size, falseHitColor)
}

func drawSwatch(img *Image, x, y, size int, color RGB) {
	for yy := y - 1; yy <= y+size; yy++ {
		for xx := x - 1; xx <= x+size; xx++ {
			if xx == x-1 || xx == x+size || yy == y-1 || yy == y+size {
				img.Set(xx, yy, legendBackground)
			} else {
				img.Set(xx, yy, color)
			}
		}
	}
}
