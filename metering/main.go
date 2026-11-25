package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	numFrames  = 12
	outDir     = "out"
	smoothA    = 0.25 // EMA blend per frame
	smoothStep = 0.35 // max EV slew per frame
)

var modes = []Mode{ModeFullFrame, ModeCenterWeighted, ModeHighlightProtect, ModeSpot}

func main() {
	scene := DefaultScene()
	smoothers := map[Mode]*Smoother{}
	for _, m := range modes {
		smoothers[m] = NewSmoother(smoothA, smoothStep)
	}

	fmt.Printf("%-5s", "frame")
	for _, m := range modes {
		fmt.Printf(" | %-18s", m.String()+" raw/smooth")
	}
	fmt.Println()

	jumpRaw := map[Mode]float64{}
	jumpSm := map[Mode]float64{}
	prevRaw := map[Mode]float64{}
	prevSm := map[Mode]float64{}

	for k := 0; k < numFrames; k++ {
		t := float64(k) / float64(numFrames-1)
		frame := scene.Render(t)
		spotXf, spotYf := scene.SpotPoint(t)
		spotX := spotXf * float64(scene.W)
		spotY := spotYf * float64(scene.H)

		fmt.Printf("%-5d", k)
		for _, m := range modes {
			var weights []float64
			var rawEV float64
			if m == ModeHighlightProtect {
				rawEV, weights = HighlightProtectEV(frame)
			} else {
				weights = Weights(frame, m, spotX, spotY)
				rawEV = EVForLuminance(Meter(frame, weights))
			}
			smEV := smoothers[m].Update(rawEV)

			if k > 0 {
				jumpRaw[m] = math.Max(jumpRaw[m], math.Abs(rawEV-prevRaw[m]))
				jumpSm[m] = math.Max(jumpSm[m], math.Abs(smEV-prevSm[m]))
			}
			prevRaw[m], prevSm[m] = rawEV, smEV

			photo := Expose(frame, smEV)
			base := filepath.Join(outDir, m.String(), fmt.Sprintf("frame_%02d", k))
			must(SavePNG(photo, base+"_photo.png"))
			overlay := MeterOverlay(photo, weights, int(spotX), int(spotY), m == ModeSpot)
			must(SavePNG(overlay, base+"_meter.png"))
			must(SavePNG(Histogram(photo, 256, 120), base+"_hist.png"))

			fmt.Printf(" | %+6.2f / %+6.2f  ", rawEV, smEV)
		}
		fmt.Println()
	}

	fmt.Println("\nmax frame-to-frame EV jump (raw vs smoothed):")
	for _, m := range modes {
		fmt.Printf("  %-10s raw %.2f EV -> smoothed %.2f EV\n", m, jumpRaw[m], jumpSm[m])
	}
	fmt.Println("\noutput written to", outDir+"/{fullframe,center,highlight,spot}/")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
