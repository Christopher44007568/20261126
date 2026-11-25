package rayao

// NewDemoScene builds the deterministic contact-shadow demonstration scene.
func NewDemoScene() (*Scene, Camera, Options) {
	materials := []Material{
		{Name: "warm-floor", Color: Vec3{X: 0.58, Y: 0.54, Z: 0.47}},
		{Name: "blue-wall", Color: Vec3{X: 0.42, Y: 0.53, Z: 0.64}},
		{Name: "terracotta", Color: Vec3{X: 0.68, Y: 0.20, Z: 0.12}},
		{Name: "teal", Color: Vec3{X: 0.08, Y: 0.42, Z: 0.40}},
		{Name: "ochre", Color: Vec3{X: 0.78, Y: 0.48, Z: 0.08}},
		{Name: "chalk", Color: Vec3{X: 0.72, Y: 0.70, Z: 0.64}},
	}

	var triangles []Triangle
	addQuad := func(a, b, c, d Vec3, material int) {
		triangles = append(triangles,
			NewTriangle(a, b, c, material),
			NewTriangle(a, c, d, material),
		)
	}
	addBox := func(center, half Vec3, material int) {
		triangles = append(triangles, NewBox(center, half, material)...)
	}

	// Floor and two walls create corners where radius-limited AO is visible.
	addQuad(
		Vec3{X: -4.2, Y: -0.35, Z: 3.4},
		Vec3{X: 4.2, Y: -0.35, Z: 3.4},
		Vec3{X: 4.2, Y: -0.35, Z: -2.8},
		Vec3{X: -4.2, Y: -0.35, Z: -2.8},
		0,
	)
	addQuad(
		Vec3{X: -4.2, Y: -0.35, Z: -2.8},
		Vec3{X: 4.2, Y: -0.35, Z: -2.8},
		Vec3{X: 4.2, Y: 3.3, Z: -2.8},
		Vec3{X: -4.2, Y: 3.3, Z: -2.8},
		1,
	)
	addQuad(
		Vec3{X: -3.35, Y: -0.35, Z: 3.4},
		Vec3{X: -3.35, Y: -0.35, Z: -2.8},
		Vec3{X: -3.35, Y: 3.3, Z: -2.8},
		Vec3{X: -3.35, Y: 3.3, Z: 3.4},
		1,
	)

	// Stacked and adjacent boxes provide same-surface contacts and crevices.
	addBox(Vec3{X: -0.95, Y: 0.25, Z: 0.10}, Vec3{X: 0.62, Y: 0.60, Z: 0.62}, 2)
	addBox(Vec3{X: -0.95, Y: 1.14, Z: 0.10}, Vec3{X: 0.28, Y: 0.29, Z: 0.28}, 4)
	addBox(Vec3{X: 0.82, Y: -0.01, Z: 0.38}, Vec3{X: 0.34, Y: 0.34, Z: 0.34}, 3)
	addBox(Vec3{X: 1.55, Y: 0.48, Z: -0.82}, Vec3{X: 0.20, Y: 0.83, Z: 0.20}, 5)
	addBox(Vec3{X: 0.10, Y: -0.23, Z: 1.35}, Vec3{X: 0.82, Y: 0.12, Z: 0.52}, 4)

	// A thin triangular fin gives the depth map a non-box silhouette.
	triangles = append(triangles, NewTriangle(
		Vec3{X: 1.15, Y: -0.35, Z: 0.05},
		Vec3{X: 2.25, Y: -0.35, Z: -0.18},
		Vec3{X: 1.55, Y: 1.25, Z: -0.42},
		3,
	))

	scene := NewScene("same-version-contact-study", triangles, materials)
	camera := NewCamera(
		Vec3{X: 0.28, Y: 1.18, Z: 4.75},
		Vec3{X: -0.05, Y: 0.43, Z: -0.25},
		Vec3{Y: 1},
		50,
	)
	options := DefaultOptions()
	return scene, camera, options
}
