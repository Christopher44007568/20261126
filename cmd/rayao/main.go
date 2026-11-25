package main

import (
	"flag"
	"fmt"
	"log"

	"rayao"
)

func main() {
	outDir := flag.String("out", "outputs", "directory for depth, shadow/AO, composite PNGs and manifest")
	width := flag.Int("width", 320, "frame width in pixels")
	height := flag.Int("height", 240, "frame height in pixels")
	samples := flag.Int("ao-samples", rayao.DefaultOptions().AOSamples, "deterministic hemisphere any-hit samples per primary hit")
	flag.Parse()

	scene, camera, options := rayao.NewDemoScene()
	options.Width = *width
	options.Height = *height
	options.AOSamples = *samples

	frame, err := rayao.Render(scene, camera, options)
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	manifest, err := rayao.WriteOutputs(frame, *outDir)
	if err != nil {
		log.Fatalf("write outputs: %v", err)
	}

	fmt.Printf("scene %s version %s\n", manifest.SceneName, manifest.SceneVersion)
	fmt.Printf("wrote %s/%s, %s/%s, %s/%s\n",
		*outDir, rayao.DepthPNGFilename,
		*outDir, rayao.AnyHitShadowFilename,
		*outDir, rayao.CompositePNGFilename,
	)
	fmt.Printf("primary=%d hits=%d key_rays=%d key_hits=%d ao_rays=%d ao_hits=%d\n",
		manifest.Stats.PrimaryQueries,
		manifest.Stats.PrimaryHits,
		manifest.Stats.KeyQueries,
		manifest.Stats.KeyAnyHits,
		manifest.Stats.AmbientQueries,
		manifest.Stats.AmbientAnyHits,
	)
}
