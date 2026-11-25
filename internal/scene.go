package internal

// Material describes an intersected surface.
type Material struct {
	Metal     bool
	Roughness float64
	Tint      RGBA
}

// Object pairs geometry with a material.
type Object struct {
	Geo      Geometry
	Material Material
}

// Scene holds objects plus the ground and sky mip chains.
type Scene struct {
	Objects   []Object
	Ground    *Texture
	Sky       *Texture
	SunDir    Vec3
	MaxBounce int
}

func NewScene() *Scene {
	return &Scene{
		Objects: []Object{
			{
				Geo: Sphere{Center: V(-3.2, 1, 0), Radius: 1},
				Material: Material{
					Metal: true, Roughness: 0.05,
					Tint: RGBA{0.95, 0.93, 0.88, 1},
				},
			},
			{
				Geo: Sphere{Center: V(0, 1, 0), Radius: 1},
				Material: Material{
					Metal: true, Roughness: 0.25,
					Tint: RGBA{0.85, 0.87, 0.92, 1},
				},
			},
			{
				Geo: Sphere{Center: V(3.2, 1, 0), Radius: 1},
				Material: Material{
					Metal: true, Roughness: 0.5,
					Tint: RGBA{0.9, 0.86, 0.8, 1},
				},
			},
		},
		Ground:    NewCheckerTexture(),
		Sky:       NewSkyTexture(),
		SunDir:    V(0.4, 0.8, 0.45).Normalize(),
		MaxBounce: 2,
	}
}
