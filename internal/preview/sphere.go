package preview

import (
	"math"

	"dds-preview/internal/dds"
)

const sphereRadiusFraction = 0.46

// RenderSphere renders a lit material sphere into an image with exactly the
// base mip's dimensions. It uses only CPU math and bilinear texture sampling.
func RenderSphere(tex *dds.Texture) dds.Image {
	w, h := tex.Width, tex.Height
	img := dds.Image{Width: w, Height: h, RGBA: make([]byte, w*h*4)}
	cx, cy := float64(w)/2, float64(h)/2
	radius := sphereRadiusFraction * float64(min(w, h))
	light := normalize([3]float64{-0.45, 0.58, 0.72})
	half := normalize([3]float64{light[0], light[1], light[2] + 1})

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px := (float64(x) + 0.5 - cx) / radius
			py := (cy - float64(y) - 0.5) / radius
			z2 := 1 - px*px - py*py
			if z2 <= 0 {
				writeBackground(img.RGBA, x, y, w, h)
				continue
			}
			nz := math.Sqrt(z2)
			n := [3]float64{px, py, nz}
			dir := normalize([3]float64{px, py, nz})
			var r, g, b, a float64
			if tex.IsCubemap {
				r, g, b, a = sampleCubemap(tex, dir)
			} else {
				u := 0.5 + math.Atan2(dir[0], dir[2])/(2*math.Pi)
				v := 0.5 - math.Asin(clamp(dir[1], -1, 1))/math.Pi
				r, g, b, a = sampleBilinear(tex.Base(0), u, v)
			}
			diffuse := math.Max(0, dot(n, light))
			specular := math.Pow(math.Max(0, dot(n, half)), 36)
			rim := math.Pow(1-math.Abs(nz), 3) * 0.12
			shade := 0.20 + 0.80*diffuse
			r = clamp01(r*shade + specular*0.32 + rim)
			g = clamp01(g*shade + specular*0.32 + rim)
			b = clamp01(b*shade + specular*0.32 + rim)
			if a < 1 {
				br, bg, bb := backgroundAt(x, y, w, h)
				r = r*a + br*(1-a)
				g = g*a + bg*(1-a)
				b = b*a + bb*(1-a)
			}
			writePixel(img.RGBA, x, y, w, toByte(r), toByte(g), toByte(b))
		}
	}
	return img
}

func sampleCubemap(tex *dds.Texture, d [3]float64) (float64, float64, float64, float64) {
	ax, ay, az := math.Abs(d[0]), math.Abs(d[1]), math.Abs(d[2])
	var face int
	var u, v float64
	switch {
	case ax >= ay && ax >= az && d[0] > 0:
		face, u, v = 0, -d[2]/ax, -d[1]/ax
	case ax >= ay && ax >= az:
		face, u, v = 1, d[2]/ax, -d[1]/ax
	case ay >= az && d[1] > 0:
		face, u, v = 2, d[0]/ay, d[2]/ay
	case ay >= az:
		face, u, v = 3, d[0]/ay, -d[2]/ay
	case d[2] > 0:
		face, u, v = 4, d[0]/az, -d[1]/az
	default:
		face, u, v = 5, -d[0]/az, -d[1]/az
	}
	return sampleBilinear(tex.Base(face), 0.5+0.5*u, 0.5+0.5*v)
}

func sampleBilinear(img dds.Image, u, v float64) (float64, float64, float64, float64) {
	u = u - math.Floor(u)
	v = clamp(v, 0, 1)
	// The half-texel offset keeps a full-width sphere from clipping the seam.
	fx := u*float64(img.Width) - 0.5
	fy := v*float64(img.Height) - 0.5
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	x1, y1 := x0+1, y0+1
	x0 = ((x0 % img.Width) + img.Width) % img.Width
	x1 = ((x1 % img.Width) + img.Width) % img.Width
	y0 = clampInt(y0, 0, img.Height-1)
	y1 = clampInt(y1, 0, img.Height-1)
	read := func(x, y int) (float64, float64, float64, float64) {
		i := (y*img.Width + x) * 4
		return float64(img.RGBA[i]) / 255, float64(img.RGBA[i+1]) / 255,
			float64(img.RGBA[i+2]) / 255, float64(img.RGBA[i+3]) / 255
	}
	r00, g00, b00, a00 := read(x0, y0)
	r10, g10, b10, a10 := read(x1, y0)
	r01, g01, b01, a01 := read(x0, y1)
	r11, g11, b11, a11 := read(x1, y1)
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	r0 := lerp(r00, r10, tx)
	r1 := lerp(r01, r11, tx)
	g0 := lerp(g00, g10, tx)
	g1 := lerp(g01, g11, tx)
	b0 := lerp(b00, b10, tx)
	b1 := lerp(b01, b11, tx)
	a0 := lerp(a00, a10, tx)
	a1 := lerp(a01, a11, tx)
	return lerp(r0, r1, ty), lerp(g0, g1, ty), lerp(b0, b1, ty), lerp(a0, a1, ty)
}

func backgroundAt(x, y, w, h int) (float64, float64, float64) {
	checker := ((x/16)+(y/16))&1 == 0
	base := 0.11
	if checker {
		base = 0.15
	}
	vignette := 0.75 + 0.25*float64(y)/float64(max(1, h-1))
	v := (base + 0.06*float64(w-x)/float64(max(1, w))) * vignette
	return v, v * 0.95, v * 0.88
}

func writeBackground(dst []byte, x, y, w, h int) {
	r, g, b := backgroundAt(x, y, w, h)
	writePixel(dst, x, y, w, toByte(r), toByte(g), toByte(b))
}

func writePixel(dst []byte, x, y, w int, r, g, b byte) {
	i := (y*w + x) * 4
	dst[i], dst[i+1], dst[i+2], dst[i+3] = r, g, b, 255
}

func normalize(v [3]float64) [3]float64 {
	n := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	if n == 0 {
		return [3]float64{0, 0, 1}
	}
	return [3]float64{v[0] / n, v[1] / n, v[2] / n}
}

func dot(a, b [3]float64) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func toByte(v float64) byte {
	return byte(math.Round(clamp01(v) * 255))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
