package shutter

// Subframe is one quarter of the full shutter interval rendered without
// inter-sample blur beyond its own short exposure.
type Subframe struct {
	Index         int
	OpenT, CloseT float64
	MidT          float64
	MidCamera     Pose
	MidObject     []Pose
	Color         []RGB
	Coverage      []float64
	Width, Height int
}

// SubframeSet bundles four quarter-interval renders in chronological order.
type SubframeSet struct {
	Frames [4]*Subframe
	Width  int
	Height int
}

// RenderSubframes splits the request's shutter interval into four equal
// quarters and renders each one independently using the same scene. Each
// quarter uses the same sample count proportionally, clamped to one.
func RenderSubframes(req FrameRequest) *SubframeSet {
	set := &SubframeSet{Width: req.Width, Height: req.Height}
	quarter := req.Duration() / 4.0
	perQuarter := req.Samples / 4
	if perQuarter < 1 {
		perQuarter = 1
	}
	for i := 0; i < 4; i++ {
		sub := FrameRequest{
			Width:      req.Width,
			Height:     req.Height,
			Projection: req.Projection,
			OpenT:      req.OpenT + quarter*float64(i),
			CloseT:     req.OpenT + quarter*float64(i+1),
			Samples:    perQuarter,
			CameraAt:   req.CameraAt,
			Objects:    req.Objects,
			Background: req.Background,
		}
		res := Render(sub)
		set.Frames[i] = &Subframe{
			Index:     i,
			OpenT:     sub.OpenT,
			CloseT:    sub.CloseT,
			MidT:      res.MidT,
			MidCamera: res.MidCamera,
			MidObject: res.MidObject,
			Color:     res.Color,
			Coverage:  res.Coverage,
			Width:     req.Width,
			Height:    req.Height,
		}
	}
	return set
}

// Montage lays the four subframes out chronologically in a 2x2 grid:
// top-left Q1, top-right Q2, bottom-left Q3, bottom-right Q4.
func (s *SubframeSet) Montage() []RGB {
	w, h := s.Width, s.Height
	out := make([]RGB, (2*w)*(2*h))
	for i, f := range s.Frames {
		ox := (i % 2) * w
		oy := (i / 2) * h
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				src := f.Color[y*w+x]
				out[(y+oy)*2*w+(x+ox)] = src
			}
		}
	}
	return out
}
