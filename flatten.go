package main

import (
	"image/color"
	"math"
)

// ellipseSegments is the polygonization resolution for ellipses. It is
// fixed so the original scene and the flattened scene produce the exact
// same vertex data.
const ellipseSegments = 72

// outlineOf returns the local-space outline of a leaf shape.
func outlineOf(n *Node) []Point {
	switch n.Kind {
	case KindRect:
		w, h := n.W/2, n.H/2
		return []Point{{-w, -h}, {w, -h}, {w, h}, {-w, h}}
	case KindEllipse:
		pts := make([]Point, ellipseSegments)
		for i := range pts {
			t := 2 * math.Pi * float64(i) / ellipseSegments
			pts[i] = Point{X: n.W / 2 * math.Cos(t), Y: n.H / 2 * math.Sin(t)}
		}
		return pts
	case KindPolygon:
		pts := make([]Point, len(n.Points))
		copy(pts, n.Points)
		return pts
	}
	return nil
}

// mergeOverride combines two overrides; top wins field by field.
func mergeOverride(base, top *MaterialOverride) *MaterialOverride {
	if base == nil {
		return top
	}
	if top == nil {
		return base
	}
	out := &MaterialOverride{
		Fill:        base.Fill,
		Stroke:      base.Stroke,
		StrokeWidth: base.StrokeWidth,
	}
	if top.Fill != nil {
		out.Fill = top.Fill
	}
	if top.Stroke != nil {
		out.Stroke = top.Stroke
	}
	if top.StrokeWidth != nil {
		out.StrokeWidth = top.StrokeWidth
	}
	return out
}

// applyOverride resolves the effective material of a shape.
func applyOverride(m Material, ov *MaterialOverride) Material {
	if ov == nil {
		return m
	}
	if ov.Fill != nil {
		m.Fill = *ov.Fill
		m.HasFill = true
	}
	if ov.Stroke != nil {
		m.Stroke = *ov.Stroke
		m.HasStroke = true
	}
	if ov.StrokeWidth != nil {
		m.StrokeWidth = *ov.StrokeWidth
	}
	return m
}

// scaleAlpha multiplies a color's alpha by opacity in [0,1].
func scaleAlpha(c color.NRGBA, opacity float64) color.NRGBA {
	a := float64(c.A) * opacity
	if a < 0 {
		a = 0
	}
	if a > 255 {
		a = 255
	}
	c.A = uint8(a + 0.5)
	return c
}

// leafVisit is the per-leaf data produced by walking the scene graph:
// world-space outline, fully resolved material and the effective
// (baked) hidden flag. Both the direct renderer and the flattener are
// driven by the same walk, so their rasterizer inputs are identical.
type leafVisit struct {
	name   string
	hidden bool
	pts    []Point
	mat    Material
}

// walkScene traverses every layer and node in paint order, accumulating
// transforms, opacity, hidden state and material overrides, and invokes
// fn for every leaf shape (hidden leaves included).
func walkScene(s *Scene, fn func(v leafVisit)) {
	for _, l := range s.Layers {
		walkNode(l.Root, Identity(), l.opacity(), l.Hidden, nil, l.Name, fn)
	}
}

func walkNode(n *Node, world Matrix, opacity float64, hidden bool, ov *MaterialOverride, path string, fn func(v leafVisit)) {
	if n == nil {
		return
	}
	world = world.Mul(n.Transform.orIdentity())
	opacity *= n.opacity()
	hidden = hidden || n.Hidden
	name := n.Name
	if path != "" && n.Name != "" {
		name = path + "/" + n.Name
	} else if path != "" {
		name = path
	}
	if n.isGroup() {
		ov = mergeOverride(ov, n.Override)
		for _, c := range n.Children {
			walkNode(c, world, opacity, hidden, ov, name, fn)
		}
		return
	}
	local := outlineOf(n)
	if len(local) == 0 {
		return
	}
	pts := make([]Point, len(local))
	for i, p := range local {
		pts[i] = world.Apply(p)
	}
	mat := applyOverride(n.Material, ov)
	if mat.HasFill {
		mat.Fill = scaleAlpha(mat.Fill, opacity)
	}
	if mat.HasStroke {
		mat.Stroke = scaleAlpha(mat.Stroke, opacity)
		mat.StrokeWidth *= world.ApproxLen()
		if mat.StrokeWidth <= 0 {
			mat.StrokeWidth = 1
		}
	}
	fn(leafVisit{name: name, hidden: hidden, pts: pts, mat: mat})
}

// Flatten bakes the whole scene graph into a flat, self-contained
// scene: hierarchy, transforms, layer opacity and material overrides
// are resolved into world-space geometry and final materials. Hidden
// shapes are kept (with Hidden=true) so no attribute is lost.
func Flatten(s *Scene) *FlatScene {
	fs := &FlatScene{
		Width:      s.Width,
		Height:     s.Height,
		Background: s.Background,
	}
	walkScene(s, func(v leafVisit) {
		fs.Shapes = append(fs.Shapes, FlatShape{
			Name:     v.name,
			Hidden:   v.hidden,
			Kind:     KindPolygon,
			Points:   v.pts,
			Material: v.mat,
		})
	})
	return fs
}
