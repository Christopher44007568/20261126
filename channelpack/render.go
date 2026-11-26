package main

import (
	"image"
	"image/color"
	"math"
)

// renderSphere shades a circular material ball. The tangent-space normal map
// is treated as the visible surface normal, while roughness widens the spec
// lobe and occlusion darkens ambient response.
func renderSphere(m *Maps, size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	center := float64(size-1) / 2
	radius := float64(size) * 0.48
	lx, ly, lz := normalize3(-0.45, -0.55, 0.70)
	viewZ := 1.0

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			dx := (float64(px) - center) / radius
			dy := (float64(py) - center) / radius
			d2 := dx*dx + dy*dy
			if d2 > 1 {
				img.SetNRGBA(px, py, color.NRGBA{
					R: sphereBG.r, G: sphereBG.g, B: sphereBG.b, A: 255,
				})
				continue
			}
			u := clampInt(px*m.W/size, 0, m.W-1)
			v := clampInt(py*m.H/size, 0, m.H-1)
			i := v*m.W + u
			c := i * 3
			nx, ny, nz := normalVector(m.Normal[c], m.Normal[c+1], m.Normal[c+2])

			// Keep a small silhouette component so the ball stays round, while
			// letting decoded normals dominate. A swapped slot therefore shows
			// up immediately as distorted lighting instead of being masked.
			silZ := math.Sqrt(math.Max(0, 1-d2))
			nx, ny, nz = normalize3(nx*0.92+dx*0.08, ny*0.92+dy*0.08, nz*0.92+silZ*0.08)

			diff := math.Max(0, nx*lx+ny*ly+nz*lz)
			rough := float64(m.Roughness[i]) / 255
			ao := float64(m.Occlusion[i]) / 255

			// Half-vector Blinn-Phong term; roughness expands the lobe.
			hvx, hvy, hvz := normalize3(lx, ly, lz+viewZ)
			ndoth := math.Max(0, nx*hvx+ny*hvy+nz*hvz)
			shininess := 4 + 96*(1-rough)*(1-rough)
			spec := math.Pow(ndoth, shininess) * (1 - rough*0.65)

			ambient := 0.16 + 0.24*ao
			albedo := [3]float64{
				float64(m.Albedo[c]) / 255,
				float64(m.Albedo[c+1]) / 255,
				float64(m.Albedo[c+2]) / 255,
			}
			col := [3]float64{}
			for k := 0; k < 3; k++ {
				v := albedo[k]*(ambient+0.85*diff*ao) + spec
				if v > 1 {
					v = 1
				}
				col[k] = v
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: clampByte(col[0] * 255),
				G: clampByte(col[1] * 255),
				B: clampByte(col[2] * 255),
				A: 255,
			})
		}
	}
	return img
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
