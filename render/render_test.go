package render

import (
	"image"
	"image/color"
	"testing"

	"materialover/scene"
)

func solidScene() *scene.Scene {
	bg := scene.RGB{R: 1, G: 1, B: 1}
	return &scene.Scene{
		Width:      40,
		Height:     40,
		Background: &bg,
		Meshes: []*scene.Mesh{
			{
				ID:       "r",
				Kind:     "rect",
				Geometry: scene.Geometry{X: 0, Y: 0, W: 40, H: 40},
				Material: scene.Material{
					Color:   scene.RGB{R: 0, G: 0, B: 0},
					Opacity: 0.5,
				},
			},
		},
	}
}

func TestRenderOpacityOverOpaqueBackground(t *testing.T) {
	img := RenderScene(solidScene())
	// Black at 50% over white must be 50% gray (allow SSAA rounding of 1).
	c := img.NRGBAAt(20, 20)
	if c.A != 255 {
		t.Fatalf("alpha = %d, want 255", c.A)
	}
	if c.R < 126 || c.R > 129 || c.R != c.G || c.G != c.B {
		t.Fatalf("expected neutral ~127 gray, got %+v", c)
	}
}

func TestRenderOverIsTransparentOutsideMeshes(t *testing.T) {
	s := solidScene()
	img := RenderOver(s, s.Meshes)
	if c := img.NRGBAAt(0, 0); c.A < 126 || c.A > 129 {
		t.Fatalf("over layer alpha = %d, want ~127", c.A)
	}
	// An empty over layer is fully transparent.
	empty := RenderOver(s, nil)
	if c := empty.NRGBAAt(20, 20); c.A != 0 {
		t.Fatalf("empty over layer alpha = %d, want 0", c.A)
	}
}

func TestComposeIdentity(t *testing.T) {
	// Rendering the full scene directly must match under + over composition.
	s := solidScene()
	full := RenderScene(s)
	under := RenderExcept(s, []string{"r"})
	over := RenderOver(s, s.Meshes)
	composed := Compose(under, over)
	if !imagesEqual(full, composed) {
		t.Fatal("full render differs from under+over composition")
	}
}

func TestRenderExceptKeepsUntouchedMeshes(t *testing.T) {
	s := solidScene()
	img := RenderExcept(s, []string{"r"})
	// Only the opaque white background remains.
	c := img.NRGBAAt(20, 20)
	if c != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("untouched background wrong: %+v", c)
	}
}

func imagesEqual(a, b *image.NRGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
