// Command demo renders one motion-blurred exposure from a scene with a
// moving camera, moving objects and a static background, then writes:
//
//	artifacts/frame.png       - the single blurred frame
//	artifacts/vectors.png     - color-coded motion vector map
//	artifacts/vectors.mvf     - raw float64 motion vector field
//	artifacts/subframes.png   - 2x2 quarter-interval reference montage
//
// It uses the Go standard library only.
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"motionblur/shutter"
)

const (
	width  = 240
	height = 180
	ppu    = 36.0
)

func main() {
	outDir := filepath.Join("artifacts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail(err)
	}

	proj := shutter.OrthoProjection{PixelsPerUnit: ppu}

	req := shutter.FrameRequest{
		Width:      width,
		Height:     height,
		Projection: proj,
		OpenT:      0.0,
		CloseT:     0.4, // shutter duration; blur scales linearly with it
		Samples:    64,
		CameraAt: func(t float64) shutter.Pose {
			// Camera dollies right and slightly up during exposure.
			return shutter.Pose{Pos: shutter.Vec2{X: 1.2 * t, Y: 0.3 * t}}
		},
		Background: checkerBackground(),
		Objects: []*shutter.Object{
			translatingSquare(),
			spinningSquare(),
		},
	}

	frame := shutter.Render(req)
	fmt.Printf("exposure [%.2f,%.2f] mid=%.3f\n", frame.OpenT, frame.CloseT, frame.MidT)
	fmt.Printf("mid camera pose: %+v\n", frame.MidCamera)
	for i, pose := range frame.MidObject {
		fmt.Printf("mid object %d pose: %+v\n", i, pose)
	}
	fmt.Printf("max vector magnitude: %.2f px\n", frame.Vectors.MaxVectorLength())

	bg := color.NRGBA{R: 14, G: 16, B: 24, A: 255}

	if err := writePNG(filepath.Join(outDir, "frame.png"),
		shutter.ToNRGBA(frame.Width(), frame.Height(), frame.Color, frame.Coverage, bg)); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(outDir, "vectors.png"),
		shutter.VectorFieldNRGBA(frame.Vectors)); err != nil {
		fail(err)
	}
	if err := writeBinary(filepath.Join(outDir, "vectors.mvf"), frame.Vectors); err != nil {
		fail(err)
	}

	subs := shutter.RenderSubframes(req)
	montage := subs.Montage()
	cov := make([]float64, len(montage))
	for i := range cov {
		cov[i] = 1
	}
	if err := writePNG(filepath.Join(outDir, "subframes.png"),
		shutter.ToNRGBA(2*width, 2*height, montage, cov, bg)); err != nil {
		fail(err)
	}

	fmt.Println("wrote frame.png, vectors.png, vectors.mvf, subframes.png to", outDir)
}

// translatingSquare moves left-to-right faster than the camera so it shows
// a clear relative-motion blur.
func translatingSquare() *shutter.Object {
	red := shutter.RGB{R: 0.92, G: 0.28, B: 0.18}
	return &shutter.Object{
		Name:   "translator",
		Shapes: centeredSquare(1.1, red),
		PoseAt: func(t float64) shutter.Pose {
			return shutter.Pose{Pos: shutter.Vec2{X: -2.2 + 4.0*t, Y: -0.8}}
		},
	}
}

// spinningSquare rotates in place near the top of the frame.
func spinningSquare() *shutter.Object {
	cyan := shutter.RGB{R: 0.2, G: 0.8, B: 0.85}
	return &shutter.Object{
		Name:   "spinner",
		Shapes: centeredSquare(0.9, cyan),
		PoseAt: func(t float64) shutter.Pose {
			return shutter.Pose{Pos: shutter.Vec2{X: 1.4, Y: 1.1}, Rot: 2.5 * t}
		},
	}
}

// checkerBackground builds a world-static grid of small squares. Because it
// is rendered only through the mid-exposure camera pose, camera motion can
// never smear it.
func checkerBackground() *shutter.Background {
	light := shutter.RGB{R: 0.32, G: 0.34, B: 0.42}
	dark := shutter.RGB{R: 0.24, G: 0.26, B: 0.33}
	tile := 0.5
	half := 0.22
	var shapes []shutter.Polygon
	for ix := -6; ix <= 6; ix++ {
		for iy := -4; iy <= 4; iy++ {
			c := light
			if (ix+iy)%2 == 0 {
				c = dark
			}
			center := shutter.Vec2{X: float64(ix) * tile, Y: float64(iy) * tile}
			shapes = append(shapes, shutter.Polygon{
				Color: c,
				Vertices: []shutter.Vec2{
					{X: center.X - half, Y: center.Y - half},
					{X: center.X + half, Y: center.Y - half},
					{X: center.X + half, Y: center.Y + half},
					{X: center.X - half, Y: center.Y + half},
				},
			})
		}
	}
	return &shutter.Background{Shapes: shapes}
}

func centeredSquare(size float64, c shutter.RGB) []shutter.Polygon {
	h := size * 0.5
	return []shutter.Polygon{{
		Color: c,
		Vertices: []shutter.Vec2{
			{X: -h, Y: -h}, {X: h, Y: -h}, {X: h, Y: h}, {X: -h, Y: h},
		},
	}}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return shutter.EncodePNG(f, img)
}

func writeBinary(path string, field *shutter.VectorField) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return shutter.WriteVectorBinary(f, field)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "demo:", err)
	os.Exit(1)
}
