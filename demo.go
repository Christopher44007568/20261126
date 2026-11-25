package main

import (
	"image/color"
	"math"
)

// star returns a star polygon centered at the origin.
func star(points int, rOuter, rInner float64) []Point {
	pts := make([]Point, points*2)
	for i := range pts {
		r := rOuter
		if i%2 == 1 {
			r = rInner
		}
		t := -math.Pi/2 + math.Pi*float64(i)/float64(points)
		pts[i] = Point{X: r * math.Cos(t), Y: r * math.Sin(t)}
	}
	return pts
}

// buildDemoScene exercises every feature the flattener must preserve:
// nested parent/child transforms, layer opacity, hidden nodes (leaf,
// group and whole layer) and shallow + deep material overrides.
func buildDemoScene() *Scene {
	red := rgba(220, 60, 50, 255)
	blue := rgba(50, 90, 220, 255)
	gold := rgba(240, 190, 60, 255)
	white := rgba(245, 245, 245, 255)

	bgLayer := &Layer{
		Name: "background",
		Root: &Node{Children: []*Node{
			{
				Name:      "backdrop",
				Kind:      KindRect,
				W:         640,
				H:         480,
				Transform: Translate(320, 240),
				Material:  Material{Fill: rgba(24, 28, 36, 255), HasFill: true},
			},
			{
				Name:      "floor-band",
				Kind:      KindRect,
				W:         640,
				H:         90,
				Transform: Translate(320, 435),
				Material:  Material{Fill: rgba(38, 44, 56, 255), HasFill: true},
			},
		}},
	}

	wheel := &Node{
		Name:      "wheel",
		Transform: Translate(200, 200).Mul(Rotate(math.Pi / 6)),
		Children: []*Node{
			{
				Name:     "rim",
				Kind:     KindEllipse,
				W:        220,
				H:        220,
				Material: Material{Stroke: white, HasStroke: true, StrokeWidth: 6},
			},
			{
				Name:      "hub-group",
				Transform: Translate(30, -20).Mul(Rotate(math.Pi / 4)).Mul(Scale(1.2, 0.8)),
				Children: []*Node{
					{
						Name:     "hub",
						Kind:     KindEllipse,
						W:        90,
						H:        90,
						Material: Material{Fill: blue, HasFill: true, Stroke: white, HasStroke: true, StrokeWidth: 3},
					},
					{
						Name:      "spoke",
						Kind:      KindRect,
						W:         200,
						H:         14,
						Transform: Rotate(math.Pi / 5),
						Material:  Material{Fill: white, HasFill: true},
					},
				},
			},
		},
	}

	badges := &Node{
		Name:      "badges",
		Transform: Translate(430, 150),
		Override:  &MaterialOverride{Fill: rgbaPtr(gold)},
		Children: []*Node{
			{
				Name:     "badge-a",
				Kind:     KindEllipse,
				W:        120,
				H:        120,
				Material: Material{Fill: red, HasFill: true, Stroke: blue, HasStroke: true, StrokeWidth: 8},
			},
			{
				Name:      "badge-b-group",
				Transform: Translate(90, 110).Mul(Scale(0.7, 0.7)),
				Override:  &MaterialOverride{Stroke: rgbaPtr(white), StrokeWidth: f64Ptr(4)},
				Children: []*Node{
					{
						Name:     "badge-b",
						Kind:     KindPolygon,
						Points:   star(5, 70, 30),
						Material: Material{Fill: blue, HasFill: true, Stroke: red, HasStroke: true, StrokeWidth: 10},
					},
				},
			},
		},
	}

	hiddenGroup := &Node{
		Name:      "secret-group",
		Hidden:    true,
		Transform: Translate(320, 380),
		Children: []*Node{
			{Name: "secret-a", Kind: KindRect, W: 100, H: 60, Material: Material{Fill: red, HasFill: true}},
			{Name: "secret-b", Kind: KindEllipse, W: 80, H: 80, Material: Material{Fill: blue, HasFill: true}},
		},
	}
	hiddenLeaf := &Node{
		Name:      "ghost",
		Hidden:    true,
		Kind:      KindPolygon,
		Points:    star(6, 60, 25),
		Transform: Translate(560, 400),
		Material:  Material{Fill: white, HasFill: true},
	}
	hiddenLayer := &Layer{
		Name:   "hidden-layer",
		Hidden: true,
		Root: &Node{Children: []*Node{
			{Name: "never-seen", Kind: KindRect, W: 400, H: 300, Transform: Translate(320, 240), Material: Material{Fill: rgba(255, 0, 255, 255), HasFill: true}},
		}},
	}

	midLayer := &Layer{
		Name:    "mid",
		Opacity: 0.92,
		Root:    &Node{Children: []*Node{wheel, badges, hiddenGroup, hiddenLeaf}},
	}

	topLayer := &Layer{
		Name: "overlay",
		Root: &Node{Children: []*Node{
			{
				Name:      "glass-a",
				Kind:      KindEllipse,
				W:         260,
				H:         160,
				Transform: Translate(330, 300).Mul(Rotate(-math.Pi / 9)),
				Material:  Material{Fill: rgba(120, 200, 255, 110), HasFill: true},
			},
			{
				Name:      "glass-b",
				Kind:      KindEllipse,
				W:         260,
				H:         160,
				Transform: Translate(390, 320).Mul(Rotate(math.Pi / 7)),
				Material:  Material{Fill: color.NRGBA{R: 255, G: 140, B: 160, A: 110}, HasFill: true},
			},
		}},
	}

	return &Scene{
		Width:      640,
		Height:     480,
		Background: rgba(24, 28, 36, 255),
		Layers:     []*Layer{bgLayer, midLayer, hiddenLayer, topLayer},
	}
}
