package main

const (
	StagePath          = "/Stage"
	AssetPath          = "/Stage/Asset"
	TreePath           = "/Stage/Asset/Tree"
	CabinPath          = "/Stage/Asset/Cabin"
	WindowBankPath     = "/Stage/Asset/Cabin/WindowBank"
	LeftWindowPath     = "/Stage/Asset/Cabin/WindowBank/Left"
	RightWindowPath    = "/Stage/Asset/Cabin/WindowBank/Right"
	RoofPath           = "/Stage/Asset/Cabin/Roof"
	InitialVariant     = VariantSummer
	PropagationVariant = VariantWinter
	OverrideVariant    = VariantNight
)

// BuildCabin 构造同一资产的固定几何和三套完整绑定表。
func BuildCabin() *Stage {
	stage := NewStage("Lakeside Cabin", "ambience", InitialVariant)
	addCabinMaterials(stage)

	mustAdd := func(parent *Node, name string, shape Shape, ids map[Variant]map[string]string) *Node {
		node, err := stage.AddNode(parent, name, shape, []string{"surface"}, ids)
		if err != nil {
			panic(err)
		}
		return node
	}

	root := mustAdd(nil, "Stage", Shape{Kind: ShapeNone}, surfaceVariants("", "", ""))
	asset := mustAdd(root, "Asset", Shape{Kind: ShapeNone}, surfaceVariants("", "", ""))
	// 结构节点不需要渲染材质；补一个不可见的 structural 材质，避免空绑定歧义。
	for _, node := range []*Node{root, asset} {
		node.Slots = nil
		node.VariantMaterials = nil
	}

	mustAdd(asset, "Sky", Shape{Kind: ShapeRect, Rect: Rect{0, 0, FrameW, FrameH}},
		surfaceVariants("sky.summer", "sky.winter", "sky.night"))

	mustAdd(asset, "Celestial", Shape{Kind: ShapeCircle, Rect: Rect{348, 28, 76, 76}},
		surfaceVariants("sun.summer", "sun.winter", "moon.night"))

	mustAdd(asset, "MountainBack", Shape{
		Kind:   ShapeTriangle,
		Points: []Point{{0, 258}, {150, 96}, {320, 258}},
	}, surfaceVariants("mountain.back.summer", "mountain.back.winter", "mountain.back.night"))

	mustAdd(asset, "MountainFront", Shape{
		Kind:   ShapeTriangle,
		Points: []Point{{170, 258}, {320, 118}, {FrameW, 258}},
	}, surfaceVariants("mountain.front.summer", "mountain.front.winter", "mountain.front.night"))

	mustAdd(asset, "Ground", Shape{Kind: ShapeRect, Rect: Rect{0, 252, FrameW, 108}},
		surfaceVariants("ground.summer", "ground.winter", "ground.night"))

	tree := mustAdd(asset, "Tree", Shape{Kind: ShapeNone, Rect: Rect{34, 88, 120, 174}},
		surfaceVariants("", "", ""))
	tree.Slots = nil
	tree.VariantMaterials = nil

	mustAdd(tree, "Trunk", Shape{Kind: ShapeRect, Rect: Rect{82, 178, 20, 82}},
		surfaceVariants("trunk.summer", "trunk.winter", "trunk.night"))
	mustAdd(tree, "CrownBack", Shape{Kind: ShapeCircle, Rect: Rect{38, 126, 76, 76}},
		surfaceVariants("crown.summer", "crown.winter", "crown.night"))
	mustAdd(tree, "CrownMain", Shape{Kind: ShapeCircle, Rect: Rect{70, 92, 88, 88}},
		surfaceVariants("crown.summer", "crown.winter", "crown.night"))
	mustAdd(tree, "CrownSide", Shape{Kind: ShapeCircle, Rect: Rect{112, 132, 62, 62}},
		surfaceVariants("crown.summer", "crown.winter", "crown.night"))

	cabin := mustAdd(asset, "Cabin", Shape{Kind: ShapeNone, Rect: Rect{132, 80, 236, 210}},
		surfaceVariants("", "", ""))
	cabin.Slots = nil
	cabin.VariantMaterials = nil

	mustAdd(cabin, "Walls", Shape{Kind: ShapeRect, Rect: Rect{150, 164, 200, 124}},
		surfaceVariants("wall.summer", "wall.winter", "wall.night"))
	mustAdd(cabin, "Roof", Shape{
		Kind:   ShapeTriangle,
		Points: []Point{{134, 172}, {250, 82}, {366, 172}},
	}, surfaceVariants("roof.summer", "roof.winter", "roof.night"))
	mustAdd(cabin, "Door", Shape{Kind: ShapeRect, Rect: Rect{229, 224, 42, 64}},
		surfaceVariants("door.summer", "door.winter", "door.night"))

	bankRect := Rect{172, 180, 156, 48}
	bank := mustAdd(cabin, "WindowBank", Shape{Kind: ShapeNone, Rect: bankRect},
		surfaceVariants("", "", ""))
	bank.Slots = nil
	bank.VariantMaterials = nil

	mustAdd(bank, "Left", Shape{Kind: ShapeRect, Rect: Rect{181, 187, 45, 34}},
		surfaceVariants("window.summer", "window.winter", "window.night"))
	mustAdd(bank, "Right", Shape{Kind: ShapeRect, Rect: Rect{274, 187, 45, 34}},
		surfaceVariants("window.summer", "window.winter", "window.night"))

	if err := stage.SelectVariant(AssetPath, InitialVariant); err != nil {
		panic(err)
	}
	return stage
}
