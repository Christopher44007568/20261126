// Package sim renders a deterministic camera scene for exercising the
// autofocus engine without a camera or graphics SDK.
package sim

import (
	"image"
	"image/color"
	"math"
)

// Scene describes a moving textured subject in front of a static background.
type Scene struct {
	Width, Height   int
	SubjectDepth    int
	BackgroundDepth int
	SubjectWidth    int
	SubjectHeight   int
	EnterFrame      int
	SettledFrame    int
	BlurPerStep     float64
}

// DefaultScene returns a 320x240 scene whose subject reaches center-stage on
// frame 26 and is sharpest at lens position 44.
func DefaultScene() Scene {
	return Scene{
		Width:           320,
		Height:          240,
		SubjectDepth:    44,
		BackgroundDepth: 6,
		SubjectWidth:    110,
		SubjectHeight:   170,
		EnterFrame:      5,
		SettledFrame:    26,
		BlurPerStep:     0.32,
	}
}

// Render creates frame at the requested integer lens position.
func (s Scene) Render(frame, lensPosition int) *image.RGBA {
	background := s.background()
	background = boxBlur(background, blurRadius(lensPosition, s.BackgroundDepth, s.BlurPerStep))

	subject := s.subject(frame)
	subject = boxBlur(subject, blurRadius(lensPosition, s.SubjectDepth, s.BlurPerStep))
	compositeOpaque(background, subject, s.subjectOffsetX(frame), s.subjectOffsetY())
	return background
}

func (s Scene) subjectOffsetX(frame int) int {
	center := (s.Width - s.SubjectWidth) / 2
	start := -s.SubjectWidth - 4
	if frame <= s.EnterFrame {
		return start
	}
	if frame >= s.SettledFrame {
		return center
	}
	t := float64(frame-s.EnterFrame) / float64(s.SettledFrame-s.EnterFrame)
	t = t * t * (3 - 2*t)
	return start + int(math.Round(t*float64(center-start)))
}

func (s Scene) subjectOffsetY() int {
	return (s.Height - s.SubjectHeight) / 2
}

func (s Scene) background() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, s.Width, s.Height))
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			band := (x/23 + y/17) & 1
			v := uint8(46 + band*7)
			if (x+y)%97 < 2 {
				v -= 9
			}
			img.SetRGBA(x, y, color.RGBA{R: v - 8, G: v, B: v + 9, A: 255})
		}
	}
	return img
}

func (s Scene) subject(frame int) *image.RGBA {
	w := s.SubjectWidth + 24
	h := s.SubjectHeight + 24
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2, float64(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (float64(x) - cx) / (float64(s.SubjectWidth) / 2)
			dy := (float64(y) - cy) / (float64(s.SubjectHeight) / 2)
			if dx*dx+dy*dy > 1.25 {
				continue
			}
			checker := ((x+frame/3)/9 + (y-frame/4)/11) & 1
			v := uint8(78)
			if checker == 0 {
				v = 218
			}
			img.SetRGBA(x, y, color.RGBA{R: v, G: v - 5, B: v - 18, A: 255})
		}
	}
	return img
}

func blurRadius(lens, depth int, perStep float64) int {
	return int(math.Round(math.Abs(float64(lens-depth)) * perStep))
}

func compositeOpaque(dst, src *image.RGBA, ox, oy int) {
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			dst.SetRGBA(ox+x, oy+y, c)
		}
	}
}

func boxBlur(src *image.RGBA, radius int) *image.RGBA {
	if radius <= 0 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	tmp := image.NewRGBA(src.Bounds())
	out := image.NewRGBA(src.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			blurPixel(src, tmp, x, y, radius, true)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			blurPixel(tmp, out, x, y, radius, false)
		}
	}
	return out
}

func blurPixel(src, dst *image.RGBA, x, y, radius int, horizontal bool) {
	var r, g, b, a, n int
	for d := -radius; d <= radius; d++ {
		sx, sy := x, y
		if horizontal {
			sx += d
		} else {
			sy += d
		}
		if !image.Pt(sx, sy).In(src.Bounds()) {
			continue
		}
		c := src.RGBAAt(sx, sy)
		r += int(c.R)
		g += int(c.G)
		b += int(c.B)
		a += int(c.A)
		n++
	}
	if n == 0 {
		return
	}
	dst.SetRGBA(x, y, color.RGBA{
		R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n),
	})
}
