// Package mipchain creates deterministic source textures and box-downsampled
// mip levels without using a GPU or graphics SDK.
package mipchain

import (
	"image"
	"math"
)

// Albedo creates a deterministic sRGB test texture with low- and high-frequency
// detail so each reduced mip level has measurable content.
func Albedo(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cell := size / 8
	fineGrid := size / 64
	if fineGrid < 1 {
		fineGrid = 1
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			u := (float64(x) + 0.5) / float64(size)
			v := (float64(y) + 0.5) / float64(size)

			r := 42 + 70*u
			g := 58 + 90*v
			b := 92 + 110*(1.0-(u+v)*0.5)

			if ((x/cell)+(y/cell))%2 == 0 {
				r = r*0.55 + 214*0.45
				g = g*0.55 + 139*0.45
				b = b*0.55 + 76*0.45
			}

			dx := u - 0.5
			dy := v - 0.5
			radius := math.Sqrt(dx*dx+dy*dy) * 2.0
			ring := math.Abs(math.Sin(radius * math.Pi * 7.0))
			if ring > 0.91 && radius < 1.7 {
				r = r*0.25 + 245*0.75
				g = g*0.25 + 221*0.75
				b = b*0.25 + 96*0.75
			}

			if x%fineGrid == 0 || y%fineGrid == 0 {
				r *= 0.42
				g *= 0.42
				b *= 0.48
			}
			if x < 3 || y < 3 || x >= size-3 || y >= size-3 {
				r, g, b = 18, 22, 30
			}

			offset := img.PixOffset(x, y)
			img.Pix[offset+0] = clampByte(r)
			img.Pix[offset+1] = clampByte(g)
			img.Pix[offset+2] = clampByte(b)
			img.Pix[offset+3] = 255
		}
	}
	return img
}

// NormalMap creates a deterministic tangent-space normal map. RGB storage is
// the conventional [-1, 1] to [0, 255] encoding; the container marker must be
// linear/data rather than sRGB.
func NormalMap(size int, strength float64) *image.RGBA {
	if strength == 0 {
		strength = 3.2
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	heightAt := func(x, y int) float64 {
		u := float64(x) / float64(size)
		v := float64(y) / float64(size)
		const tau = 2.0 * math.Pi
		return 0.5 +
			0.18*math.Sin(tau*6.0*u)*math.Cos(tau*6.0*v) +
			0.08*math.Sin(tau*18.0*(u+v)) +
			0.045*math.Cos(tau*31.0*(u-v))
	}
	wrapped := func(x, y int) float64 {
		x = (x + size) % size
		y = (y + size) % size
		return heightAt(x, y)
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			tl := wrapped(x-1, y-1)
			tc := wrapped(x, y-1)
			tr := wrapped(x+1, y-1)
			ml := wrapped(x-1, y)
			mr := wrapped(x+1, y)
			bl := wrapped(x-1, y+1)
			bc := wrapped(x, y+1)
			br := wrapped(x+1, y+1)

			gx := -tl + tr - 2*ml + 2*mr - bl + br
			gy := -tl - 2*tc - bl + tr + 2*bc + br
			nx := -gx * strength
			ny := -gy * strength
			nz := 1.0
			length := math.Sqrt(nx*nx + ny*ny + nz*nz)
			nx /= length
			ny /= length
			nz /= length

			offset := img.PixOffset(x, y)
			img.Pix[offset+0] = clampByte((nx*0.5 + 0.5) * 255)
			img.Pix[offset+1] = clampByte((ny*0.5 + 0.5) * 255)
			img.Pix[offset+2] = clampByte((nz*0.5 + 0.5) * 255)
			img.Pix[offset+3] = 255
		}
	}
	return img
}

func clampByte(value float64) uint8 {
	return uint8(math.Round(math.Min(255, math.Max(0, value))))
}
