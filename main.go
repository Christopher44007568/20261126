package main

import (
	"fmt"
	"log"
)

// buildScene creates the demo scene. Each instance gets a distinct mask
// so the three channels have different hit sets.
func buildScene() *Scene {
	s := &Scene{Light: Vec3{4, 6, 2}, Ambient: 0.25}
	s.Add(&Instance{Center: Vec3{0, 0, -6}, Radius: 1.2, Color: Vec3{0.9, 0.3, 0.3}, Mask: MaskAll})
	s.Add(&Instance{Center: Vec3{-2.2, -0.3, -5}, Radius: 0.9, Color: Vec3{0.3, 0.9, 0.4}, Mask: MaskVisible | MaskShadow})       // no reflection
	s.Add(&Instance{Center: Vec3{2.2, -0.2, -5.5}, Radius: 1.0, Color: Vec3{0.3, 0.5, 0.95}, Mask: MaskVisible | MaskReflection}) // casts no shadow
	s.Add(&Instance{Center: Vec3{0.9, 0.9, -4.2}, Radius: 0.55, Color: Vec3{0.95, 0.85, 0.3}, Mask: MaskShadow | MaskReflection}) // camera-invisible
	s.Add(&Instance{Center: Vec3{0, -51.5, -6}, Radius: 50, Color: Vec3{0.7, 0.7, 0.72}, Mask: MaskAll})                          // floor
	return s
}

func main() {
	const w, h = 320, 200
	cam := NewCamera(w, h)
	scene := buildScene()

	ch := Render(scene, cam, w, h)
	outs := map[string]*Image{
		"out/primary.ppm":    ch.Primary,
		"out/shadow.ppm":     ch.Shadow,
		"out/reflection.ppm": ch.Reflection,
		"out/heatmap.ppm":    ch.Heatmap,
	}
	for path, im := range outs {
		if err := im.SavePPM(path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)
	}
}
