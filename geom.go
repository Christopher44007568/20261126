package main

// Point 是整数像素坐标。
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rect 是左上角坐标加整数宽高。
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (r Rect) Center() Point {
	return Point{X: r.X + r.W/2, Y: r.Y + r.H/2}
}

// ShapeKind 枚举零依赖渲染器支持的基础形状。
type ShapeKind string

const (
	ShapeNone     ShapeKind = "none"
	ShapeRect     ShapeKind = "rect"
	ShapeCircle   ShapeKind = "circle"
	ShapeTriangle ShapeKind = "triangle"
)

// Shape 描述节点的固定几何。variant 只替换真实材质绑定，不替换几何，
// 这样像素差异可以明确归因于绑定而不是形状变化。
type Shape struct {
	Kind   ShapeKind `json:"kind"`
	Rect   Rect      `json:"rect"`
	Points []Point   `json:"points,omitempty"`
}
