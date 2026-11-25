// Package scene builds a deterministic synthetic linear-light HDR scene.
// All radiance values are scene-referred float RGB; nothing is gamma encoded
// here, and emissive elements deliberately exceed 1.0 so the highlight
// headroom path can be exercised.
package scene

import (
	"math"

	"hdrexport/internal/hdr"
)

type sphere struct {
	cx, cy, rad float64
	base        [3]float64 // diffuse reflectance (linear)
	ambient     float64
}

// Build renders the demo scene at the requested size.
func Build(w, h int) *hdr.Image {
	m := hdr.NewImage(w, h)
	horizon := int(float64(h) * 0.62)

	// Directional light vector (points from surface toward the light).
	light := normalize([3]float64{-0.45, -0.55, 0.7})

	spheres := []sphere{
		{cx: 0.28, cy: 0.66, rad: 0.17, base: [3]float64{0.85, 0.28, 0.22}, ambient: 0.10},
		{cx: 0.50, cy: 0.70, rad: 0.21, base: [3]float64{0.30, 0.62, 0.90}, ambient: 0.12},
		{cx: 0.70, cy: 0.64, rad: 0.13, base: [3]float64{0.85, 0.80, 0.30}, ambient: 0.10},
	}

	sunCx, sunCy := 0.82, 0.20
	sunRad := 0.115
	bulbCx, bulbCy := 0.435, 0.585

	for y := 0; y < h; y++ {
		fy := float64(y) / float64(h-1)
		for x := 0; x < w; x++ {
			fx := float64(x) / float64(w-1)
			var r, g, b float64
			if y < horizon {
				// Sky: linear vertical gradient, stays below 1.0.
				t := fy / (float64(horizon) / float64(h))
				r = 0.05 + 0.20*(1-t)
				g = 0.07 + 0.25*(1-t)
				b = 0.10 + 0.38*(1-t)
			} else {
				// Ground: dim neutral base.
				t := (fy - float64(horizon)/float64(h)) / (1 - float64(horizon)/float64(h))
				v := 0.05 + 0.05*(1-t)
				r, g, b = v, v*0.98, v*0.92
			}

			for _, sp := range spheres {
				dx, dy := fx-sp.cx, fy-sp.cy
				d2 := dx*dx + dy*dy
				if d2 <= sp.rad*sp.rad {
					dz := math.Sqrt(sp.rad*sp.rad - d2)
					n := normalize([3]float64{dx, dy, dz / float64(w) * float64(h)})
					diffuse := math.Max(0, dot(n, light))
					shade := sp.ambient + 1.05*diffuse
					r = sp.base[0] * shade
					g = sp.base[1] * shade
					b = sp.base[2] * shade

					// Small specular bulb, peak ~4x SDR white.
					sdx, sdy := fx-bulbCx, fy-bulbCy
					sd := math.Sqrt(sdx*sdx + sdy*sdy)
					if sd < 0.022 {
						k := math.Pow(1-sd/0.022, 2) * 4.0
						r += k
						g += k * 0.95
						b += k * 0.85
					}
					break
				}
			}

			// Emissive disc: smooth falloff with a 10.0 linear peak.
			sdx, sdy := fx-sunCx, fy-sunCy
			sd := math.Sqrt(sdx*sdx + sdy*sdy)
			if sd < sunRad {
				k := math.Pow(1-sd/sunRad, 1.6) * 10.0
				r += k
				g += k * 0.97
				b += k * 0.85
			}

			m.Set(x, y, float32(r), float32(g), float32(b))
		}
	}
	return m
}

func normalize(v [3]float64) [3]float64 {
	l := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	return [3]float64{v[0] / l, v[1] / l, v[2] / l}
}

func dot(a, b [3]float64) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}
