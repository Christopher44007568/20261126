package viz

import (
	"nsmigrate/raster"
	"nsmigrate/scene"
)

type rect struct{ x, y, w, h int }

func (r rect) cx() int { return r.x + r.w/2 }
func (r rect) cy() int { return r.y + r.h/2 }

type tform struct {
	scale    float64
	cx, cy   float64 // screen center
	wcx, wcy float64
}

func (t tform) X(wx float64) int { return int(t.cx + (wx-t.wcx)*t.scale) }
func (t tform) Y(wy float64) int { return int(t.cy + (wy-t.wcy)*t.scale) }

func primRect(t tform, p *scene.Prim) rect {
	hw := int(p.W*t.scale/2 + 0.5)
	hh := int(p.H*t.scale/2 + 0.5)
	x := t.X(p.X) - hw
	y := t.Y(p.Y) - hh
	return rect{x: x, y: y, w: hw * 2, h: hh * 2}
}

func shade(col raster.Color, f float64) raster.Color {
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return raster.Color{
		R: clamp(float64(col.R) * f),
		G: clamp(float64(col.G) * f),
		B: clamp(float64(col.B) * f),
	}
}

func textColorFor(col raster.Color) raster.Color {
	lum := 0.299*float64(col.R) + 0.587*float64(col.G) + 0.114*float64(col.B)
	if lum > 150 {
		return palette.ink
	}
	return raster.Hex(0xFFFFFF)
}

// tag draws a centered white label chip with dark text.
func (p *Poster) tag(text string, cx, topY int) {
	c := p.Canvas
	tw := raster.TextWidth(text, 2)
	w := tw + 14
	x := cx - w/2
	c.FillRect(x, topY, w, 22, raster.Hex(0xFFFFFF))
	c.FrameRect(x, topY, w, 22, 1, palette.border)
	c.Text(text, x+7, topY+3, 2, palette.ink)
}

func (p *Poster) drawPrimShape(pr *scene.Prim, r rect, fill raster.Color) {
	c := p.Canvas
	cx, cy := r.cx(), r.cy()

	switch pr.Shape {
	case scene.ShapeGroup:
		return
	case scene.ShapeBox:
		c.FillRect(r.x+5, r.y+6, r.w, r.h, raster.Hex(0xC3CDDC))
		c.FillRect(r.x, r.y, r.w, r.h, fill)
		c.FillRect(r.x, r.y+r.h-12, r.w, 12, shade(fill, 0.72))
		c.FrameRect(r.x, r.y, r.w, r.h, 2, shade(fill, 0.5))
		p.tag(pr.Tag, cx, r.y+r.h+16)
	case scene.ShapeSphere:
		rad := r.w / 2
		if r.h/2 < rad {
			rad = r.h / 2
		}
		c.Disc(cx+4, cy+6, rad, raster.Hex(0xC3CDDC))
		c.Disc(cx, cy, rad, fill)
		c.Circle(cx, cy, rad, 2, shade(fill, 0.5))
		c.Disc(cx-rad/3, cy-rad/3, rad/4, shade(fill, 1.25))
		p.tag(pr.Tag, cx, cy+rad+8)
	case scene.ShapeComponent:
		c.FillRect(r.x+5, r.y+6, r.w, r.h, raster.Hex(0xC3CDDC))
		c.FillRect(r.x, r.y, r.w, r.h, shade(fill, 0.9))
		c.FrameRect(r.x, r.y, r.w, r.h, 3, raster.Hex(0xFFFFFF))
		c.FrameRect(r.x+5, r.y+5, r.w-10, r.h-10, 2, shade(fill, 0.55))
		tw := raster.TextWidth(pr.Tag, 2)
		c.Text(pr.Tag, cx-tw/2, cy-8, 2, raster.Hex(0xFFFFFF))
		sub := "REF SOURCE"
		sw := raster.TextWidth(sub, 1)
		c.Text(sub, cx-sw/2, cy+12, 1, raster.Hex(0xDDF5E6))
	case scene.ShapeCamera:
		bodyW, bodyH := r.w+14, r.h+8
		bx, by := cx-bodyW/2, cy-bodyH/2
		c.FillRect(bx+4, by+5, bodyW, bodyH, raster.Hex(0xC3CDDC))
		c.FillRect(bx, by, bodyW, bodyH, fill)
		c.FrameRect(bx, by, bodyW, bodyH, 2, shade(fill, 0.5))
		c.FillRect(bx+10, by-8, 18, 10, fill)
		lr := bodyH/2 - 6
		c.Disc(cx+8, cy+2, lr+3, raster.Hex(0x0B1220))
		c.Disc(cx+8, cy+2, lr, raster.Hex(0x7DD3FC))
		c.Disc(cx+8-lr/3, cy+2-lr/3, lr/3, raster.Hex(0xE0F2FE))
		p.tag(pr.Tag, cx, by+bodyH+8)
	case scene.ShapeTarget:
		rad := r.w / 2
		c.Disc(cx, cy, rad+3, raster.Hex(0xFFFFFF))
		c.Circle(cx, cy, rad+2, 3, palette.ink)
		c.Circle(cx, cy, rad-3, 2, shade(cvt(pr.Color), 0.85))
		c.LineWidth(cx-rad-8, cy, cx-rad+2, cy, 3, palette.ink)
		c.LineWidth(cx+rad-2, cy, cx+rad+8, cy, 3, palette.ink)
		c.LineWidth(cx, cy-rad-8, cx, cy-rad+2, 3, palette.ink)
		c.LineWidth(cx, cy+rad-2, cx, cy+rad+8, 3, palette.ink)
		c.Disc(cx, cy, 3, palette.bad)
		p.tag(pr.Tag, cx, cy+rad+12)
	}
}
