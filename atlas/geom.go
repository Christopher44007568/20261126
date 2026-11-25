package atlas

import "fmt"

// Rect is an integer pixel rectangle with half-open bounds [X, X+W).
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func R(x, y, w, h int) Rect {
	return Rect{X: x, Y: y, W: w, H: h}
}

func (r Rect) String() string {
	return fmt.Sprintf("{%d,%d %dx%d}", r.X, r.Y, r.W, r.H)
}

func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

func (r Rect) Area() int {
	if r.W <= 0 || r.H <= 0 {
		return 0
	}
	return r.W * r.H
}

func (r Rect) MaxX() int { return r.X + r.W }
func (r Rect) MaxY() int { return r.Y + r.H }

func (r Rect) ContainsPoint(x, y int) bool {
	return x >= r.X && x < r.MaxX() && y >= r.Y && y < r.MaxY()
}

func (r Rect) Contains(other Rect) bool {
	return !other.Empty() && other.X >= r.X && other.Y >= r.Y &&
		other.MaxX() <= r.MaxX() && other.MaxY() <= r.MaxY()
}

func (r Rect) Intersects(other Rect) bool {
	return r.X < other.MaxX() && r.MaxX() > other.X &&
		r.Y < other.MaxY() && r.MaxY() > other.Y
}

// Intersection returns the overlap of two rectangles.
func (r Rect) Intersection(other Rect) Rect {
	x0 := maxInt(r.X, other.X)
	y0 := maxInt(r.Y, other.Y)
	x1 := minInt(r.MaxX(), other.MaxX())
	y1 := minInt(r.MaxY(), other.MaxY())
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return R(x0, y0, x1-x0, y1-y0)
}

// Expand grows all four sides by n.
func (r Rect) Expand(n int) Rect {
	if n < 0 {
		return r.Shrink(-n)
	}
	return R(r.X-n, r.Y-n, r.W+2*n, r.H+2*n)
}

// Shrink removes n pixels from all four sides.
func (r Rect) Shrink(n int) Rect {
	if 2*n >= r.W || 2*n >= r.H {
		return Rect{}
	}
	return R(r.X+n, r.Y+n, r.W-2*n, r.H-2*n)
}

func (r Rect) Clip(bounds Rect) Rect {
	x0 := maxInt(r.X, bounds.X)
	y0 := maxInt(r.Y, bounds.Y)
	x1 := minInt(r.MaxX(), bounds.MaxX())
	y1 := minInt(r.MaxY(), bounds.MaxY())
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return R(x0, y0, x1-x0, y1-y0)
}

func (r Rect) Move(x, y int) Rect {
	return R(x, y, r.W, r.H)
}

func (r Rect) Translate(dx, dy int) Rect {
	return R(r.X+dx, r.Y+dy, r.W, r.H)
}

func (r Rect) Equal(other Rect) bool {
	return r == other
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

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// PaddingPolicy describes bilinear guard widths across a mip chain.
//
// Padding is measured in level-zero texels needed by one bilinear tap at the
// base level. Mip level l spans 2^l level-zero texels, so the same one-texel
// filter guard needs Padding<<l level-zero pixels. The packer reserves the
// largest requirement, which occurs at the deepest generated mip.
type PaddingPolicy struct {
	Padding   int `json:"padding"`
	MipLevels int `json:"mip_levels"`
}

func (p PaddingPolicy) Normalized() PaddingPolicy {
	if p.Padding < 0 {
		p.Padding = 0
	}
	if p.MipLevels < 1 {
		p.MipLevels = 1
	}
	return p
}

// BasePixelsForMip returns the required guard width on each side measured in
// original/level-zero atlas pixels. It doubles at every coarser mip.
func (p PaddingPolicy) BasePixelsForMip(level int) int {
	p = p.Normalized()
	if level < 0 {
		level = 0
	}
	if level >= p.MipLevels {
		level = p.MipLevels - 1
	}
	return p.Padding << level
}

// PackPadding is the per-side reservation the rectangle packer must use.
func (p PaddingPolicy) PackPadding() int {
	p = p.Normalized()
	return p.Padding << (p.MipLevels - 1)
}

// LevelTexels returns the guard width to paint around a rect in mip image l.
func (p PaddingPolicy) LevelTexels(level int) int {
	_ = level
	return p.Normalized().Padding
}
