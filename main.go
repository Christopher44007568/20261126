package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"raydiff/internal"
)

func main() {
	width := flag.Int("w", 512, "image width")
	height := flag.Int("h", 384, "image height")
	outDir := flag.String("out", "out", "output directory")
	naive := flag.Bool("naive", false,
		"reuse primary LOD for secondary rays (demonstrates the noise/aliasing problem)")
	flag.Parse()

	renderer := internal.NewRenderer(*width, *height, *naive)
	buf := renderer.Render()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}

	suffix := ""
	if *naive {
		suffix = "_naive"
	}
	files := map[string][]internal.RGBA{
		"beauty" + suffix:     buf.Beauty,
		"reflection" + suffix: buf.Reflection,
		"lod" + suffix:        buf.LODMap,
	}
	for name, data := range files {
		path := filepath.Join(*outDir, name+".png")
		if err := internal.WritePNG(path, data, *width, *height); err != nil {
			panic(err)
		}
		fmt.Printf("wrote %s\n", path)
	}

	// Report LOD statistics so the differential behavior is visible in logs.
	min, max, mean := lodStats(buf.LODLevel)
	fmt.Printf("LOD range [%.3f, %.3f] mean %.3f (max mip %.0f)\n",
		min, max, mean, renderer.Scene.Ground.MaxLOD())
}

func lodStats(levels []float64) (min, max, mean float64) {
	min, max = levels[0], levels[0]
	sum := 0.0
	for _, v := range levels {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	return min, max, sum / float64(len(levels))
}
