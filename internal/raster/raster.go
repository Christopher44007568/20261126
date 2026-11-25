// Package raster is a tiny 8-bit RGB canvas used for annotated preview
// panels and comparison sheets. It draws rectangles only; glyph rendering
// lives in the font package.
package raster

// Image8 is a packed top-down RGB8 image.
type Image8 struct {
	W, H int
	Pix  []byte
}

// NewImage8 allocates a black RGB8 canvas.
func NewImage8(w, h int) *Image8 {
	return &Image8{W: w, H: h, Pix: make([]byte, w*h*3)}
}

func (m *Image8) idx(x, y int) int { return (y*m.W + x) * 3 }

// Set writes one pixel, ignoring coordinates outside the canvas.
func (m *Image8) Set(x, y int, r, g, b uint8) {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return
	}
	i := m.idx(x, y)
	m.Pix[i], m.Pix[i+1], m.Pix[i+2] = r, g, b
}

// FillRect paints a solid rectangle.
func (m *Image8) FillRect(x, y, w, h int, r, g, b uint8) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			m.Set(xx, yy, r, g, b)
		}
	}
}

// StrokeRect paints a one-pixel outline.
func (m *Image8) StrokeRect(x, y, w, h int, r, g, b uint8) {
	for xx := x; xx < x+w; xx++ {
		m.Set(xx, y, r, g, b)
		m.Set(xx, y+h-1, r, g, b)
	}
	for yy := y; yy < y+h; yy++ {
		m.Set(x, yy, r, g, b)
		m.Set(x+w-1, yy, r, g, b)
	}
}

// Blit copies an RGB8 image into the canvas at (dx,dy).
func (m *Image8) Blit(dx, dy int, src *Image8) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			si := (y*src.W + x) * 3
			m.Set(dx+x, dy+y, src.Pix[si], src.Pix[si+1], src.Pix[si+2])
		}
	}
}

// PasteHDR writes an HDR-derived RGB8 buffer of the same dimensions.
func (m *Image8) PasteHDR(dx, dy int, rgb []byte, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			si := (y*w + x) * 3
			m.Set(dx+x, dy+y, rgb[si], rgb[si+1], rgb[si+2])
		}
	}
}
