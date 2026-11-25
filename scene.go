package main

// Scene dimensions are part of the provenance contract. Changing them changes
// every pixel index, so tests pin both dimensions.
const (
	SceneWidth  = 240
	SceneHeight = 160
)

type ClassNode struct {
	ID       string
	Parent   string
	Material string
}

type Scene struct {
	Width      int
	Height     int
	Chain      []ClassNode
	InstanceOf string
	Opinions   []Opinion
}

func demoScene() Scene {
	chain := []ClassNode{
		{ID: "vessel", Parent: "alloy"},
		{ID: "alloy", Parent: "stone", Material: "alloy.panel"},
		{ID: "stone", Material: "stone.field"},
	}
	depths := inheritanceDepths(chain, "vessel")

	stone := Material{ID: "stone.field", Name: "Stone field", Color: RGB{24, 38, 50}}
	alloy := Material{ID: "alloy.panel", Name: "Alloy panel", Color: RGB{47, 102, 144}}
	ember := Material{ID: "instance.ember", Name: "Instance ember", Color: RGB{231, 111, 81}}
	halo := Material{ID: "local.halo", Name: "Local halo", Color: RGB{239, 164, 69}}
	core := Material{ID: "local.core", Name: "Local core", Color: RGB{249, 220, 92}}
	spark := Material{ID: "local.spark", Name: "Local spark", Color: RGB{78, 205, 196}}
	slit := Material{ID: "local.slit", Name: "Local slit", Color: RGB{247, 255, 247}}

	return Scene{
		Width:      SceneWidth,
		Height:     SceneHeight,
		Chain:      chain,
		InstanceOf: "vessel",
		Opinions: []Opinion{
			{
				ID:        "inherited:stone.field",
				Layer:     LayerInherited,
				Material:  stone,
				Shape:     All{W: SceneWidth, H: SceneHeight},
				Depth:     depths["stone"],
				Source:    "class stone",
				Rationale: "root class material; weakest inherited fallback",
			},
			{
				ID:        "inherited:alloy.panel",
				Layer:     LayerInherited,
				Material:  alloy,
				Shape:     Rect{X0: 14, Y0: 14, X1: 226, Y1: 146},
				Depth:     depths["alloy"],
				Source:    "class alloy -> stone",
				Rationale: "nearest class material with coverage; outranks farther ancestors",
			},
			{
				ID:        "instance:vessel.ember",
				Layer:     LayerInstance,
				Material:  ember,
				Shape:     Ellipse{CX: 120, CY: 88, RX: 64, RY: 44},
				Source:    "instance vessel#1",
				Rationale: "instance material; must outrank every inherited material",
			},
			{
				ID:        "local:halo",
				Layer:     LayerLocal,
				Material:  halo,
				Shape:     Ellipse{CX: 92, CY: 86, RX: 34, RY: 34},
				Salience:  1,
				Source:    "local opinion A",
				Rationale: "local paint around the core",
			},
			{
				ID:        "local:core",
				Layer:     LayerLocal,
				Material:  core,
				Shape:     Ellipse{CX: 92, CY: 86, RX: 20, RY: 20},
				Salience:  2,
				Source:    "local opinion B",
				Rationale: "local core paint; strongest opinion",
			},
			{
				ID:        "local:spark",
				Layer:     LayerLocal,
				Material:  spark,
				Shape:     Rect{X0: 150, Y0: 58, X1: 170, Y1: 74},
				Source:    "local opinion C",
				Rationale: "local accent tile",
			},
			{
				ID:        "local:slit",
				Layer:     LayerLocal,
				Material:  slit,
				Shape:     Rect{X0: 134, Y0: 93, X1: 149, Y1: 125},
				Source:    "local opinion D",
				Rationale: "local highlight slit",
			},
		},
	}
}
