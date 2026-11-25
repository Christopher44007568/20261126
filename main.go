package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func writePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	outDir := "out"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	out := func(name string) string { return filepath.Join(outDir, name) }
	scene := buildDemoScene()

	// 1. Render the original hierarchical scene.
	orig := RenderScene(scene)
	must(writePNG(orig, out("original.png")))

	// 2. Flatten: bake transforms, layer opacity, hidden flags and
	// material overrides into a self-contained flat scene.
	flat := Flatten(scene)
	must(SaveFlatScene(flat, out("flattened.json")))

	// 3. Reload the flattened scene from disk and render it again.
	reloaded, err := LoadFlatScene(out("flattened.json"))
	must(err)
	flatImg := RenderFlat(reloaded)
	must(writePNG(flatImg, out("flattened.png")))

	// 4. Pixel diff between the two renders.
	diff, maxDelta, changed := DiffImages(orig, flatImg)
	must(writePNG(diff, out("diff.png")))

	hidden, total := 0, len(reloaded.Shapes)
	for _, s := range reloaded.Shapes {
		if s.Hidden {
			hidden++
		}
	}
	fmt.Printf("layers: %d, flattened shapes: %d (hidden preserved: %d)\n",
		len(scene.Layers), total, hidden)
	fmt.Printf("max channel delta: %d, differing pixels: %d / %d\n",
		maxDelta, changed, scene.Width*scene.Height)
	if maxDelta != 0 {
		fmt.Println("FAIL: flattened render differs from original")
		os.Exit(1)
	}
	fmt.Println("OK: flattened scene re-renders to the identical image")
}

func must(err error) {
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
