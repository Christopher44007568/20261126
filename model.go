package main

import "fmt"

// Layer is the source of an opinion. The correct cascade from strongest to
// weakest is Local, Instance, Inherited.
type Layer string

const (
	LayerLocal      Layer = "local"
	LayerInstance   Layer = "instance"
	LayerInherited  Layer = "inherited"
	LayerDiagnostic Layer = "diagnostic"
)

// RGB is a reproducible, dependency-free color value.
type RGB struct {
	R, G, B uint8
}

func (c RGB) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Material is the paint selected by an opinion.
type Material struct {
	ID    string
	Name  string
	Color RGB
}

// Shape is the area in which an opinion applies. It is intentionally small and
// deterministic so provenance can be reproduced from a pixel coordinate.
type Shape interface {
	Contains(x, y int) bool
	Describe() string
}

// Opinion is one layer's claim about a pixel. Inherited opinions carry a class
// depth: zero is the nearest class and larger values are farther ancestors.
type Opinion struct {
	ID        string
	Layer     Layer
	Material  Material
	Shape     Shape
	Depth     int
	Salience  int
	Source    string
	Rationale string
}

type Rect struct {
	X0, Y0, X1, Y1 int
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X0 && x < r.X1 && y >= r.Y0 && y < r.Y1
}

func (r Rect) Describe() string {
	return fmt.Sprintf("rect(%d,%d..%d,%d)", r.X0, r.Y0, r.X1, r.Y1)
}

type Ellipse struct {
	CX, CY, RX, RY int
}

func (e Ellipse) Contains(x, y int) bool {
	if e.RX <= 0 || e.RY <= 0 {
		return false
	}
	dx := int64(x-e.CX) * int64(x-e.CX) * int64(e.RY) * int64(e.RY)
	dy := int64(y-e.CY) * int64(y-e.CY) * int64(e.RX) * int64(e.RX)
	return dx+dy <= int64(e.RX)*int64(e.RX)*int64(e.RY)*int64(e.RY)
}

func (e Ellipse) Describe() string {
	return fmt.Sprintf("ellipse(%d,%d r=%d,%d)", e.CX, e.CY, e.RX, e.RY)
}

type All struct {
	W, H int
}

func (a All) Contains(x, y int) bool {
	return x >= 0 && y >= 0 && x < a.W && y < a.H
}

func (a All) Describe() string {
	return fmt.Sprintf("all(%dx%d)", a.W, a.H)
}
