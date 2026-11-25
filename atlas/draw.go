package atlas

import (
	"image"
	"image/color"
	"strconv"
)

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
		return
	}
	if c.A == 255 {
		img.SetRGBA(x, y, c)
		return
	}
	old := img.RGBAAt(x, y)
	a := float64(c.A) / 255
	out := color.RGBA{
		R: uint8(float64(old.R)*(1-a) + float64(c.R)*a + 0.5),
		G: uint8(float64(old.G)*(1-a) + float64(c.G)*a + 0.5),
		B: uint8(float64(old.B)*(1-a) + float64(c.B)*a + 0.5),
		A: 255,
	}
	img.SetRGBA(x, y, out)
}

func drawRectOutline(img *image.RGBA, r Rect, c color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	for t := 0; t < thickness; t++ {
		top := R(r.X-t, r.Y-t, r.W+2*t, thickness)
		bottom := R(r.X-t, r.Y+r.H, r.W+2*t, thickness)
		left := R(r.X-t, r.Y-t, thickness, r.H+2*thickness)
		right := R(r.X+r.W, r.Y-t, thickness, r.H+2*thickness)
		fillRGBA(img, top, c)
		fillRGBA(img, bottom, c)
		fillRGBA(img, left, c)
		fillRGBA(img, right, c)
	}
}

func fillRGBA(img *image.RGBA, r Rect, c color.RGBA) {
	b := img.Bounds()
	r = r.Intersection(R(b.Min.X, b.Min.Y, b.Dx(), b.Dy()))
	for y := r.Y; y < r.MaxY(); y++ {
		for x := r.X; x < r.MaxX(); x++ {
			setPixel(img, x, y, c)
		}
	}
}

func drawMarker(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			d2 := dx*dx + dy*dy
			if d2 <= radius*radius && d2 >= (radius-2)*(radius-2) {
				setPixel(img, x, y, c)
			}
		}
	}
	fillRGBA(img, R(cx-1, cy-1, 3, 3), color.RGBA{R: 255, G: 240, B: 60, A: 255})
}

func dimImage(img *image.RGBA, amount float64) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			c.R = uint8(float64(c.R) * (1 - amount))
			c.G = uint8(float64(c.G) * (1 - amount))
			c.B = uint8(float64(c.B) * (1 - amount))
			img.SetRGBA(x, y, c)
		}
	}
}

var digitGlyphs = map[rune][5]string{
	'0': {"111", "101", "101", "101", "111"},
	'1': {"010", "110", "010", "010", "111"},
	'2': {"111", "001", "111", "100", "111"},
	'3': {"111", "001", "111", "001", "111"},
	'4': {"101", "101", "111", "001", "001"},
	'5': {"111", "100", "111", "001", "111"},
	'6': {"111", "100", "111", "101", "111"},
	'7': {"111", "001", "010", "010", "010"},
	'8': {"111", "101", "111", "101", "111"},
	'9': {"111", "101", "111", "001", "111"},
}

func drawNumberBadge(img *image.RGBA, x, y, number, scale int) {
	if scale < 1 {
		scale = 2
	}
	text := strconv.Itoa(number)
	width := len(text)*4*scale - scale
	fillRGBA(img, R(x-2*scale, y-2*scale, width+4*scale, 5*scale+4*scale),
		color.RGBA{R: 0, G: 0, B: 0, A: 210})
	for i, ch := range text {
		glyph, ok := digitGlyphs[ch]
		if !ok {
			continue
		}
		for row, line := range glyph {
			for col, on := range line {
				if on != '1' {
					continue
				}
				fillRGBA(img, R(x+i*4*scale+col*scale, y+row*scale, scale, scale),
					color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
}
