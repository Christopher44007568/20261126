package atlas

import (
	"image"
	"image/color"
	"testing"
)

func TestOddSizedRotatedSpritesCanPassValidation(t *testing.T) {
	mk := func(id string, w, h int, c color.RGBA) SpriteImage {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				cc := c
				if x < 2 || y < 2 || x >= w-2 || y >= h-2 {
					cc = color.RGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: 255}
				}
				img.SetRGBA(x, y, cc)
			}
		}
		return SpriteImage{ID: id, Image: img}
	}

	sprites := []SpriteImage{
		mk("a", 13, 17, color.RGBA{R: 200, G: 30, B: 30, A: 255}),
		mk("b", 31, 7, color.RGBA{R: 30, G: 200, B: 220, A: 255}),
		mk("c", 9, 23, color.RGBA{R: 220, G: 30, B: 200, A: 255}),
	}
	built, err := BuildAtlas(sprites, BuildOptions{
		MipLevels: 3,
		Extrude:   true,
		Pack: PackOptions{
			Width: 256, Height: 256, AllowRotation: true,
			Policy: PaddingPolicy{Padding: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	report := ValidateSampling(built, "odd", ValidateOptions{Grid: 11, ColorTolerance: 24})
	if !report.OK {
		t.Fatalf("odd-size atlas has %d violations; first=%+v", report.ViolationCount, report.Violations[0])
	}
}
