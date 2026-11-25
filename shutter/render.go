package shutter

// VectorField holds per-pixel screen-space motion vectors in pixels. The
// vector points from the shutter-open projection to the shutter-close
// projection, expressed in the mid-exposure canonical image. Static
// background pixels are always (0,0).
type VectorField struct {
	Width, Height int
	X, Y          []float64
	// Mask is true where a moving object contributed a vector.
	Mask []bool
}

// FrameResult is the output of one exposure.
type FrameResult struct {
	// OpenT, CloseT and MidT echo the exposure interval.
	OpenT, CloseT, MidT float64
	// MidCamera is the camera pose at the middle of the interval.
	MidCamera Pose
	// MidObject holds each object's pose at the middle of the interval,
	// keyed by object index.
	MidObject []Pose
	// Color is the final tone-mapped image.
	Color []RGB
	// Coverage is per-pixel accumulated object/background occupancy in [0,1].
	Coverage []float64
	// Vectors is the camera-relative motion vector field.
	Vectors *VectorField
	width   int
	height  int
}

// Width returns the image width.
func (r *FrameResult) Width() int { return r.width }

// Height returns the image height.
func (r *FrameResult) Height() int { return r.height }

// Render integrates one exposure interval and returns the blurred frame,
// motion vector field, and recorded mid-interval poses.
func Render(req FrameRequest) *FrameResult {
	if req.Width <= 0 || req.Height <= 0 {
		panic("shutter: frame dimensions must be positive")
	}
	if req.CloseT <= req.OpenT {
		panic("shutter: CloseT must be greater than OpenT")
	}
	if req.CameraAt == nil {
		panic("shutter: CameraAt is required")
	}

	midT := req.MidTime()
	midCam := req.CameraAt(midT)

	result := &FrameResult{
		OpenT:     req.OpenT,
		CloseT:    req.CloseT,
		MidT:      midT,
		MidCamera: midCam,
		MidObject: make([]Pose, len(req.Objects)),
		width:     req.Width,
		height:    req.Height,
	}

	buf := newAccumBuffer(req.Width, req.Height)

	// The static background is evaluated once through the mid-exposure
	// camera pose. Because every later sample reuses this same canonical
	// frame, camera translation/rotation cannot turn it into a smear.
	if req.Background != nil {
		paintBackground(buf, req.Background, Pose{}, req.Projection, req.Width, req.Height, 1)
	}

	times := req.SampleTimes()
	weight := 1.0 / float64(len(times))

	for objIdx, obj := range req.Objects {
		result.MidObject[objIdx] = obj.PoseAt(midT)
		for _, t := range times {
			// World pose of the object at this sample time.
			worldPose := obj.PoseAt(t)
			// Re-express in the mid-exposure camera frame so the whole
			// exposure lands on one stable image plane.
			canonical := RelativeTo(midCam, worldPose)
			paintObject(buf, obj, canonical, req.Projection, req.Width, req.Height, weight)
		}
	}

	result.Color, result.Coverage = composite(buf)
	result.Vectors = buildVectors(req, midCam)
	return result
}

func paintBackground(buf *accumBuffer, bg *Background, camInv Pose, p OrthoProjection, w, h int, weight float64) {
	for _, shape := range bg.Shapes {
		paintShape(buf, shape, camInv, p, w, h, weight)
	}
}

func paintObject(buf *accumBuffer, obj *Object, pose Pose, p OrthoProjection, w, h int, weight float64) {
	for _, shape := range obj.Shapes {
		paintShape(buf, shape, pose, p, w, h, weight)
	}
}

func paintShape(buf *accumBuffer, shape Polygon, pose Pose, p OrthoProjection, w, h int, weight float64) {
	points := make([]Vec2, len(shape.Vertices))
	for i, v := range shape.Vertices {
		cam := pose.Point(v)
		px, py := p.Project(cam, w, h)
		points[i] = Vec2{X: px, Y: py}
	}
	buf.paintPolygon(points, shape.Color, weight)
}

// composite resolves the weighted accumulation buffer into premultiplied
// average colors over occupied samples and a coverage channel.
func composite(buf *accumBuffer) ([]RGB, []float64) {
	n := buf.width * buf.height
	color := make([]RGB, n)
	coverage := make([]float64, n)
	for i := 0; i < n; i++ {
		coverage[i] = clamp01(buf.weight[i])
		if buf.weight[i] > 0 {
			inv := 1.0 / buf.weight[i]
			color[i] = RGB{
				R: clamp01(buf.color[i].R * inv),
				G: clamp01(buf.color[i].G * inv),
				B: clamp01(buf.color[i].B * inv),
			}
		}
	}
	return color, coverage
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
