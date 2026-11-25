package texturegen

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
)

const size = 320

type textureSpec struct {
	dir      string
	filename string
	kind     string
	seed     int64
}

var specs = []textureSpec{
	{"assets/textures", "oak_albedo.png", "oak", 11},
	{"assets/textures", "oak_normal.png", "normal", 12},
	{"assets/textures", "oak_roughness.png", "roughness", 13},
	{".cache/textures", "oak_albedo.png", "oak-cache", 21},
	{".cache/textures", "brass_albedo.png", "brass", 22},
	{".cache/textures", "brass_normal.png", "normal-metal", 23},
	{"assets/fallback", "brass_albedo.png", "fallback-brass", 31},
	{"assets/fallback", "brass_roughness.png", "roughness", 32},
	{"assets/fallback", "clay_normal.png", "normal-clay", 33},
}

// Ensure writes the deterministic demo textures when they are not present.
func Ensure(base string) error {
	for _, spec := range specs {
		path := filepath.Join(base, filepath.FromSlash(spec.dir), spec.filename)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		img := render(spec.kind, spec.seed)
		if err := writePNG(path, img); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// CheckerboardPNG returns the visible placeholder used for missing textures.
func CheckerboardPNG() ([]byte, error) {
	const tile = 32
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	light := color.RGBA{226, 218, 201, 255}
	dark := color.RGBA{91, 86, 77, 255}
	accent := color.RGBA{232, 139, 74, 255}
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			c := light
			if ((x/tile)+(y/tile))%2 == 1 {
				c = dark
			}
			if x < 5 || y < 5 || x >= 251 || y >= 251 {
				c = accent
			}
			img.SetRGBA(x, y, c)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func render(kind string, seed int64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	rng := rand.New(rand.NewSource(seed))
	noise := make([]float64, size*size)
	for i := range noise {
		noise[i] = rng.Float64()*2 - 1
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			n := noise[y*size+x]
			var c color.RGBA
			switch kind {
			case "oak":
				c = oak(x, y, n, false)
			case "oak-cache":
				c = oak(x, y, n, true)
			case "normal":
				c = normalMap(x, y, n, false)
			case "normal-metal":
				c = normalMap(x, y, n, true)
			case "normal-clay":
				c = normalMap(x, y, n, false)
				c.R = clamp(int(c.R) - 18)
				c.G = clamp(int(c.G) + 5)
				c.B = clamp(int(c.B) - 8)
			case "brass":
				c = brass(x, y, n)
			case "fallback-brass":
				c = brass(x, y, n)
				c.R = clamp(int(c.R) - 38)
				c.G = clamp(int(c.G) - 25)
				c.B = clamp(int(c.B) - 20)
			case "roughness":
				c = roughness(x, y, n)
			default:
				c = color.RGBA{120, 120, 120, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func oak(x, y int, n float64, cache bool) color.RGBA {
	fx, fy := float64(x), float64(y)
	warp := math.Sin(fy*0.024+math.Sin(fx*0.012)*2.4) * 7
	ring := math.Sin((fx+warp)*0.085 + math.Sin(fy*0.045)*1.3)
	fiber := math.Sin(fy*0.82+fx*0.07) * 0.10
	v := ring*0.45 + n*0.22 + fiber
	t := 0.5 + v*0.5
	base := mix(color.RGBA{76, 43, 25, 255}, color.RGBA{188, 130, 73, 255}, t)
	if cache {
		base = mix(color.RGBA{94, 55, 31, 255}, color.RGBA{205, 151, 95, 255}, t)
	}
	if math.Abs(ring) > 0.94 {
		base = mix(base, color.RGBA{40, 22, 15, 255}, 0.42)
	}
	return base
}

func brass(x, y int, n float64) color.RGBA {
	brush := math.Sin(float64(y)*1.7+math.Sin(float64(x)*0.011)*3) * 0.5
	v := 0.5 + brush*0.12 + n*0.06
	base := mix(color.RGBA{111, 74, 25, 255}, color.RGBA{229, 189, 101, 255}, v)
	spot := math.Exp(-math.Pow(float64(x)-235, 2)/18000 - math.Pow(float64(y)-76, 2)/8000)
	return mix(base, color.RGBA{255, 229, 160, 255}, spot*0.45)
}

func normalMap(x, y int, n float64, metal bool) color.RGBA {
	dx := math.Sin(float64(y)*0.07)*2.2 + math.Sin(float64(x)*0.11)*1.1
	dy := math.Cos(float64(x)*0.06)*2.0 + math.Cos(float64(y)*0.12)*1.2
	dx += n * 0.7
	dy += n * 0.5
	tint := 0.0
	if metal {
		tint = 7
	}
	return color.RGBA{
		clamp(int(128 + dx*3.2)),
		clamp(int(128 + dy*3.2 + tint)),
		clamp(int(247 - math.Abs(dx+dy)*1.3)),
		255,
	}
}

func roughness(x, y int, n float64) color.RGBA {
	fx, fy := float64(x), float64(y)
	wave := math.Sin(fx*0.032)*0.18 + math.Cos(fy*0.028)*0.15
	v := 0.58 + wave + n*0.10
	g := clamp(int(v * 255))
	return color.RGBA{uint8(g), uint8(g), uint8(g), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	return color.RGBA{
		clamp(int(float64(a.R)*(1-t) + float64(b.R)*t)),
		clamp(int(float64(a.G)*(1-t) + float64(b.G)*t)),
		clamp(int(float64(a.B)*(1-t) + float64(b.B)*t)),
		255,
	}
}

func clamp(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
