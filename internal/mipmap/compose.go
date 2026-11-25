package mipmap

import (
	"image"
	"image/color"
	"image/draw"
)

const (
	sheetPadding = 18
	cellSize     = 168
	labelHeight  = 24
)

func ScaleNearest(src image.Image, factor int) *image.RGBA {
	if factor < 1 {
		factor = 1
	}
	bounds := src.Bounds()
	width := bounds.Dx() * factor
	height := bounds.Dy() * factor
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			c := src.At(bounds.Min.X+x, bounds.Min.Y+y)
			for sy := 0; sy < factor; sy++ {
				for sx := 0; sx < factor; sx++ {
					dst.Set(x*factor+sx, y*factor+sy, c)
				}
			}
		}
	}
	return dst
}

// ResizeNearest maps output texel centers back to the source without filtering.
// It supports both integer magnification and fractional montage shrink-fit.
func ResizeNearest(src image.Image, width, height int) *image.RGBA {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + x*bounds.Dx()/width
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func integerFit(srcWidth, srcHeight, box int) int {
	factor := box / maxInt(1, maxInt(srcWidth, srcHeight))
	if factor < 1 {
		factor = 1
	}
	return factor
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fillRect(dst *image.RGBA, rect image.Rectangle, c color.Color) {
	draw.Draw(dst, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func fillBackground(dst *image.RGBA, c color.Color) {
	fillRect(dst, dst.Bounds(), c)
}

// PeriodicCornerCrop crops a 2*radius square around the tile corner (0,0).
// The visible seam runs between radius-1 and radius in both axes.
func PeriodicCornerCrop(img *RGBImage, radius int) *RGBImage {
	if radius < 1 {
		radius = 1
	}
	size := radius * 2
	crop := NewRGBImage(size, size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r, g, b := img.WrapRGB(x-radius, y-radius)
			crop.Set(x, y, r, g, b)
		}
	}
	return crop
}

func levelSeamRadius(level *RGBImage) int {
	minimum := level.Width
	if level.Height < minimum {
		minimum = level.Height
	}
	radius := 8
	if minimum < radius*2 {
		radius = maxInt(1, minimum/2)
	}
	return radius
}

func drawCenteredScaled(dst *image.RGBA, src image.Image, cell image.Rectangle) {
	bounds := src.Bounds()
	factor := integerFit(bounds.Dx(), bounds.Dy(), cell.Dx()-12)
	scaled := ScaleNearest(src, factor)
	x := cell.Min.X + (cell.Dx()-scaled.Bounds().Dx())/2
	y := cell.Min.Y + (cell.Dy()-scaled.Bounds().Dy())/2
	draw.Draw(dst, scaled.Bounds().Add(image.Pt(x, y)), scaled, image.Point{}, draw.Over)
}

func drawSeamGuides(dst *image.RGBA, cell image.Rectangle, srcWidth, srcHeight, factor int) {
	scaledWidth := srcWidth * factor
	scaledHeight := srcHeight * factor
	x0 := cell.Min.X + (cell.Dx()-scaledWidth)/2
	y0 := cell.Min.Y + (cell.Dy()-scaledHeight)/2
	guide := color.RGBA{R: 255, G: 48, B: 168, A: 210}

	verticalX := x0 + (srcWidth/2)*factor - 1
	if verticalX >= x0 {
		fillRect(dst, image.Rect(verticalX, y0, verticalX+1, y0+scaledHeight), guide)
	}
	horizontalY := y0 + (srcHeight/2)*factor - 1
	if horizontalY >= y0 {
		fillRect(dst, image.Rect(x0, horizontalY, x0+scaledWidth, horizontalY+1), guide)
	}
}
