package main

func standardLights() []Light {
	return []Light{
		{Position: V(4.5, 7.0, 5.5), Color: V(1.0, 0.82, 0.62), Intensity: 1.15},
		{Position: V(-5.0, 3.5, 4.0), Color: V(0.42, 0.62, 1.0), Intensity: 0.85},
	}
}

func standardScene(name string, width, height int) Scene {
	camera := NewCamera(V(0, 2.45, 7.2), V(0, 1.35, 0), 43, width, height)
	return Scene{
		Name:       name,
		Lights:     standardLights(),
		Ambient:    V(0.30, 0.32, 0.40),
		Background: V(0.16, 0.22, 0.36),
		Camera:     camera,
	}
}

func GrazingScene(width, height int) *Scene {
	scene := standardScene("grazing", width, height)

	e0 := V(0.0, 0.12, 0.05)
	e1 := V(0.0, 2.95, -0.05)
	scene.AddTriangle("shared-edge-left", e0, e1, V(-1.85, 0.28, 0.35), V(0.95, 0.42, 0.18))
	scene.AddTriangle("shared-edge-right", e0, V(1.75, 0.22, -0.35), e1, V(0.20, 0.72, 0.95))

	knifeX := []float64{-3.05, -2.55, -2.05, 2.05, 2.55, 3.05}
	for i, x := range knifeX {
		z := -0.35 + 0.18*float64(i%3)
		h := 2.15 + 0.22*float64(i%2)
		scene.AddTriangle(
			"knife-edge",
			V(x, 0.08, z),
			V(x+0.03, h, z+0.10),
			V(x+0.10, 0.18, z-0.62),
			V(0.78, 0.28+0.08*float64(i%3), 0.32),
		)
	}

	slivers := []struct {
		x, y, z, w float64
		c          Vec3
	}{
		{-0.92, 0.18, 0.35, 0.018, V(0.82, 0.35, 1.0)},
		{-0.55, 0.12, -0.25, 0.012, V(1.0, 0.72, 0.22)},
		{0.58, 0.16, 0.18, 0.015, V(0.30, 1.0, 0.72)},
		{0.95, 0.10, -0.38, 0.010, V(1.0, 0.38, 0.42)},
	}
	for _, s := range slivers {
		scene.AddTriangle(
			"grazing-sliver",
			V(s.x, s.y, s.z),
			V(s.x+s.w, 2.65, s.z+0.12),
			V(s.x+0.16, s.y+0.10, s.z-0.48),
			s.c,
		)
	}

	scene.AddTriangle("low-prism-a", V(-1.4, 0.05, 0.8), V(-0.2, 0.05, 0.9), V(-0.75, 1.15, 0.1), V(0.95, 0.66, 0.20))
	scene.AddTriangle("low-prism-b", V(0.2, 0.05, 0.85), V(1.45, 0.05, 0.65), V(0.72, 1.25, -0.05), V(0.35, 0.88, 0.95))
	return &scene
}

func ParallelScene(width, height int) *Scene {
	scene := standardScene("parallel", width, height)
	xs := []float64{-3.2, -1.6, 0, 1.6, 3.2}
	colors := []Vec3{
		V(0.95, 0.42, 0.25), V(0.95, 0.76, 0.20), V(0.30, 0.82, 0.95),
		V(0.42, 0.55, 0.98), V(0.72, 0.42, 0.95),
	}
	for i, x := range xs {
		fb := V(x, 0.28, 3.8)
		ft := V(x, 2.75, 3.8)
		rb := V(x, 0.48, -6.5)
		rt := V(x, 2.15+0.12*float64(i%2), -6.5)
		scene.AddTriangle("parallel-strip-a", fb, ft, rt, colors[i])
		scene.AddTriangle("parallel-strip-b", fb, rt, rb, colors[i].Scale(0.72))
	}

	for i, z := range []float64{2.2, 0.4, -1.6, -3.6} {
		y := 0.18 + float64(i)*0.05
		scene.AddTriangle(
			"parallel-crossbar",
			V(-3.8, y, z),
			V(3.8, y, z),
			V(0, y+0.02, z-1.15),
			V(0.72, 0.82, 0.90),
		)
	}
	return &scene
}

func LargeTriangleScene(width, height int) *Scene {
	scene := standardScene("large_triangles", width, height)
	scene.Camera.Position = V(0, 2.2, 8.5)
	scene.Camera.Target = V(0, 1.2, -4)
	scene.Camera = NewCamera(scene.Camera.Position, scene.Camera.Target, 48, width, height)

	scene.AddTriangle("huge-left",
		V(-22, -8.5, -8.0), V(-0.8, -8.0, -8.0), V(-8.5, 12.5, -8.5),
		V(0.55, 0.16, 0.20))
	scene.AddTriangle("huge-right",
		V(0.8, -8.0, -8.2), V(22, -8.5, -8.0), V(8.5, 12.5, -8.5),
		V(0.16, 0.28, 0.58))
	scene.AddTriangle("huge-center",
		V(-34, -11.0, -15.0), V(34, -11.0, -15.0), V(0, 18.0, -16.0),
		V(0.55, 0.43, 0.16))

	for row := 0; row < 3; row++ {
		for col := 0; col < 7; col++ {
			x := -3.3 + float64(col)*1.1
			y := 0.55 + float64(row)*0.82
			z := -7.5 - 0.15*float64(row)
			s := 0.34
			albedo := V(0.25+0.1*float64(row), 0.85-0.06*float64(col%3), 0.55+0.12*float64(col%2))
			scene.AddTriangle("local-marker",
				V(x-s, y-s, z), V(x+s, y-s*0.7, z+0.05), V(x, y+s, z-0.04),
				albedo)
		}
	}
	return &scene
}

func AllScenes(width, height int) []*Scene {
	return []*Scene{
		GrazingScene(width, height),
		ParallelScene(width, height),
		LargeTriangleScene(width, height),
	}
}
