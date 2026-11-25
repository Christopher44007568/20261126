package shutter

import "math"

// accumBuffer collects weighted exposure samples. Each sample paints opaque
// or anti-aliased color with weight w; the final pixel is the weighted mean,
// and Coverage reports how much of the interval each pixel was occupied.
type accumBuffer struct {
	width, height int
	color         []RGB
	weight        []float64
}

func newAccumBuffer(width, height int) *accumBuffer {
	n := width * height
	return &accumBuffer{
		width:  width,
		height: height,
		color:  make([]RGB, n),
		weight: make([]float64, n),
	}
}

func (b *accumBuffer) idx(x, y int) int { return y*b.width + x }

// addColor accumulates color c with sample weight w at pixel (x,y).
func (b *accumBuffer) addColor(x, y int, c RGB, w float64) {
	if x < 0 || y < 0 || x >= b.width || y >= b.height {
		return
	}
	i := b.idx(x, y)
	b.color[i].R += c.R * w
	b.color[i].G += c.G * w
	b.color[i].B += c.B * w
	b.weight[i] += w
}

// paintPolygon fills the polygon defined by pixel-space points using a
// top-left fill rule so adjacent polygons share no seams or double-covered
// pixels.
func (b *accumBuffer) paintPolygon(points []Vec2, c RGB, weight float64) {
	if len(points) < 3 {
		return
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range points {
		minX = math.Min(minX, p.X)
		maxX = math.Max(maxX, p.X)
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	x0 := maxInt(0, int(math.Floor(minX)))
	y0 := maxInt(0, int(math.Floor(minY)))
	x1 := minInt(b.width-1, int(math.Floor(maxX)))
	y1 := minInt(b.height-1, int(math.Floor(maxY)))

	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			sample := Vec2{X: float64(px) + 0.5, Y: float64(py) + 0.5}
			if pointInPolygon(sample, points) {
				b.addColor(px, py, c, weight)
			}
		}
	}
}

// edge evaluates the signed cross product of edge (a,b) against p. Positive
// means p is on the interior (left) side of the directed edge.
func edge(a, b, p Vec2) float64 {
	return (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
}

// pointInPolygon reports whether p lies inside a convex polygon. It is
// winding-agnostic because screen-space polygons are clockwise (Y down)
// while locally defined shapes tend to be counter-clockwise: a point is
// inside when every edge function carries the same sign as the polygon
// area. Zero (on-edge) is accepted so adjacent edges do not leave gaps.
func pointInPolygon(p Vec2, poly []Vec2) bool {
	area := 0.0
	for i := 0; i < len(poly); i++ {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		area += a.X*b.Y - b.X*a.Y
	}
	for i := 0; i < len(poly); i++ {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		e := edge(a, b, p)
		if area > 0 {
			if e < 0 {
				return false
			}
		} else if e > 0 {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
