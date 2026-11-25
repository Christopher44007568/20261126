package main

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
)

type RenderResult struct {
	W, H  int
	Color []Vec3
	Depth []float64
}

type RenderOptions struct {
	Samples int
	Seed    int64
}

func Render(scene *Scene, cam *Camera, width, height int, opts RenderOptions) *RenderResult {
	if opts.Samples <= 0 {
		opts.Samples = 64
	}
	out := &RenderResult{
		W:     width,
		H:     height,
		Color: make([]Vec3, width*height),
		Depth: make([]float64, width*height),
	}

	rows := make(chan int)
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers > height {
		workers = height
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				rng := rand.New(rand.NewSource(opts.Seed + int64(y)*1000003))
				for x := 0; x < width; x++ {
					col := Vec3{}
					for s := 0; s < opts.Samples; s++ {
						sx := rng.Float64()
						sy := rng.Float64()
						lensU := rng.Float64()
						lensV := rng.Float64()
						u := (float64(x) + sx) / float64(width)
						v := 1 - (float64(y)+sy)/float64(height)
						ray := cam.ray(u, v, lensU, lensV)
						col = col.Add(traceRay(ray, scene, 0, rng))
					}
					i := y*width + x
					out.Color[i] = col.Div(float64(opts.Samples))

					center := cam.ray((float64(x)+0.5)/float64(width), 1-(float64(y)+0.5)/float64(height), 0, 0)
					if h, ok := scene.Nearest(center); ok {
						out.Depth[i] = h.T
					} else {
						out.Depth[i] = math.Inf(1)
					}
				}
			}
		}()
	}
	for y := 0; y < height; y++ {
		rows <- y
	}
	close(rows)
	wg.Wait()
	return out
}

func traceRay(r Ray, scene *Scene, bounce int, rng *rand.Rand) Vec3 {
	h, ok := scene.Nearest(r)
	if !ok {
		if scene.BackgroundColor != nil {
			return scene.BackgroundColor(r)
		}
		return V(0.02, 0.03, 0.06)
	}

	mat := h.Mat
	albedo := mat.Albedo
	if mat.Texture != nil {
		tex := mat.Texture.Sample(h.U, h.V)
		if mat.TexStrength > 0 {
			albedo = albedo.Lerp(tex, clamp01(mat.TexStrength))
		} else {
			albedo = tex
		}
	}
	if mat.Unlit {
		return mat.Emission.Add(albedo)
	}

	n := h.Normal
	if n.Dot(r.Dir) > 0 {
		n = n.Mul(-1)
	}
	view := r.Dir.Mul(-1).Norm()
	color := mat.Emission
	color = color.Add(albedo.MulVec(V(0.10, 0.12, 0.17)))

	lights := []struct {
		dir       Vec3
		color     Vec3
		intensity float64
	}{
		{V(-0.45, 0.82, -0.55).Norm(), V(1.0, 0.93, 0.84), 1.15},
		{V(0.70, 0.35, -0.45).Norm(), V(0.45, 0.62, 1.0), 0.42},
	}
	for _, light := range lights {
		shadowRay := Ray{Orig: h.P.Add(n.Mul(0.0015)), Dir: light.dir}
		if _, blocked := scene.Nearest(shadowRay); blocked {
			continue
		}
		diff := math.Max(0, n.Dot(light.dir))
		color = color.Add(albedo.MulVec(light.color).Mul(diff * light.intensity))
		if mat.Shininess > 0 {
			half := light.dir.Add(view).Norm()
			spec := math.Pow(math.Max(0, n.Dot(half)), mat.Shininess)
			specColor := V(1, 1, 1)
			if mat.Metallic > 0 {
				specColor = albedo
			}
			color = color.Add(specColor.Mul(spec * light.intensity * (0.25 + mat.Metallic)))
		}
	}

	if mat.Metallic > 0.15 && bounce < 1 {
		reflDir := r.Dir.Sub(n.Mul(2 * r.Dir.Dot(n))).Norm()
		refl := traceRay(Ray{Orig: h.P.Add(n.Mul(0.002)), Dir: reflDir}, scene, bounce+1, rng)
		reflectAmount := 0.18 + 0.55*mat.Metallic
		color = color.Lerp(refl.MulVec(albedo.Add(V(0.35, 0.35, 0.35))), reflectAmount)
	}
	return color
}

func ACESFilm(x float64) float64 {
	a, b, c, d, e := 2.51, 0.03, 2.43, 0.59, 0.14
	return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0, 1)
}

func EncodeSRGB(x float64) uint8 {
	x = clamp01(x)
	if x <= 0.0031308 {
		x = 12.92 * x
	} else {
		x = 1.055*math.Pow(x, 1/2.4) - 0.055
	}
	return uint8(clamp(x*255+0.5, 0, 255))
}

func (r *RenderResult) ToCanvas(exposure float64) *Canvas {
	c := NewCanvas(r.W, r.H)
	if exposure <= 0 {
		exposure = 1
	}
	for i, col := range r.Color {
		y := i / r.W
		x := i % r.W
		c.setRGB(x, y,
			EncodeSRGB(ACESFilm(col.X*exposure)),
			EncodeSRGB(ACESFilm(col.Y*exposure)),
			EncodeSRGB(ACESFilm(col.Z*exposure)),
		)
	}
	return c
}
