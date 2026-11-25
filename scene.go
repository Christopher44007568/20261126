package main

// SceneObject deliberately contains no material. It can only be understood
// after ObjectID is resolved through an ObjectMapping.
type SceneObject struct {
	ID    ObjectID
	Name  string
	Shape Shape
}

type Camera struct {
	Eye    Vec3
	Target Vec3
	Up     Vec3
	FOVY   float64
}

type Scene struct {
	Objects  []SceneObject
	Catalog  *GroupCatalog
	Camera   Camera
	SkyBelow Vec3
	SkyAbove Vec3
	byID     map[ObjectID]SceneObject
}

func NewScene() *Scene {
	scene := &Scene{
		Objects: []SceneObject{
			{
				ID: ObjectFloor, Name: "floor",
				Shape: Quad{
					Corner: Vec3{-5.2, 0, -6.4}, EdgeU: Vec3{10.4, 0, 0},
					EdgeV: Vec3{0, 0, 13}, Normal: Vec3{0, 1, 0},
				},
			},
			{
				ID: ObjectStoneWall, Name: "back_stone_wall",
				Shape: Quad{
					Corner: Vec3{-5.2, 0, -6.0}, EdgeU: Vec3{10.4, 0, 0},
					EdgeV: Vec3{0, 5.2, 0}, Normal: Vec3{0, 0, 1},
				},
			},
			{
				ID: ObjectSideWall, Name: "left_stone_wall",
				Shape: Quad{
					Corner: Vec3{-5.0, 0, -6.0}, EdgeU: Vec3{0, 0, 12},
					EdgeV: Vec3{0, 5.2, 0}, Normal: Vec3{1, 0, 0},
				},
			},
			{
				ID: ObjectGlassPanel, Name: "glass_panel",
				Shape: Quad{
					Corner: Vec3{0.65, 0.35, -0.55}, EdgeU: Vec3{1.55, 0, 0},
					EdgeV: Vec3{0, 2.35, 0}, Normal: Vec3{0, 0, 1},
				},
			},
			{
				ID: ObjectAreaLight, Name: "wall_area_light",
				Shape: Quad{
					Corner: Vec3{-4.15, 2.25, -5.94}, EdgeU: Vec3{1.55, 0, 0},
					EdgeV: Vec3{0, 1.15, 0}, Normal: Vec3{0, 0, 1},
				},
			},
			{
				ID: ObjectCeramicSphere, Name: "ceramic_sphere",
				Shape: Sphere{Center: Vec3{1.55, 0.70, -2.25}, Radius: 0.70},
			},
			{
				ID: ObjectMetalSphere, Name: "metal_sphere",
				Shape: Sphere{Center: Vec3{-1.35, 0.57, -1.45}, Radius: 0.57},
			},
			{
				ID: ObjectPlinth, Name: "stone_plinth",
				Shape: Box{Min: Vec3{-0.25, 0, -3.05}, Max: Vec3{0.45, 0.46, -1.55}},
			},
		},
		Catalog: NewGroupCatalog(),
		Camera: Camera{
			Eye: Vec3{4.15, 2.9, 6.2}, Target: Vec3{-0.15, 1.20, -2.05},
			Up: Vec3{0, 1, 0}, FOVY: 49 * 3.141592653589793 / 180,
		},
		SkyBelow: Vec3{0.24, 0.22, 0.20},
		SkyAbove: Vec3{0.045, 0.065, 0.095},
	}
	scene.byID = make(map[ObjectID]SceneObject, len(scene.Objects))
	for _, object := range scene.Objects {
		scene.byID[object.ID] = object
	}
	return scene
}

func (s *Scene) Object(id ObjectID) (SceneObject, bool) {
	object, ok := s.byID[id]
	return object, ok
}
