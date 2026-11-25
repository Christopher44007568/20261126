package synth

import (
	"deshake/internal/geom"
)

// Capture renders one rolling-shutter frame. Every row is sampled with the
// camera pose at that row's readout instant, so fast angular motion appears
// as inter-row shear (jelly) rather than a global perspective transform.
func (m *Motion) Capture(sc *Scene, fi int) *Frame {
	w, h := m.Cam.W, m.Cam.H
	img := NewRGB(w, h)
	rowH := make([]geom.Mat3, h)

	for y := 0; y < h; y++ {
		rf := geom.RowReadTime(float64(y), h)
		t := m.ReadTime(fi, rf)
		pose := m.PoseAt(t)
		inv := m.Cam.PixelToWorld(pose)
		rowH[y] = m.Cam.WorldToPixel(pose)
		for x := 0; x < w; x++ {
			wx, wy := inv.Apply(float64(x)+0.5, float64(y)+0.5)
			if !sc.Inside(wx, wy) {
				continue // undefined pixels stay zero; pipeline marks holes itself
			}
			r, g, b := sampleScene(sc, wx, wy)
			img.Set(x, y, r, g, b)
		}
	}

	mid := m.PoseAt(m.ReadTime(fi, 0.5))
	return &Frame{
		Index:   fi,
		W:       w,
		H:       h,
		Img:     img,
		MidPose: mid,
		RefPose: m.RefPoseAt(float64(fi) + 0.5),
		RowH:    rowH,
		MidH:    m.Cam.WorldToPixel(mid),
	}
}

// sampleScene performs bilinear interpolation of the procedural scene.
func sampleScene(sc *Scene, x, y float64) (r, g, b float64) {
	x0 := floorI(x)
	y0 := floorI(y)
	fx := x - float64(x0)
	fy := y - float64(y0)

	var acc [3]float64
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			xx, yy := x0+dx, y0+dy
			if !sc.Inside(float64(xx), float64(yy)) {
				return sc.Sample(x, y)
			}
			w := (1 - fx + float64(dx)*(2*fx-1)) * (1 - fy + float64(dy)*(2*fy-1))
			sr, sg, sb := sc.Sample(float64(xx)+0.5, float64(yy)+0.5)
			acc[0] += w * sr
			acc[1] += w * sg
			acc[2] += w * sb
		}
	}
	return acc[0], acc[1], acc[2]
}

func floorI(v float64) int {
	i := int(v)
	if v < float64(i) {
		i--
	}
	return i
}
