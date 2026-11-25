package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return flattenImage(img, color.RGBA{R: 10, G: 12, B: 16, A: 255}), nil
}

func flattenImage(src image.Image, bg color.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func renderSwatches(m *Manifest, valueField string) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, m.Canvas.Width, m.Canvas.Height))
	fillRect(img, img.Bounds(), color.RGBA{R: 12, G: 16, B: 22, A: 255})
	for _, p := range m.Patches {
		hex := p.Source
		if valueField == "target" {
			hex = p.Target
		}
		if hex == "" {
			return nil, fmt.Errorf("patch %q has no %s color and no %s image was supplied", p.ID, valueField, valueField)
		}
		r, g, b, err := parseHexColor(hex)
		if err != nil {
			return nil, err
		}
		fillRect(img, p.Rect.imageRect(), color.RGBA{R: r, G: g, B: b, A: 255})
	}
	return img, nil
}

func validateImageSize(img image.Image, m *Manifest, role string) error {
	b := img.Bounds()
	if b.Dx() < m.Canvas.Width || b.Dy() < m.Canvas.Height {
		return fmt.Errorf("%s image is %dx%d, smaller than manifest canvas %dx%d",
			role, b.Dx(), b.Dy(), m.Canvas.Width, m.Canvas.Height)
	}
	return nil
}

func cropToCanvas(img image.Image, m *Manifest) image.Image {
	b := img.Bounds()
	if b.Dx() == m.Canvas.Width && b.Dy() == m.Canvas.Height {
		return img
	}
	out := image.NewRGBA(image.Rect(0, 0, m.Canvas.Width, m.Canvas.Height))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}
