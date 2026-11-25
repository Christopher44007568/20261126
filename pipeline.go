package main

import (
	"image"
	"image/color"
)

// paintLeaf rasterizes one world-space shape onto the supersampled canvas.
func paintLeaf(c *canvas, pts []Point, mat Material) {
	scaled := make([]Point, len(pts))
	for i, p := range pts {
		scaled[i] = Point{X: p.X * ssFactor, Y: p.Y * ssFactor}
	}
	if mat.HasFill {
		c.fillPolygon(scaled, mat.Fill)
	}
	if mat.HasStroke && mat.StrokeWidth > 0 {
		c.strokePolyline(scaled, mat.StrokeWidth*ssFactor, mat.Stroke)
	}
}

func newBackground(w, h int, bg color.NRGBA) *canvas {
	c := newCanvas(w*ssFactor, h*ssFactor)
	c.fillAll(bg)
	return c
}

// RenderScene renders the original, hierarchical scene directly by
// walking the scene graph (transforms, overrides, hidden flags and
// layer opacity are applied on the fly).
func RenderScene(s *Scene) *image.NRGBA {
	c := newBackground(s.Width, s.Height, s.Background)
	walkScene(s, func(v leafVisit) {
		if v.hidden {
			return
		}
		paintLeaf(c, v.pts, v.mat)
	})
	return downsample(c, ssFactor)
}

// RenderFlat renders a flattened scene. It needs no scene-graph
// knowledge: just an ordered list of world-space shapes.
func RenderFlat(fs *FlatScene) *image.NRGBA {
	c := newBackground(fs.Width, fs.Height, fs.Background)
	for _, sh := range fs.Shapes {
		if sh.Hidden {
			continue
		}
		paintLeaf(c, sh.Points, sh.Material)
	}
	return downsample(c, ssFactor)
}

// DiffImages produces a per-pixel absolute-difference image (amplified
// for visibility) and reports the max channel delta and the number of
// pixels that differ at all.
func DiffImages(a, b *image.NRGBA) (diff *image.NRGBA, maxDelta int, changed int) {
	w, h := a.Bounds().Dx(), a.Bounds().Dy()
	diff = image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pa := a.NRGBAAt(x, y)
			pb := b.NRGBAAt(x, y)
			dr := absDelta(pa.R, pb.R)
			dg := absDelta(pa.G, pb.G)
			db := absDelta(pa.B, pb.B)
			da := absDelta(pa.A, pb.A)
			m := dr
			for _, d := range []int{dg, db, da} {
				if d > m {
					m = d
				}
			}
			if m > maxDelta {
				maxDelta = m
			}
			if m > 0 {
				changed++
			}
			amp := func(v int) uint8 {
				v *= 8
				if v > 255 {
					v = 255
				}
				return uint8(v)
			}
			diff.SetNRGBA(x, y, color.NRGBA{
				R: amp(dr), G: amp(dg), B: amp(db), A: 255,
			})
		}
	}
	return diff, maxDelta, changed
}

func absDelta(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}
