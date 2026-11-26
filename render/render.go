// Package render is a dependency-free 2D software rasterizer.
//
// It renders scenes into straight-alpha NRGBA images using 4x supersampling
// and standard source-over compositing, and exposes the three images an
// override workflow needs:
//
//   - Base:  the original asset, unchanged.
//   - Over:  only the overridden meshes, on transparency, drawn with their
//     override materials (the override layer).
//   - Final: background + untouched meshes + override layer composited
//     together, so local color and opacity correctly replace the
//     original material rather than repainting a duplicate mesh.
package render

import (
	"image"
	"image/color"
	"math"

	"materialover/scene"
)

const ssaa = 4

// buffer is a premultiplied RGBA float buffer at supersampled resolution.
type buffer struct {
	w, h int
	px   []float64 // w*h entries, each 4 floats: r, g, b, a (premultiplied)
}

func newBuffer(w, h int) *buffer {
	return &buffer{w: w, h: h, px: make([]float64, w*h*4)}
}

// sourceOver blends a straight-alpha color into one pixel.
func (b *buffer) paint(x, y int, r, g, bl, a float64) {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return
	}
	i := (y*b.w + x) * 4
	sa := a
	da := b.px[i+3]
	ia := 1 - sa
	b.px[i] = r*sa + b.px[i]*ia
	b.px[i+1] = g*sa + b.px[i+1]*ia
	b.px[i+2] = bl*sa + b.px[i+2]*ia
	b.px[i+3] = sa + da*ia
}

func (b *buffer) fill(pr, pg, pbl, pa float64, inside func(x, y float64) bool) {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			// Sample at pixel centers.
			if inside(float64(x)+0.5, float64(y)+0.5) {
				b.paint(x, y, pr, pg, pbl, pa)
			}
		}
	}
}

func meshCoverage(m *scene.Mesh) func(x, y float64) bool {
	g := m.Geometry
	switch m.Kind {
	case "rect":
		return func(x, y float64) bool {
			return x >= g.X && x < g.X+g.W && y >= g.Y && y < g.Y+g.H
		}
	case "circle":
		r2 := g.Radius * g.Radius
		return func(x, y float64) bool {
			dx, dy := x-g.CX, y-g.CY
			return dx*dx+dy*dy <= r2
		}
	case "polygon":
		return func(x, y float64) bool {
			return pointInPolygon(x, y, g.Points)
		}
	}
	return func(float64, float64) bool { return false }
}

func pointInPolygon(x, y float64, pts [][2]float64) bool {
	inside := false
	j := len(pts) - 1
	for i := 0; i < len(pts); i++ {
		xi, yi := pts[i][0], pts[i][1]
		xj, yj := pts[j][0], pts[j][1]
		if (yi > y) != (yj > y) {
			cross := (xj-xi)*(y-yi)/(yj-yi) + xi
			if x < cross {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// downscale averages the supersampled buffer into a straight-alpha NRGBA image.
func (b *buffer) downscale(dstW, dstH int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			var r, g, bl, a float64
			for sy := 0; sy < ssaa; sy++ {
				for sx := 0; sx < ssaa; sx++ {
					i := ((y*ssaa+sy)*b.w + (x*ssaa + sx)) * 4
					r += b.px[i]
					g += b.px[i+1]
					bl += b.px[i+2]
					a += b.px[i+3]
				}
			}
			n := float64(ssaa * ssaa)
			r, g, bl, a = r/n, g/n, bl/n, a/n
			out := img.NRGBAAt(x, y)
			out.R = toByte(unpremul(r, a))
			out.G = toByte(unpremul(g, a))
			out.B = toByte(unpremul(bl, a))
			out.A = toByte(a)
			img.SetNRGBA(x, y, out)
		}
	}
	return img
}

func unpremul(c, a float64) float64 {
	if a <= 0 {
		return 0
	}
	v := c / a
	if v > 1 {
		v = 1
	}
	return v
}

func toByte(v float64) uint8 {
	v = math.Round(v * 255)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func renderMeshes(w, h int, meshes []*scene.Mesh) *image.NRGBA {
	buf := newBuffer(w*ssaa, h*ssaa)
	scale := float64(ssaa)
	for _, m := range meshes {
		mm := scaleMesh(m, scale)
		c := mm.Material.Color
		buf.fill(c.R, c.G, c.B, mm.Material.Opacity, meshCoverage(mm))
	}
	return buf.downscale(w, h)
}

// scaleMesh returns a mesh whose geometry is scaled into SSAA space.
func scaleMesh(m *scene.Mesh, s float64) *scene.Mesh {
	out := *m
	g := m.Geometry
	switch m.Kind {
	case "rect":
		g.X *= s
		g.Y *= s
		g.W *= s
		g.H *= s
	case "circle":
		g.CX *= s
		g.CY *= s
		g.Radius *= s
	case "polygon":
		pts := make([][2]float64, len(g.Points))
		for i, p := range g.Points {
			pts[i] = [2]float64{p[0] * s, p[1] * s}
		}
		g.Points = pts
	}
	out.Geometry = g
	return &out
}

// composite draws top over bottom with source-over and returns a new image.
func composite(bottom, top *image.NRGBA) *image.NRGBA {
	b := bottom.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := bottom.NRGBAAt(x, y)
			s := top.NRGBAAt(x, y)
			sa := float64(s.A) / 255
			da := float64(d.A) / 255
			ia := 1 - sa
			oa := sa + da*ia
			or := (float64(s.R)/255*sa + float64(d.R)/255*da*ia)
			og := (float64(s.G)/255*sa + float64(d.G)/255*da*ia)
			ob := (float64(s.B)/255*sa + float64(d.B)/255*da*ia)
			out.SetNRGBA(x, y, color.NRGBA{
				R: toByte(unpremul(or, oa)),
				G: toByte(unpremul(og, oa)),
				B: toByte(unpremul(ob, oa)),
				A: toByte(oa),
			})
		}
	}
	return out
}

func backgroundImage(s *scene.Scene) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s.Width, s.Height))
	if s.Background == nil {
		return img
	}
	c := *s.Background
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: toByte(c.R), G: toByte(c.G), B: toByte(c.B), A: 255,
			})
		}
	}
	return img
}

func meshSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// RenderScene renders every mesh of s over the background (the full picture).
func RenderScene(s *scene.Scene) *image.NRGBA {
	return composite(backgroundImage(s), renderMeshes(s.Width, s.Height, s.Meshes))
}

// RenderExcept renders background plus every mesh whose id is not in ids.
// Geometry and draw order of untouched meshes remain exactly as authored.
func RenderExcept(s *scene.Scene, ids []string) *image.NRGBA {
	skip := meshSet(ids)
	var keep []*scene.Mesh
	for _, m := range s.Meshes {
		if !skip[m.ID] {
			keep = append(keep, m)
		}
	}
	return composite(backgroundImage(s), renderMeshes(s.Width, s.Height, keep))
}

// RenderOver renders only the given meshes (in scene draw order) onto a
// transparent layer: the visual "over" contribution.
func RenderOver(s *scene.Scene, meshes []*scene.Mesh) *image.NRGBA {
	return renderMeshes(s.Width, s.Height, meshes)
}

// Compose stacks top over base and returns the final image.
func Compose(base, top *image.NRGBA) *image.NRGBA {
	return composite(base, top)
}
