package atlas

import (
	"image"
	"image/color"
	"testing"
)

func TestPaddingPolicyDoublesEveryMip(t *testing.T) {
	p := PaddingPolicy{Padding: 1, MipLevels: 4}.Normalized()
	want := []int{1, 2, 4, 8}
	for level, expected := range want {
		if got := p.BasePixelsForMip(level); got != expected {
			t.Fatalf("level %d padding = %d, want %d", level, got, expected)
		}
	}
	if got := p.PackPadding(); got != 8 {
		t.Fatalf("pack padding = %d, want 8", got)
	}
}

func TestPackDoesNotOverlapAndCanRotate(t *testing.T) {
	sprites := []Sprite{{ID: "wide", W: 64, H: 16}}
	layout, err := Pack(sprites, PackOptions{
		Width: 32, Height: 128, AllowRotation: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Placements) != 1 || !layout.Placements[0].Rotated {
		t.Fatalf("expected wide sprite to rotate into narrow atlas: %+v", layout.Placements)
	}

	many := []Sprite{
		{ID: "a", W: 40, H: 20}, {ID: "b", W: 30, H: 30},
		{ID: "c", W: 24, H: 48}, {ID: "d", W: 16, H: 16},
	}
	layout, err = Pack(many, PackOptions{Width: 128, Height: 128, AllowRotation: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(layout.Placements); i++ {
		for j := i + 1; j < len(layout.Placements); j++ {
			a := layout.Placements[i].Cell
			b := layout.Placements[j].Cell
			if a.Intersects(b) {
				t.Fatalf("cells overlap: %s %s vs %s %s", layout.Placements[i].ID, a, layout.Placements[j].ID, b)
			}
		}
	}
}

func TestTightAtlasBleedsButMipPaddedAtlasPasses(t *testing.T) {
	inputs := testSprites()

	tight, err := BuildAtlas(inputs, BuildOptions{
		MipLevels: 4,
		Extrude:   false,
		Pack: PackOptions{
			Width: 256, Height: 256, AllowRotation: false, Alignment: 1,
			Policy: PaddingPolicy{Padding: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bad := ValidateSampling(tight, "tight", ValidateOptions{Grid: 9, ColorTolerance: 24})
	if bad.OK || bad.ViolationCount == 0 {
		t.Fatalf("expected tight atlas sampling violations, report=%+v", bad)
	}
	for _, stat := range bad.Levels {
		if stat.Violations == 0 {
			t.Fatalf("tight atlas has no bleed detected at mip %d", stat.Level)
		}
	}

	safe, err := BuildAtlas(inputs, BuildOptions{
		MipLevels: 4,
		Extrude:   true,
		Pack: PackOptions{
			AllowRotation: true,
			Alignment:     8,
			MinSize:       256,
			Policy:        PaddingPolicy{Padding: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	good := ValidateSampling(safe, "safe", ValidateOptions{Grid: 9, ColorTolerance: 24})
	if !good.OK {
		t.Fatalf("padded atlas has %d violations; first=%+v max=%.1f levels=%+v",
			good.ViolationCount, good.Violations[0], good.MaxDistance, good.Levels)
	}
	for _, stat := range good.Levels {
		if stat.Violations != 0 {
			t.Fatalf("mip %d has violations", stat.Level)
		}
	}

	rt := NewRuntimeAtlas(safe)
	for _, p := range safe.Layout.Placements {
		for _, lod := range []float64{0, 0.5, 1.5, 2.5, 3} {
			actual, err := rt.SampleLOD(p.ID, 0, 0.5, lod)
			if err != nil {
				t.Fatal(err)
			}
			lo := int(lod)
			hi := minInt(lo+1, len(safe.SourceMips[p.ID])-1)
			ft := lod - float64(lo)
			e0 := SampleBilinear(safe.SourceMips[p.ID][lo], 0, 0.5)
			e1 := SampleBilinear(safe.SourceMips[p.ID][hi], 0, 0.5)
			expected := lerpPremul(e0, e1, ft)
			if dist := colorDistance(actual, expected); dist > 24 {
				t.Fatalf("trilinear mismatch %s lod %.1f dist %.1f", p.ID, lod, dist)
			}
		}
	}
}

func TestRotatedUVRoundTrip(t *testing.T) {
	p := Placement{Frame: R(32, 48, 64, 128), Rotated: true}
	for _, tc := range [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {0.25, 0.75}} {
		au, av := SpriteToAtlasUV(p, 512, 512, tc[0], tc[1])
		u, v, ok := AtlasToSpriteUV(p, 512, 512, au, av)
		if !ok || absFloat(u-tc[0]) > 1e-9 || absFloat(v-tc[1]) > 1e-9 {
			t.Fatalf("UV round trip failed: want %v got %.4f,%.4f ok=%v", tc, u, v, ok)
		}
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestDownsampleBoxAveragesOpaquePixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	src.SetRGBA(1, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	src.SetRGBA(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	src.SetRGBA(1, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	dst := DownsampleBox(src)
	got := dst.RGBAAt(0, 0)
	if got.R < 120 || got.R > 135 || got.B < 120 || got.B > 135 || got.G != 0 {
		t.Fatalf("unexpected 2x2 average: %+v", got)
	}
}

func testSprites() []SpriteImage {
	specs := []struct {
		id string
		c  color.RGBA
	}{
		{"red", color.RGBA{R: 230, G: 40, B: 40, A: 255}},
		{"cyan", color.RGBA{R: 30, G: 220, B: 230, A: 255}},
		{"yellow", color.RGBA{R: 245, G: 220, B: 40, A: 255}},
		{"blue", color.RGBA{R: 50, G: 80, B: 240, A: 255}},
		{"lime", color.RGBA{R: 90, G: 230, B: 80, A: 255}},
		{"magenta", color.RGBA{R: 230, G: 70, B: 220, A: 255}},
	}
	out := make([]SpriteImage, 0, len(specs))
	for i, sp := range specs {
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				c := sp.c
				if (x/8+y/8)%2 == 0 {
					c = shade(sp.c, -35)
				}
				if x < 3 || y < 3 || x >= 61 || y >= 61 {
					c = contrasting(sp.c, i)
				}
				img.SetRGBA(x, y, c)
			}
		}
		out = append(out, SpriteImage{ID: sp.id, Image: img})
	}
	return out
}

func shade(c color.RGBA, delta int) color.RGBA {
	clamp := func(v int) uint8 { return uint8(maxInt(0, minInt(255, v+delta))) }
	return color.RGBA{R: clamp(int(c.R)), G: clamp(int(c.G)), B: clamp(int(c.B)), A: 255}
}

func contrasting(c color.RGBA, i int) color.RGBA {
	if i%2 == 0 {
		return color.RGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: 255}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}
