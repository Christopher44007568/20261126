package main

import (
	"fmt"
	"math"
	"os"
)

type Image struct {
	W, H int
	Pix  []Vec3
}

func NewImage(w, h int) *Image { return &Image{W: w, H: h, Pix: make([]Vec3, w*h)} }

func (im *Image) Set(x, y int, c Vec3) { im.Pix[y*im.W+x] = c }
func (im *Image) Get(x, y int) Vec3    { return im.Pix[y*im.W+x] }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (im *Image) SavePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "P6\n%d %d\n255\n", im.W, im.H)
	buf := make([]byte, 3*im.W*im.H)
	for i, p := range im.Pix {
		buf[3*i] = byte(255*clamp01(p.X) + 0.5)
		buf[3*i+1] = byte(255*clamp01(p.Y) + 0.5)
		buf[3*i+2] = byte(255*clamp01(p.Z) + 0.5)
	}
	_, err = f.Write(buf)
	return err
}

type Camera struct {
	Orig, LowerLeft, Horiz, Vert Vec3
}

func NewCamera(w, h int) Camera {
	vh := 2.0
	vw := vh * float64(w) / float64(h)
	focal := 2.5
	return Camera{
		Orig:      Vec3{0, 0, 0},
		LowerLeft: Vec3{-vw / 2, -vh / 2, -focal},
		Horiz:     Vec3{vw, 0, 0},
		Vert:      Vec3{0, vh, 0},
	}
}

func (c Camera) Ray(u, v float64) Ray {
	dir := c.LowerLeft.Add(c.Horiz.Scale(u)).Add(c.Vert.Scale(v)).Sub(c.Orig)
	return Ray{Orig: c.Orig, Dir: dir.Norm()}
}

// Channels holds the four output images for one scene configuration.
type Channels struct {
	Primary    *Image
	Shadow     *Image
	Reflection *Image
	Heatmap    *Image
}

func heatColor(n float64) Vec3 {
	// blue -> red ramp
	t := clamp01(n)
	return Vec3{t, 0.25 * t * (1 - t) * 4 * 0.25, 1 - t}
}

// Render produces primary, shadow and reflection channel images plus a
// heatmap counting mask-excluded instances encountered along each path.
func Render(s *Scene, cam Camera, w, h int) *Channels {
	ch := &Channels{
		Primary:    NewImage(w, h),
		Shadow:     NewImage(w, h),
		Reflection: NewImage(w, h),
		Heatmap:    NewImage(w, h),
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u := (float64(x) + 0.5) / float64(w)
			v := 1 - (float64(y)+0.5)/float64(h)
			ray := cam.Ray(u, v)

			hit, excl, ok := s.Intersect(ray, RayPrimary)
			heat := float64(len(excl))
			if !ok {
				ch.Primary.Set(x, y, Vec3{0.08, 0.08, 0.1})
				ch.Heatmap.Set(x, y, heatColor(heat/4))
				continue
			}

			// Shadow ray toward the light.
			toLight := s.Light.Sub(hit.P)
			dist := toLight.Len()
			sray := Ray{Orig: hit.P.Add(hit.N.Scale(1e-3)), Dir: toLight.Scale(1 / dist)}
			occl, sexcl, blocked := s.Intersect(sray, RayShadow)
			heat += float64(len(sexcl))
			lit := 1.0
			if blocked && occl.T < dist {
				lit = 0.15
				ch.Shadow.Set(x, y, occl.Inst.Color)
			} else {
				ch.Shadow.Set(x, y, Vec3{1, 1, 1})
			}

			// Reflection ray.
			rray := Ray{
				Orig: hit.P.Add(hit.N.Scale(1e-3)),
				Dir:  Reflect(ray.Dir, hit.N).Norm(),
			}
			rhit, rexcl, rok := s.Intersect(rray, RayReflection)
			heat += float64(len(rexcl))
			var rcol Vec3
			if rok {
				rcol = rhit.Inst.Color
				ch.Reflection.Set(x, y, rcol)
			}

			diff := math.Max(0, hit.N.Dot(toLight.Norm()))
			shade := s.Ambient + (1-s.Ambient)*diff*lit
			pcol := hit.Inst.Color.Scale(shade)
			if rok {
				pcol = pcol.Scale(0.8).Add(rcol.Scale(0.2))
			}
			ch.Primary.Set(x, y, pcol)
			ch.Heatmap.Set(x, y, heatColor(heat/4))
		}
	}
	return ch
}
