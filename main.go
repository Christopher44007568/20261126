package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory for images and stats")
	frames := flag.Int("frames", 6, "number of animation frames")
	width := flag.Int("w", 320, "image width in pixels")
	height := flag.Int("h", 240, "image height in pixels")
	aa := flag.Int("aa", 2, "supersampling grid size per pixel (1 disables AA)")
	flag.Parse()

	if *frames < 2 {
		*frames = 2
	}
	if *aa < 1 {
		*aa = 1
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	csvPath := filepath.Join(*outDir, "stats.csv")
	csv, err := os.Create(csvPath)
	if err != nil {
		fatal(err)
	}
	defer csv.Close()
	if _, err := csv.WriteString(CSVHeader()); err != nil {
		fatal(err)
	}

	fmt.Println("frame | thickness | absorb | clear | shadow rays | " +
		"opaque blocked | glass penetrated | penetrating | crossings")
	for i := 0; i < *frames; i++ {
		params := FrameParamsFor(i, *frames)
		scene := BuildScene(params)
		st := &Stats{}

		beauty, shadowMap, transMap := RenderFrame(scene, *width, *height, *aa, st)
		if err := beauty.WritePPM(framePath(*outDir, i, "beauty")); err != nil {
			fatal(err)
		}
		if err := shadowMap.WritePPM(framePath(*outDir, i, "shadow")); err != nil {
			fatal(err)
		}
		if err := transMap.WritePPM(framePath(*outDir, i, "transmission")); err != nil {
			fatal(err)
		}

		res := FrameResult{params, *st}
		if _, err := csv.WriteString(res.CSVRow()); err != nil {
			fatal(err)
		}
		fmt.Printf("%5d | %9.3f | %6.3f | %5v | %11d | %14d | %16d | %11d | %9d\n",
			i, params.SlabThick, params.AbsorbScale, params.Clear,
			st.ShadowRays, st.OpaqueOccluded, st.GlassPenetrated,
			st.PenetratingRays, st.GlassCrossings)
	}

	fmt.Printf("\nwrote %d frames x 3 images and %s\n", *frames, csvPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "glasstrace:", err)
	os.Exit(1)
}
