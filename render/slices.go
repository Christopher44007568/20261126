package render

import (
	"image"
	"image/color"

	"luttool/lut"
)

// SliceGrid renders every lattice slice (one tile per blue grid node,
// blue increasing left-to-right, top-to-bottom) followed by one
// high-resolution slice at a fractional blue value, which exercises
// the trilinear interpolation between lattice planes.
func SliceGrid(l *lut.LUT, tileRes, interpRes int) *image.NRGBA {
	n := l.Size
	tiles := make([]*image.NRGBA, 0, n+1)
	for bi := 0; bi < n; bi++ {
		blue := l.Min[2] + (l.Max[2]-l.Min[2])*float64(bi)/float64(n-1)
		tiles = append(tiles, SliceAt(l, blue, tileRes))
	}
	mid := (l.Min[2] + l.Max[2]) / 2
	tiles = append(tiles, SliceAt(l, mid, interpRes))

	cols := 1
	for cols*cols < len(tiles) {
		cols++
	}
	rows := (len(tiles) + cols - 1) / cols
	gap := 4
	w := cols*interpRes + (cols-1)*gap
	h := rows*interpRes + (rows-1)*gap
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	bg := color.NRGBA{16, 16, 16, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(x, y, bg)
		}
	}
	for i, t := range tiles {
		cx := (i % cols) * (interpRes + gap)
		cy := (i / cols) * (interpRes + gap)
		scale := float64(t.Bounds().Dx()-1) / float64(interpRes-1)
		for y := 0; y < interpRes; y++ {
			for x := 0; x < interpRes; x++ {
				sx := int(float64(x) * scale)
				sy := int(float64(y) * scale)
				out.SetNRGBA(cx+x, cy+y, t.NRGBAAt(sx, sy))
			}
		}
	}
	return out
}
