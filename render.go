package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
)

// Shutter selects one of the three exposure/readout paths.
type Shutter int

const (
	Rolling Shutter = iota
	Segmented
	Global
)

func (s Shutter) String() string {
	switch s {
	case Rolling:
		return "rolling"
	case Segmented:
		return "segmented"
	case Global:
		return "global"
	default:
		return "unknown"
	}
}

// Image is a tightly packed RGB image, row major, three bytes per pixel.
type Image struct {
	Width, Height int
	Pix           []uint8
}

func newImage(w, h int) *Image {
	return &Image{Width: w, Height: h, Pix: make([]byte, w*h*3)}
}

func (im *Image) set(x, y int, r, g, b uint8) {
	i := (y*im.Width + x) * 3
	im.Pix[i], im.Pix[i+1], im.Pix[i+2] = r, g, b
}

// writePPM writes the binary P6 PPM format, which needs no external codec.
func (im *Image) writePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", im.Width, im.Height); err != nil {
		return err
	}
	if _, err := w.Write(im.Pix); err != nil {
		return err
	}
	return w.Flush()
}

// centerTime maps a sensor row to its mid-exposure simulation time.
func centerTime(p Params, s Shutter, frameStart float64, y int) float64 {
	switch s {
	case Rolling:
		return p.rowCenter(frameStart, float64(y)/float64(p.Height-1))
	case Segmented:
		return p.segmentCenter(frameStart, y)
	case Global:
		return p.globalCenter(frameStart)
	default:
		panic("unknown shutter")
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// backgroundShade renders world-fixed vertical stripes with faint row bands,
// so the shear of the rod against the static scene is easy to see.
func backgroundShade(worldX float64, y int) (r, g, b float64) {
	light := int(mathFloor(worldX)/40)%2 == 0
	if light {
		r, g, b = 214, 214, 214
	} else {
		r, g, b = 170, 170, 170
	}
	if y%40 == 0 {
		r, g, b = r*0.72, g*0.72, b*0.72
	}
	return r, g, b
}

func mathFloor(v float64) int {
	// math.Floor wrapper kept local to avoid an extra import in this file.
	return int(math.Floor(v))
}

// renderFrame integrates the scene over each row's exposure window.
// Temporal supersampling gives genuine motion blur instead of a frozen pose.
func renderFrame(p Params, s Shutter, frameStart float64) *Image {
	im := newImage(p.Width, p.Height)
	rodR, rodG, rodB := 200.0, 60.0, 40.0
	for y := 0; y < p.Height; y++ {
		tc := centerTime(p, s, frameStart, y)
		for x := 0; x < p.Width; x++ {
			var rr, gg, bb float64
			for k := 0; k < p.Samples; k++ {
				off := (float64(k)/float64(p.Samples-1) - 0.5) * p.Exposure
				t := tc + off
				shift := p.panPos(t)
				worldX := float64(x) + shift
				bgr, bgg, bgb := backgroundShade(worldX, y)
				rodScreenX := p.RodX - shift
				cover := clamp01(p.RodHalf - abs(float64(x)-rodScreenX) + 0.5)
				rr += bgr*(1-cover) + rodR*cover
				gg += bgg*(1-cover) + rodG*cover
				bb += bgb*(1-cover) + rodB*cover
			}
			n := float64(p.Samples)
			im.set(x, y, uint8(rr/n+0.5), uint8(gg/n+0.5), uint8(bb/n+0.5))
		}
	}
	return im
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// rodLeftEdge returns the analytic left edge of the rod on every row at the
// row's mid-exposure time. This is the data behind the edge-trajectory plot.
func rodLeftEdge(p Params, s Shutter, frameStart float64) []float64 {
	edge := make([]float64, p.Height)
	for y := 0; y < p.Height; y++ {
		t := centerTime(p, s, frameStart, y)
		edge[y] = p.RodX - p.panPos(t) - p.RodHalf
	}
	return edge
}
