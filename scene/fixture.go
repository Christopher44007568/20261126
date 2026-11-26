package scene

// Migration anchors. The old namespace is a component subtree that is re-rooted
// to a deeper production path.
const (
	OldRoot = "/Show/OldNamespace"
	NewRoot = "/Show/Shot042/Set"

	// DefaultMaterialName is the built-in surface retained when a material
	// binding cannot be resolved after migration.
	DefaultMaterialName = "DefaultSurface"
)

// DefaultMaterialColor is the retained fallback value.
var DefaultMaterialColor = Hex(0xB9BCC6)

func oldPath(suffix string) string { return OldRoot + suffix }

// Mapping returns the canonical old -> new namespace mapping.
func FixtureMapping() Mapping { return Mapping{OldRoot: OldRoot, NewRoot: NewRoot} }

// BuildDemoStage constructs the audited scene. Every concrete old path is
// reachable through the returned links so reports and tests can quote them.
func BuildDemoStage() *Stage {
	root := &Prim{ID: "root", Path: OldRoot, Type: "Xform", Shape: ShapeGroup, Tag: "OldNamespace"}

	env := &Prim{ID: "env", Path: oldPath("/Environment"), Type: "Xform", Shape: ShapeGroup, ParentID: "root", Tag: "Environment"}
	props := &Prim{ID: "props", Path: oldPath("/Props"), Type: "Xform", Shape: ShapeGroup, ParentID: "root", Tag: "Props"}
	assets := &Prim{ID: "assets", Path: oldPath("/Assets"), Type: "Scope", Shape: ShapeGroup, ParentID: "root", Tag: "Assets"}
	mats := &Prim{ID: "mats", Path: oldPath("/Materials"), Type: "Scope", Shape: ShapeGroup, ParentID: "root", Tag: "Materials"}
	cams := &Prim{ID: "cams", Path: oldPath("/Cameras"), Type: "Scope", Shape: ShapeGroup, ParentID: "root", Tag: "Cameras"}
	rig := &Prim{ID: "rig", Path: oldPath("/Rig"), Type: "Scope", Shape: ShapeGroup, ParentID: "root", Tag: "Rig"}

	table := &Prim{ID: "table", Path: oldPath("/Environment/Table"), Type: "Mesh", Shape: ShapeBox, ParentID: "env", X: 0.2, Y: 0.3, W: 2.9, H: 1.7, Color: Hex(0x9C6B43), Tag: "Table"}
	chair := &Prim{ID: "chair", Path: oldPath("/Environment/Chair"), Type: "Mesh", Shape: ShapeBox, ParentID: "env", X: 0.1, Y: 2.5, W: 1.15, H: 1.15, Color: Hex(0x6E9A78), Tag: "Chair_INST"}
	crate := &Prim{ID: "crate", Path: oldPath("/Environment/Crate"), Type: "Mesh", Shape: ShapeBox, ParentID: "env", X: 2.9, Y: 1.4, W: 1.25, H: 1.25, Color: Hex(0xAA8C5A), Tag: "Crate"}

	teapot := &Prim{ID: "teapot", Path: oldPath("/Props/Teapot"), Type: "Mesh", Shape: ShapeSphere, ParentID: "props", X: 2.35, Y: -1.75, W: 0.85, H: 0.85, Color: Hex(0x88A0B8), Tag: "Teapot"}
	lamp := &Prim{ID: "lamp", Path: oldPath("/Props/Lamp"), Type: "Mesh", Shape: ShapeSphere, ParentID: "props", X: -2.0, Y: 0.2, W: 0.75, H: 0.75, Color: DefaultMaterialColor, Tag: "Lamp"}

	chairRig := &Prim{ID: "chairRig", Path: oldPath("/Assets/ChairRig"), Type: "Component", Shape: ShapeComponent, ParentID: "assets", X: -5.0, Y: 2.0, W: 1.7, H: 1.5, Color: Hex(0x4F8A63), Tag: "ChairRig"}

	matRed := &Prim{ID: "matRed", Path: oldPath("/Materials/RedPlastic"), Type: "Material", Shape: ShapeSphere, ParentID: "mats", X: -2.0, Y: -3.8, W: 1.0, H: 1.0, Color: Hex(0xC94334), Tag: "RedPlastic"}
	matBlue := &Prim{ID: "matBlue", Path: oldPath("/Materials/BlueMetal"), Type: "Material", Shape: ShapeSphere, ParentID: "mats", X: 0.1, Y: -4.1, W: 1.0, H: 1.0, Color: Hex(0x4674C8), Tag: "BlueMetal"}

	cam := &Prim{ID: "cam", Path: oldPath("/Cameras/HeroCam"), Type: "Camera", Shape: ShapeCamera, ParentID: "cams", X: 3.8, Y: 4.35, W: 1.2, H: 0.95, Color: Hex(0x2B3340), Tag: "HeroCam"}
	focus := &Prim{ID: "focus", Path: oldPath("/Rig/FocusTarget"), Type: "Xform", Shape: ShapeTarget, ParentID: "rig", X: 1.5, Y: 2.1, W: 0.7, H: 0.7, Color: Hex(0xE6C83C), Tag: "FocusTarget"}

	s := &Stage{Name: "diner_set", Prims: []*Prim{
		root, env, props, assets, mats, cams, rig,
		table, chair, crate, teapot, lamp, chairRig,
		matRed, matBlue, cam, focus,
	}}

	s.Links = []*Link{
		{
			ID: "R1", Kind: KindReference, RelName: "references = </Assets/ChairRig>",
			OwnerID: "chair", Target: oldPath("/Assets/ChairRig"), OldTarget: oldPath("/Assets/ChairRig"),
			Expect: ExpectResolves,
		},
		{
			ID: "R2", Kind: KindReference, RelName: "references @crate_rig.usd@</Assets/MissingCrateRig>",
			OwnerID: "crate", Target: oldPath("/Assets/MissingCrateRig"), OldTarget: oldPath("/Assets/MissingCrateRig"),
			Asset: "crate_rig.usd", Expect: ExpectBroken,
		},
		{
			ID: "M1", Kind: KindMaterialBinding, RelName: "rel material:binding",
			OwnerID: "table", Target: oldPath("/Materials/RedPlastic"), OldTarget: oldPath("/Materials/RedPlastic"),
			Expect: ExpectResolves,
		},
		{
			ID: "M2", Kind: KindMaterialBinding, RelName: "rel material:binding",
			OwnerID: "teapot", Target: oldPath("/Materials/BlueMetal"), OldTarget: oldPath("/Materials/BlueMetal"),
			Expect: ExpectResolves,
		},
		{
			ID: "M3", Kind: KindMaterialBinding, RelName: "rel material:binding",
			OwnerID: "lamp", Target: oldPath("/Materials/GhostGlass"), OldTarget: oldPath("/Materials/GhostGlass"),
			Expect: ExpectDefault,
		},
		{
			ID: "C1", Kind: KindCameraTarget, RelName: "camera.target",
			OwnerID: "cam", Target: oldPath("/Rig/FocusTarget"), OldTarget: oldPath("/Rig/FocusTarget"),
			Expect: ExpectResolves,
		},
	}
	return s
}
