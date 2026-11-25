package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
)

func main() {
	ev := flag.Float64("ev", 0, "scene exposure in EV stops (shared by all displays)")
	out := flag.String("out", "artifacts/displays.png", "output PNG path")
	flag.Parse()

	// One scene, one exposure decision. The exposed buffer is handed to
	// every display; displays never rebuild or re-expose it.
	scene := NewScene()
	scene.Exposure = *ev
	exposed := scene.Exposed()

	displays := DefaultDisplays()

	// Simulate a user switching between displays in an arbitrary order.
	// Re-rendering after a switch must reproduce the identical result.
	first := map[string]*Rendered{}
	results := make([]PanelResult, len(displays))
	switchOrder := []int{1, 0, 2, 0, 1, 2}
	for _, idx := range switchOrder {
		d := displays[idx]
		r := d.Render(exposed, scene.W, scene.H)
		if prior, ok := first[d.Name]; ok {
			verifyStableExposure(d.Name, prior, r)
		} else {
			first[d.Name] = r
		}
	}

	for i, d := range displays {
		results[i] = PanelResult{Display: d, Render: first[d.Name]}
		fmt.Printf("%-16s clip=%6.2f%%  pre-tone max=(%.2f %.2f %.2f)\n",
			d.Name, first[d.Name].ClipPct,
			first[d.Name].MaxRGB.R, first[d.Name].MaxRGB.G, first[d.Name].MaxRGB.B)
	}

	sheet := BuildComposite(results, *ev)
	if err := writePNG(*out, sheet); err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", *out)
}

func verifyStableExposure(name string, a, b *Rendered) {
	if len(a.Preview) != len(b.Preview) || len(a.Mask) != len(b.Mask) {
		panic("exposure reset on display switch: buffer length mismatch")
	}
	for i := range a.Preview {
		if a.Preview[i] != b.Preview[i] || a.Mask[i] != b.Mask[i] {
			panic("exposure reset on display switch: " + name)
		}
	}
}

func writePNG(path string, img *RGBImage) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img.NRGBA)
}
