// Package hdr defines a scene-referred linear-light floating-point image
// and basic statistics over it. Values are NOT gamma encoded and are allowed
// to exceed 1.0 (highlight headroom).
package hdr

import "math"

// Image is a packed RGB float32 image. Pix holds R,G,B triplets row by row,
// top row first (the usual display orientation; PFM I/O flips as needed).
type Image struct {
	W, H int
	Pix  []float32
}

// NewImage allocates an RGB float image filled with zeros.
func NewImage(w, h int) *Image {
	return &Image{W: w, H: h, Pix: make([]float32, w*h*3)}
}

func (m *Image) index(x, y int) int { return (y*m.W + x) * 3 }

// At returns the linear RGB triplet at (x, y).
func (m *Image) At(x, y int) (r, g, b float32) {
	i := m.index(x, y)
	return m.Pix[i], m.Pix[i+1], m.Pix[i+2]
}

// Set writes the linear RGB triplet at (x, y).
func (m *Image) Set(x, y int, r, g, b float32) {
	i := m.index(x, y)
	m.Pix[i], m.Pix[i+1], m.Pix[i+2] = r, g, b
}

// Add adds light (in linear units) to the pixel at (x, y).
func (m *Image) Add(x, y int, r, g, b float32) {
	i := m.index(x, y)
	m.Pix[i] += r
	m.Pix[i+1] += g
	m.Pix[i+2] += b
}

// Stats summarizes radiometric content of the image.
type Stats struct {
	MinLum     float64
	MaxLum     float64
	MeanLum    float64
	PeakR      float64
	PeakG      float64
	PeakB      float64
	OverWhite  int // number of pixels whose luminance exceeds 1.0
	TotalPx    int
	MaxChannel float64 // largest single channel value in the scene
}

// luma709 computes Rec.709 luma from linear RGB (weights apply to linear light).
func luma709(r, g, b float64) float64 {
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// Compute collects min/max/mean luminance and highlight counts.
func (m *Image) Compute() Stats {
	s := Stats{MinLum: math.Inf(1), TotalPx: m.W * m.H}
	var sum float64
	for i := 0; i < len(m.Pix); i += 3 {
		r, g, b := float64(m.Pix[i]), float64(m.Pix[i+1]), float64(m.Pix[i+2])
		l := luma709(r, g, b)
		if l < s.MinLum {
			s.MinLum = l
		}
		if l > s.MaxLum {
			s.MaxLum = l
		}
		if r > s.PeakR {
			s.PeakR = r
		}
		if g > s.PeakG {
			s.PeakG = g
		}
		if b > s.PeakB {
			s.PeakB = b
		}
		if r > s.MaxChannel {
			s.MaxChannel = r
		}
		if g > s.MaxChannel {
			s.MaxChannel = g
		}
		if b > s.MaxChannel {
			s.MaxChannel = b
		}
		if l > 1.0 {
			s.OverWhite++
		}
		sum += l
	}
	if s.TotalPx > 0 {
		s.MeanLum = sum / float64(s.TotalPx)
	}
	return s
}

// Clone returns a deep copy.
func (m *Image) Clone() *Image {
	pix := make([]float32, len(m.Pix))
	copy(pix, m.Pix)
	return &Image{W: m.W, H: m.H, Pix: pix}
}
