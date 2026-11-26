package shadow

import (
	"image"
	"image/color"
)

// ImageSize is the square resolution of every shadow map.
const ImageSize = 256

// View bounds in receiver-plane world units (x horizontal, z vertical).
const (
	viewMinX = -8
	viewMaxX = 8
	viewMinZ = -6
	viewMaxZ = 6
)

// RenderConfig holds one row of the three-image comparison.
type RenderConfig struct {
	Flag RayFlag
	Name string
}

// RenderConfigs lists the shadow maps in output order.
var RenderConfigs = []RenderConfig{
	{BackfaceCull, "backface_cull"},
	{DoubleSidedHit, "double_sided"},
	{ReflectionClosure, "reflection_closure"},
}

func receiverPoint(px, py int) Vec3 {
	fx := float64(px) / float64(ImageSize-1)
	fy := float64(py) / float64(ImageSize-1)
	x := viewMinX + fx*(viewMaxX-viewMinX)
	// Screen rows grow downward; world z grows upward on the top view.
	z := viewMaxZ - fy*(viewMaxZ-viewMinZ)
	return Vec3{X: x, Y: ReceiverY, Z: z}
}

func toByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

// Render traces a full shadow map and returns an RGBA image.
// Grayscale encodes remaining light energy; saturated overlays encode the
// extra/ignored/merged hits produced by the active ray flag.
func (s *Scene) Render(flag RayFlag) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, ImageSize, ImageSize))
	for py := 0; py < ImageSize; py++ {
		for px := 0; px < ImageSize; px++ {
			point := receiverPoint(px, py)
			eval := s.Evaluate(ShadowRay(point), flag)
			img.SetNRGBA(px, py, shade(eval))
		}
	}
	return img
}

func shade(eval Eval) color.NRGBA {
	lum := eval.Energy
	base := toByte(lum)
	switch eval.Tag {
	case TagExtraOccluder:
		// Red: a back face newly accepted as an occluder.
		return color.NRGBA{maxByte(base, 220), toByte(lum * 0.25), toByte(lum * 0.25), 255}
	case TagIgnoredBackface:
		// Blue: a geometric back-face hit that must not cast a solid shadow.
		return color.NRGBA{toByte(lum * 0.25), toByte(lum * 0.45), maxByte(base, 210), 255}
	case TagClosureExit:
		// Amber: a real exit hit whose energy was merged with the entry.
		return color.NRGBA{maxByte(base, 225), maxByte(toByte(lum*0.6), 150), toByte(lum * 0.2), 255}
	default:
		return color.NRGBA{base, base, base, 255}
	}
}

func maxByte(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
