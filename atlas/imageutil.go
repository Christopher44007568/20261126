package atlas

import (
	"image"
	"image/color"
)

func ToRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}

func NewRGBA(w, h int, bg color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if bg != nil {
		fillRect(img, img.Rect, bg)
	}
	return img
}

func fillRect(dst *image.RGBA, r image.Rectangle, c color.Color) {
	r = r.Intersect(dst.Rect)
	cc := color.RGBAModel.Convert(c).(color.RGBA)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, cc)
		}
	}
}

// ExtrudeRect fills every pixel in target outside srcRect with the nearest
// edge/corner texel from srcRect. This is the standard gutter dilation that
// prevents bilinear taps from reading a neighboring sprite.
func ExtrudeRect(dst *image.RGBA, srcRect Rect, target Rect) int {
	bounds := dst.Bounds()
	target = target.Clip(R(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()))
	count := 0
	for y := target.Y; y < target.MaxY(); y++ {
		for x := target.X; x < target.MaxX(); x++ {
			if srcRect.ContainsPoint(x, y) {
				continue
			}
			sx := clampInt(x, srcRect.X, srcRect.MaxX()-1)
			sy := clampInt(y, srcRect.Y, srcRect.MaxY()-1)
			dst.SetRGBA(x, y, dst.RGBAAt(sx, sy))
			count++
		}
	}
	return count
}

// ExtrudePlacement expands a frame by pad texels on every side.
func ExtrudePlacement(dst *image.RGBA, p Placement, pad int) int {
	if pad <= 0 {
		return 0
	}
	return ExtrudeRect(dst, p.Frame, p.Frame.Expand(pad))
}

// RotateCW returns a 90-degree clockwise copy.
func RotateCW(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(b.Min.X+x, b.Min.Y+y)
			dx := h - 1 - y
			dy := x
			dst.SetRGBA(dx, dy, c)
		}
	}
	return dst
}

func copyRectNearest(dst *image.RGBA, src *image.RGBA, from Rect, to Rect) {
	for y := 0; y < to.H; y++ {
		sy := clampInt(y, 0, from.H-1)
		dy := to.Y + y
		for x := 0; x < to.W; x++ {
			sx := clampInt(x, 0, from.W-1)
			dx := to.X + x
			dst.SetRGBA(dx, dy, src.RGBAAt(from.X+sx, from.Y+sy))
		}
	}
}
