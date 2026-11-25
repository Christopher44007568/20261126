package main

import "fmt"

type DemoScene struct {
	FileBase string
	Title    string
	Subtitle string
	Scene    *Scene
	Camera   *Camera
	Exposure float64
	Annotate func(c *Canvas, cam *Camera)
}

func emissiveMaterial(r, g, b, power float64) *Material {
	col := V(r, g, b)
	return &Material{Albedo: col.Mul(0.22), Emission: col.Mul(power), Shininess: 96}
}

func polishedMaterial(r, g, b, metallic, shine float64) *Material {
	return &Material{Albedo: V(r, g, b), Metallic: metallic, Shininess: shine}
}

func darkBackground(r Ray) Vec3 {
	t := clamp01(0.55 + 0.45*r.Dir.Y)
	bottom := V(0.010, 0.014, 0.030)
	top := V(0.055, 0.075, 0.130)
	return bottom.Lerp(top, t)
}

func highlightScene(fStop float64) DemoScene {
	aperture := NewAperture(6, 0.420, 1.4)
	if fStop > 0 {
		aperture = withFStop(aperture, fStop)
	}
	cam := NewCamera(V(0, 0.12, 0), V(0, 0.05, 4.0), V(0, 1, 0), 30, 1.6, 4.0, aperture)
	scene := &Scene{BackgroundColor: darkBackground}

	type ballSpec struct {
		fx, fy float64
		z      float64
		radius float64
		color  Vec3
		power  float64
		label  string
		rgb    [3]uint8
	}
	specs := []ballSpec{
		{0.13, 0.47, 1.55, 0.040, V(1.00, 0.72, 0.42), 10, "1.55 NEAR", [3]uint8{255, 196, 128}},
		{0.255, 0.59, 2.25, 0.042, V(1.00, 0.24, 0.16), 10, "2.25", [3]uint8{255, 120, 105}},
		{0.38, 0.43, 3.10, 0.032, V(1.00, 0.88, 0.20), 11, "3.10", [3]uint8{255, 230, 90}},
		{0.50, 0.52, 4.00, 0.055, V(1.00, 0.98, 0.90), 5, "4.00 FOCUS", [3]uint8{255, 255, 245}},
		{0.625, 0.61, 5.10, 0.032, V(0.30, 1.00, 0.52), 11, "5.10", [3]uint8{120, 255, 170}},
		{0.75, 0.43, 6.50, 0.036, V(0.20, 0.62, 1.00), 11, "6.50", [3]uint8{120, 190, 255}},
		{0.88, 0.55, 8.00, 0.040, V(0.82, 0.30, 1.00), 11, "8.00 FAR", [3]uint8{220, 145, 255}},
	}
	for _, spec := range specs {
		halfH := spec.z * 0.267949
		center := V(
			(spec.fx*2-1)*1.6*halfH,
			0.12+(0.5-spec.fy)*2*halfH,
			spec.z,
		)
		scene.Add(&Sphere{Center: center, Radius: spec.radius, Mat: emissiveMaterial(spec.color.X, spec.color.Y, spec.color.Z, spec.power)})
	}

	return DemoScene{
		FileBase: "highlight_spheres",
		Title:    "THIN LENS HIGHLIGHT BOKEH",
		Subtitle: fmt.Sprintf("HEX APERTURE F%g  FOCUS 4.0M  -  COC SIZE", aperture.FNumber),
		Scene:    scene, Camera: cam, Exposure: 1.0,
		Annotate: func(c *Canvas, cam *Camera) {
			for i, spec := range specs {
				x := int(spec.fx * float64(c.W))
				text := fmt.Sprintf("%.2f", spec.z)
				scale := 2
				w := TextWidth(text, scale)
				px := x - w/2
				py := c.H - 78
				c.Rect(px-8, py-6, w+16, 27, 3, 7, 16, 0.62)
				DrawText(c, text, px, py, scale, spec.rgb[0], spec.rgb[1], spec.rgb[2])
				if i == 0 || i == 3 || i == len(specs)-1 {
					category := "NEAR"
					if i == 3 {
						category = "FOCUS"
					}
					if i == len(specs)-1 {
						category = "FAR"
					}
					cw := TextWidth(category, scale)
					cy := c.H - 116
					c.Rect(x-cw/2-8, cy-5, cw+16, 25, 20, 34, 62, 0.72)
					DrawText(c, category, x-cw/2, cy, scale, 235, 242, 255)
				}
			}
		},
	}
}

func withFStop(a Aperture, fNumber float64) Aperture {
	a.FNumber = fNumber
	a.Radius = 0.5 * a.FocalLength / fNumber
	return a
}

func foregroundRodScene(fStop float64) DemoScene {
	aperture := NewAperture(6, 0.238, 1.4)
	if fStop > 0 {
		aperture = withFStop(aperture, fStop)
	}
	rodZ := 2.10
	focus := 2.34
	cam := NewCamera(V(0, 0.10, 0), V(0, 0.04, focus), V(0, 1, 0), 34, 1.6, focus, aperture)
	scene := &Scene{BackgroundColor: darkBackground}

	rodMat := polishedMaterial(0.34, 0.40, 0.52, 0.45, 130)
	rodXs := []float64{-0.78, -0.26, 0.26, 0.78}
	for _, x := range rodXs {
		scene.Add(&Rod{X: x, Z: rodZ, Radius: 0.043, YMin: -2.2, YMax: 2.2, Mat: rodMat})
	}

	type farBall struct {
		anchorX, y, z, r float64
		col              Vec3
		power            float64
	}
	balls := []farBall{
		{-0.78, 0.12, 6.10, 0.055, V(1.00, 0.20, 0.12), 12},
		{-0.52, -0.18, 6.70, 0.045, V(1.00, 0.76, 0.18), 12},
		{-0.26, 0.08, 7.15, 0.058, V(0.42, 1.00, 0.35), 12},
		{0.00, -0.16, 6.45, 0.048, V(1.00, 0.96, 0.82), 11},
		{0.26, 0.18, 7.00, 0.058, V(0.25, 0.62, 1.00), 12},
		{0.52, -0.10, 6.55, 0.045, V(0.25, 0.95, 1.00), 12},
		{0.78, 0.06, 7.45, 0.055, V(0.86, 0.34, 1.00), 12},
	}
	for _, ball := range balls {
		x := ball.anchorX * ball.z / rodZ
		scene.Add(&Sphere{
			Center: V(x, ball.y, ball.z), Radius: ball.r,
			Mat: emissiveMaterial(ball.col.X, ball.col.Y, ball.col.Z, ball.power),
		})
	}

	return DemoScene{
		FileBase: "thin_foreground_rods",
		Title:    "OCCLUSION ORDER: NEAR BEFORE FAR",
		Subtitle: fmt.Sprintf("NEAR RODS BLOCK FAR BOKEH  F%g  -  NEAREST HIT", aperture.FNumber),
		Scene:    scene, Camera: cam, Exposure: 1.0,
	}
}
