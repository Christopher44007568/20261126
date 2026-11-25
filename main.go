package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for rendered PPM files")
	width := flag.Int("width", 800, "PPM width in pixels")
	height := flag.Int("height", 500, "PPM height in pixels")
	samples := flag.Int("samples", 256, "lens/AA samples per pixel")
	seed := flag.Int64("seed", 20261126, "deterministic render seed")
	fStop := flag.Float64("fstop", 0, "override aperture f-number (0 uses each scene default)")
	only := flag.String("scene", "all", "highlight|rods|ruler|all")
	flag.Parse()

	if *width < 16 || *height < 16 {
		fmt.Fprintln(os.Stderr, "width and height must be at least 16")
		os.Exit(2)
	}

	all := []DemoScene{highlightScene(*fStop), foregroundRodScene(*fStop), rulerScene(*fStop)}
	for _, demo := range all {
		if *only != "all" && *only != demo.FileBase {
			continue
		}
		demo.Camera.Aspect = float64(*width) / float64(*height)
		fmt.Printf("rendering %s (%dx%d, %d samples)...\n", demo.FileBase, *width, *height, *samples)
		result := Render(demo.Scene, demo.Camera, *width, *height, RenderOptions{Samples: *samples, Seed: *seed})
		canvas := result.ToCanvas(demo.Exposure)
		if demo.Annotate != nil {
			demo.Annotate(canvas, demo.Camera)
		}
		DrawHUD(canvas, demo.Title, demo.Subtitle)
		path := filepath.Join(*outDir, demo.FileBase+".ppm")
		if err := canvas.SavePPM(path); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}
