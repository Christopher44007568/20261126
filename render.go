package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
)

type Hit struct {
	T       float32
	ID      int32
	N, C    Vec3
	Invalid bool
}

type Scene struct {
	BLAS  []*BVH
	CBLAS []*CBVH
	TLAS  []Node
}

var inf = float32(math.Inf(1))

// Intersect walks the TLAS, then the chosen BLAS layout. Hit selection is
// order-independent (min t, tie-break on triangle ID), so the compressed
// traversal — which may visit nodes in a different order and test a
// superset of boxes — returns bit-identical results.
func (s *Scene) Intersect(r Ray, compressed, checked bool) Hit {
	best := Hit{T: inf, ID: -1}
	stack := [256]int32{0}
	sp := 1
	for sp > 0 {
		sp--
		nd := s.TLAS[stack[sp]]
		if !slabHit(r, nd.Min, nd.Max, best.T) {
			continue
		}
		if nd.Count > 0 {
			if compressed {
				s.intersectCBLAS(r, nd.First, &best, checked)
			} else {
				s.intersectBLAS(r, nd.First, &best)
			}
			if best.Invalid {
				return best
			}
		} else {
			stack[sp] = nd.Left
			sp++
			stack[sp] = nd.Right
			sp++
		}
	}
	return best
}

func (s *Scene) updateBest(best *Hit, tri *Triangle, r Ray) {
	if t, ok := tri.Intersect(r); ok && (t < best.T || (t == best.T && tri.ID < best.ID)) {
		best.T, best.ID, best.N, best.C = t, tri.ID, tri.N, tri.C
	}
}

func (s *Scene) intersectBLAS(r Ray, bi int32, best *Hit) {
	b := s.BLAS[bi]
	stack := [64]int32{0}
	sp := 1
	for sp > 0 {
		sp--
		nd := b.Nodes[stack[sp]]
		if !slabHit(r, nd.Min, nd.Max, best.T) {
			continue
		}
		if nd.Count > 0 {
			for k := nd.First; k < nd.First+nd.Count; k++ {
				s.updateBest(best, &b.Tris[b.Indices[k]], r)
			}
		} else {
			stack[sp] = nd.Left
			sp++
			stack[sp] = nd.Right
			sp++
		}
	}
}

func (s *Scene) intersectCBLAS(r Ray, bi int32, best *Hit, checked bool) {
	c := s.CBLAS[bi]
	stack := [64]int32{0}
	sp := 1
	for sp > 0 {
		sp--
		ni := stack[sp]
		if checked && (ni < 0 || int(ni) >= len(c.Nodes)) {
			best.Invalid = true
			return
		}
		nd := &c.Nodes[ni]
		mn, mx := c.nodeBounds(nd)
		if !slabHit(r, mn, mx, best.T) {
			continue
		}
		if nd.Meta&leafFlag != 0 {
			first := int32(nd.Meta & 0xffff)
			cnt := int32((nd.Meta >> 16) & 0x7fff)
			if checked && (cnt == 0 || first < 0 || first+cnt > int32(len(c.Tris))) {
				best.Invalid = true
				return
			}
			for k := first; k < first+cnt; k++ {
				s.updateBest(best, &c.Tris[k], r)
			}
		} else {
			l := int32(nd.Meta & 0x7fffffff)
			stack[sp] = l
			sp++
			stack[sp] = l + 1
			sp++
		}
	}
}

var toSun = vnorm(Vec3{0.45, 0.8, 0.3})

func (s *Scene) shade(r Ray, h Hit, compressed, checked bool) Vec3 {
	if h.ID < 0 {
		t := 0.5 * (r.Dir.Y + 1)
		return vadd(vscale(Vec3{0.85, 0.90, 0.95}, 1-t), vscale(Vec3{0.35, 0.55, 0.85}, t))
	}
	n := h.N
	if vdot(n, r.Dir) > 0 {
		n = vscale(n, -1)
	}
	p := vadd(r.Orig, vscale(r.Dir, h.T))
	sr := NewRay(vadd(p, vscale(n, 1e-4)), toSun)
	sh := s.Intersect(sr, compressed, checked)
	vis := float32(1)
	if sh.ID >= 0 {
		vis = 0.25
	}
	dif := max(0, vdot(n, toSun))
	return vscale(h.C, 0.25+0.75*dif*vis)
}

type Camera struct {
	Pos, Look, Up Vec3
	FovYDeg       float32
}

func (c Camera) RayAt(x, y, w, h int) Ray {
	aspect := float32(w) / float32(h)
	f := 1 / float32(math.Tan(float64(c.FovYDeg*math.Pi/180/2)))
	fwd := vnorm(vsub(c.Look, c.Pos))
	right := vnorm(vcross(fwd, c.Up))
	up := vcross(right, fwd)
	u := ((float32(x)+0.5)/float32(w)*2 - 1) * aspect
	v := 1 - (float32(y)+0.5)/float32(h)*2
	d := vnorm(vadd(vadd(vscale(right, u), vscale(up, v)), vscale(fwd, f)))
	return NewRay(c.Pos, d)
}

type Image struct {
	W, H int
	Pix  []Vec3
	IDs  []int32 // hit triangle ID; -1 miss; -2 invalid reference
}

// Render is parallel over row bands; every pixel is computed independently,
// so output is fully deterministic regardless of scheduling.
func Render(s *Scene, cam Camera, w, h int, compressed, checked bool) *Image {
	img := &Image{W: w, H: h, Pix: make([]Vec3, w*h), IDs: make([]int32, w*h)}
	bands := runtime.NumCPU()
	rowsPer := (h + bands - 1) / bands
	var wg sync.WaitGroup
	for b := 0; b < bands; b++ {
		y0, y1 := b*rowsPer, min((b+1)*rowsPer, h)
		if y0 >= y1 {
			continue
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			for y := y0; y < y1; y++ {
				for x := 0; x < w; x++ {
					ray := cam.RayAt(x, y, w, h)
					hit := s.Intersect(ray, compressed, checked)
					i := y*w + x
					img.Pix[i] = s.shade(ray, hit, compressed, checked)
					img.IDs[i] = hit.ID
					if hit.Invalid {
						img.IDs[i] = -2
					}
				}
			}
		}(y0, y1)
	}
	wg.Wait()
	return img
}

func (img *Image) SavePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "P6\n%d %d\n255\n", img.W, img.H)
	buf := make([]byte, 3*img.W)
	for y := 0; y < img.H; y++ {
		for x := 0; x < img.W; x++ {
			c := img.Pix[y*img.W+x]
			for k, v := range [3]float32{c.X, c.Y, c.Z} {
				g := math.Pow(float64(max(0, min(1, v))), 1/2.2)
				buf[3*x+k] = byte(g*255 + 0.5)
			}
		}
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	return w.Flush()
}

// DiffImage returns |a-b| amplified by scale, plus the max abs difference.
func DiffImage(a, b *Image, scale float32) (*Image, float32) {
	out := &Image{W: a.W, H: a.H, Pix: make([]Vec3, len(a.Pix))}
	maxd := float32(0)
	for i := range a.Pix {
		d := vsub(a.Pix[i], b.Pix[i])
		d = Vec3{float32(math.Abs(float64(d.X))), float32(math.Abs(float64(d.Y))), float32(math.Abs(float64(d.Z)))}
		maxd = max(maxd, max(d.X, max(d.Y, d.Z)))
		out.Pix[i] = vscale(d, scale)
	}
	return out, maxd
}

// HighlightInvalid dims the scene and paints invalid-reference pixels red.
func HighlightInvalid(img *Image) (*Image, int) {
	out := &Image{W: img.W, H: img.H, Pix: make([]Vec3, len(img.Pix))}
	count := 0
	for i := range img.Pix {
		if img.IDs[i] == -2 {
			out.Pix[i] = Vec3{1, 0, 0}
			count++
		} else {
			out.Pix[i] = vscale(img.Pix[i], 0.12)
		}
	}
	return out, count
}
