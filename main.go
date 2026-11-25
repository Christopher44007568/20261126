package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type level struct {
	name    string
	quality int
}

func main() {
	outDir := "out"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	materials := []struct {
		name   string
		render func(int) *Image
	}{
		{"sphere", RenderSphere},
		{"cloth", RenderCloth},
		{"metal", RenderMetal},
	}
	levels := []level{
		{"low", 12},  // low bitrate, heavy compression
		{"mid", 50},  // medium bitrate
		{"high", 88}, // high bitrate, light compression
	}
	const size = 256
	const heatScale = 32.0 // error of 32/255 maps to white on heatmap

	fmt.Printf("%-7s %-5s %-9s %7s %8s %8s %8s %8s %8s\n",
		"material", "level", "mode", "bpp", "PSNR", "blkMean", "blkMax", "flat", "active")
	for _, m := range materials {
		src := m.render(size)
		must(src.SavePPM(filepath.Join(outDir, m.name+"_orig.ppm")))
		for _, lv := range levels {
			uni := &Codec{Quality: lv.quality, Adaptive: false}
			recU := uni.Compress(src)
			mu := ComputeMetrics(src, recU)

			ada := &Codec{Quality: lv.quality, Adaptive: true}
			recA := ada.Compress(src)
			ma := ComputeMetrics(src, recA)

			base := fmt.Sprintf("%s_%s", m.name, lv.name)
			must(recA.SavePPM(filepath.Join(outDir, base+"_render.ppm")))
			must(Heatmap(src, recA, heatScale).SavePPM(filepath.Join(outDir, base+"_heat.ppm")))
			must(BlockMap(src, recA).SavePPM(filepath.Join(outDir, base+"_blocks.ppm")))

			fmt.Printf("%-7s %-5s %-9s %7.2f %8.2f %8.2f %8.2f %8.2f %8.2f\n",
				m.name, lv.name, "uniform", uni.EstBPP, mu.PSNR,
				mu.MeanBlockErr, mu.MaxBlockErr, mu.FlatErr, mu.EdgeErr)
			fmt.Printf("%-7s %-5s %-9s %7.2f %8.2f %8.2f %8.2f %8.2f %8.2f\n",
				m.name, lv.name, "adaptive", ada.EstBPP, ma.PSNR,
				ma.MeanBlockErr, ma.MaxBlockErr, ma.FlatErr, ma.EdgeErr)
		}
	}
	fmt.Println("\nblkMean/blkMax/flat/active: per-8x8-block RMSE (0-255).")
	fmt.Println("flat = low-activity blocks, active = highlight/texture blocks.")
	fmt.Println("Adaptive keeps active-block error close to flat-block error,")
	fmt.Println("so highlights do not develop large blocky jumps.")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
