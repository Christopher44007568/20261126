package atlas

import (
	"image"
	"image/color"
	"image/draw"
)

var (
	colorCyan    = color.RGBA{R: 70, G: 230, B: 255, A: 255}
	colorYellow  = color.RGBA{R: 255, G: 214, B: 64, A: 255}
	colorRed     = color.RGBA{R: 255, G: 45, B: 70, A: 255}
	colorGreen   = color.RGBA{R: 70, G: 255, B: 140, A: 255}
	colorPanel   = color.RGBA{R: 18, G: 22, B: 34, A: 255}
	colorPanelLn = color.RGBA{R: 52, G: 62, B: 86, A: 255}
)

// RenderLayout draws the packed atlas with yellow cell/padding bounds and
// cyan sprite frame/UV bounds.
func RenderLayout(b *BuiltAtlas) *image.RGBA {
	img := cloneAsRGBA(b.Base)
	for i, p := range b.Layout.Placements {
		drawRectOutline(img, p.Cell, colorYellow, 1)
		drawRectOutline(img, p.Frame, colorCyan, 1)
		drawNumberBadge(img, p.Frame.X+5, p.Frame.Y+5, i+1, 2)
	}
	drawRectOutline(img, R(0, 0, b.Layout.Width, b.Layout.Height), colorPanelLn, 2)
	return img
}

// RenderValidationOverlay dims the atlas and marks sampling violations red.
// A passing report gets green frame outlines.
func RenderValidationOverlay(b *BuiltAtlas, report ValidationReport) *image.RGBA {
	img := cloneAsRGBA(b.Base)
	dimImage(img, 0.48)

	if report.OK {
		for _, p := range b.Layout.Placements {
			drawRectOutline(img, p.Frame, colorGreen, 2)
		}
		drawRectOutline(img, R(2, 2, b.Layout.Width-4, b.Layout.Height-4), colorGreen, 4)
		return img
	}

	for _, v := range report.Violations {
		x := int(v.AtlasPX + 0.5)
		y := int(v.AtlasPY + 0.5)
		radius := 5 + int(v.MipLevel)
		drawMarker(img, x, y, radius, colorRed)
	}
	for _, p := range b.Layout.Placements {
		drawRectOutline(img, p.Frame, colorRed, 1)
	}
	return img
}

// RenderSideBySide places two same-sized diagnostic images next to each other.
func RenderSideBySide(left, right *image.RGBA, gap int) *image.RGBA {
	if gap <= 0 {
		gap = 12
	}
	w := left.Bounds().Dx() + gap + right.Bounds().Dx()
	h := maxInt(left.Bounds().Dy(), right.Bounds().Dy())
	out := NewRGBA(w, h, colorPanel)
	draw.Draw(out, left.Bounds(), left, left.Bounds().Min, draw.Src)
	rightRect := image.Rect(gap+left.Bounds().Dx(), 0, gap+left.Bounds().Dx()+right.Bounds().Dx(), right.Bounds().Dy())
	draw.Draw(out, rightRect, right, right.Bounds().Min, draw.Src)
	drawRectOutline(out, R(0, 0, w, h), colorPanelLn, 2)
	return out
}

// RenderMipContactSheet shows every mip and its scaled padding rectangle.
func RenderMipContactSheet(b *BuiltAtlas) *image.RGBA {
	margin, gap := 18, 28
	contentH := 0
	width := margin
	for _, mip := range b.Mips {
		width += mip.Bounds().Dx() + gap
		contentH = maxInt(contentH, mip.Bounds().Dy())
	}
	width += margin - gap
	height := margin + 34 + contentH + margin
	out := NewRGBA(width, height, colorPanel)

	x := margin
	top := margin + 30
	for level, mip := range b.Mips {
		r := image.Rect(x, top, x+mip.Bounds().Dx(), top+mip.Bounds().Dy())
		draw.Draw(out, r, mip, mip.Bounds().Min, draw.Src)
		drawRectOutline(out, R(x, top, mip.Bounds().Dx(), mip.Bounds().Dy()), colorPanelLn, 1)
		drawNumberBadge(out, x+4, margin+4, level, 2)

		levelPad := b.Layout.PackPadding >> uint(level)
		for _, p := range b.Layout.Placements {
			frame := ScaledRect(p.Frame, level).Translate(x, top)
			drawRectOutline(out, frame, colorCyan, 1)
			if levelPad > 0 {
				drawRectOutline(out, frame.Expand(levelPad), colorYellow, 1)
			}
		}
		x += mip.Bounds().Dx() + gap
	}
	return out
}
