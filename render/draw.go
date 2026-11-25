// Package render draws timing contact sheets with only the Go standard
// library and the local zero-dependency raster canvas.
package render

import (
	"math"

	"animationtiming/raster"
	"animationtiming/timeline"
)

var (
	ink       = raster.Hex(0x172033)
	softInk   = raster.Hex(0x5d6b80)
	paper     = raster.Hex(0xe8edf4)
	card      = raster.Hex(0xfbfdff)
	stage     = raster.Hex(0xf4f8fb)
	blue      = raster.Hex(0x2f80ed)
	blueDark  = raster.Hex(0x1b5fb8)
	green     = raster.Hex(0x16a36a)
	greenPale = raster.Hex(0xdff5e9)
	amber     = raster.Hex(0xd99400)
	amberPale = raster.Hex(0xffedc7)
	red       = raster.Hex(0xd33b3b)
	redPale   = raster.Hex(0xfbdada)
	gold      = raster.Hex(0xf2b705)
	purple    = raster.Hex(0x8b5cf6)
	purpleDk  = raster.Hex(0x6d3fc4)
	teal      = raster.Hex(0x20b486)
	white     = raster.Hex(0xffffff)
	grayLine  = raster.Hex(0xc8d2df)
)

type point struct {
	X, Y float64
}

func polar(from point, angleDeg, length float64) point {
	rad := angleDeg * math.Pi / 180
	return point{
		X: from.X + math.Sin(rad)*length,
		Y: from.Y + math.Cos(rad)*length,
	}
}

func lineF(c *raster.Canvas, a, b point, width int, col raster.Color) {
	c.LineWidth(int(math.Round(a.X)), int(math.Round(a.Y)),
		int(math.Round(b.X)), int(math.Round(b.Y)), width, col)
}

func discF(c *raster.Canvas, p point, r int, col raster.Color) {
	c.Disc(int(math.Round(p.X)), int(math.Round(p.Y)), r, col)
}

func ellipse(c *raster.Canvas, cx, cy, rx, ry int, col raster.Color) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			v := float64(x*x)/float64(rx*rx) + float64(y*y)/float64(ry*ry)
			if v <= 1 {
				c.Set(cx+x, cy+y, col)
			}
		}
	}
}

func diamond(c *raster.Canvas, cx, cy, r int, col raster.Color) {
	for y := -r; y <= r; y++ {
		span := r - absInt(y)
		if span < 0 {
			continue
		}
		c.FillRect(cx-span, cy+y, span*2+1, 1, col)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func statusColors(status timeline.RangeStatus) (bg, badge, border raster.Color, label string) {
	switch status {
	case timeline.StatusBefore:
		return amberPale, amber, amber, "< BEFORE RANGE"
	case timeline.StatusAfter:
		return redPale, red, red, "AFTER RANGE >"
	case timeline.StatusStatic:
		return raster.Hex(0xf1f3f6), softInk, grayLine, "STATIC"
	default:
		return greenPale, green, raster.Hex(0xd0a52c), "IN RANGE"
	}
}
