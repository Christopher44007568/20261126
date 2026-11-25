package shutter

import "math"

// buildVectors constructs the camera-relative motion vector field.
//
// For every object pixel at its mid-exposure projection we compute where the
// same material point projects at shutter open and at shutter close. Both
// projections pass through the mid-exposure camera, so camera ego-motion is
// removed: a world-static point always maps to itself and therefore keeps a
// zero vector. The vector equals close-open displacement in pixels.
func buildVectors(req FrameRequest, midCam Pose) *VectorField {
	field := &VectorField{
		Width:  req.Width,
		Height: req.Height,
		X:      make([]float64, req.Width*req.Height),
		Y:      make([]float64, req.Width*req.Height),
		Mask:   make([]bool, req.Width*req.Height),
	}

	openT, closeT, midT := req.OpenT, req.CloseT, req.MidTime()

	for _, obj := range req.Objects {
		openPose := RelativeTo(midCam, obj.PoseAt(openT))
		closePose := RelativeTo(midCam, obj.PoseAt(closeT))
		midPose := RelativeTo(midCam, obj.PoseAt(midT))

		// A world-static or perfectly co-moving object occupies the same
		// place in the mid-exposure camera frame at both shutter edges.
		if posesWithin(openPose, closePose, 1e-9) {
			continue
		}

		for _, shape := range obj.Shapes {
			rasterMotion(field, shape, openPose, closePose, midPose, req.Projection)
		}
	}
	return field
}

// rasterMotion stamps per-pixel vectors for one shape by transforming its
// local vertices at the three key poses and using point-in-poly tests.
func rasterMotion(field *VectorField, shape Polygon, openPose, closePose, midPose Pose, p OrthoProjection) {
	midPts := make([]Vec2, len(shape.Vertices))
	for i, v := range shape.Vertices {
		px, py := p.Project(midPose.Point(v), field.Width, field.Height)
		midPts[i] = Vec2{X: px, Y: py}
	}

	minX, minY, maxX, maxY := bounds(midPts)
	x0 := maxInt(0, minX)
	y0 := maxInt(0, minY)
	x1 := minInt(field.Width-1, maxX)
	y1 := minInt(field.Height-1, maxY)

	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			sample := Vec2{X: float64(px) + 0.5, Y: float64(py) + 0.5}
			if !pointInPolygon(sample, midPts) {
				continue
			}
			// Invert the mid pose to recover the local point under this
			// pixel, then project it at the shutter edges.
			midInv := midPose.Inverse()
			local := midInv.Point(p.unproject(sample, field.Width, field.Height))
			ox, oy := p.Project(openPose.Point(local), field.Width, field.Height)
			cx, cy := p.Project(closePose.Point(local), field.Width, field.Height)
			i := py*field.Width + px
			field.X[i] = cx - ox
			field.Y[i] = cy - oy
			field.Mask[i] = true
		}
	}
}

// unproject inverts Project: pixel coordinates back to camera-frame units.
func (p OrthoProjection) unproject(px Vec2, width, height int) Vec2 {
	dx := (px.X - float64(width)*0.5) / p.PixelsPerUnit
	dy := (float64(height)*0.5 - px.Y) / p.PixelsPerUnit
	return p.Center.Add(Vec2{X: dx, Y: dy})
}

func bounds(pts []Vec2) (minX, minY, maxX, maxY int) {
	inf := 1 << 30
	minX, minY = inf, inf
	maxX, maxY = -inf, -inf
	for _, pt := range pts {
		minX = minInt(minX, int(math.Floor(pt.X)))
		maxX = maxInt(maxX, int(math.Floor(pt.X)))
		minY = minInt(minY, int(math.Floor(pt.Y)))
		maxY = maxInt(maxY, int(math.Floor(pt.Y)))
	}
	return
}

func posesWithin(a, b Pose, eps float64) bool {
	return abs(a.Pos.X-b.Pos.X) < eps &&
		abs(a.Pos.Y-b.Pos.Y) < eps &&
		abs(a.Rot-b.Rot) < eps
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
