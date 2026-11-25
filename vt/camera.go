package vt

import "math"

// Camera is a front-facing pinhole camera aimed at the square wall.
// Distance is measured in wall-height units; smaller values mean "closer".
type Camera struct {
	LookU       float64
	LookV       float64
	Distance    float64
	FOVYDegrees float64
}

func NewCamera(u, v, distance float64) Camera {
	return Camera{LookU: u, LookV: v, Distance: distance, FOVYDegrees: 55}
}

// NewCameraWithViewHeight constructs a camera with an exact UV footprint.
func NewCameraWithViewHeight(u, v, viewHeight float64) Camera {
	const fov = 55 * math.Pi / 180
	return Camera{
		LookU:       u,
		LookV:       v,
		FOVYDegrees: 55,
		Distance:    viewHeight / (2 * math.Tan(fov/2)),
	}
}

// ViewHeight returns the visible UV span vertically.
func (c Camera) ViewHeight() float64 {
	fov := c.FOVYDegrees * math.Pi / 180
	return 2 * c.Distance * math.Tan(fov/2)
}

// Bounds returns left/top/right/bottom normalized wall UV coordinates.
func (c Camera) Bounds(aspect float64) (u0, v0, u1, v1 float64) {
	h := c.ViewHeight()
	w := h * aspect
	return c.LookU - w/2, c.LookV - h/2,
		c.LookU + w/2, c.LookV + h/2
}

// ClampToWall keeps the complete footprint on the square texture when possible.
func (c Camera) ClampToWall(aspect float64) Camera {
	u0, v0, u1, v1 := c.Bounds(aspect)
	out := c
	if u0 < 0 {
		out.LookU += -u0
	}
	if u1 > 1 {
		out.LookU -= u1 - 1
	}
	if v0 < 0 {
		out.LookV += -v0
	}
	if v1 > 1 {
		out.LookV -= v1 - 1
	}
	_, _, u1, v1 = out.Bounds(aspect)
	if u1 < 0 || u1 > 1 {
		out.LookU = 0.5
	}
	if v1 < 0 || v1 > 1 {
		out.LookV = 0.5
	}
	return out
}

// Mip selects an integer mip from base-texel footprint per framebuffer pixel.
func Mip(info Info, duDX, dvDY float64) int {
	texels := math.Max(math.Abs(duDX)*float64(info.BaseSize),
		math.Abs(dvDY)*float64(info.BaseSize))
	if texels <= 0 {
		return 0
	}
	mip := int(math.Round(math.Log2(texels)))
	return info.ClampMip(mip)
}

// FeedbackSample is one read-back request generated from a coarse UV block.
type FeedbackSample struct {
	BufferX int
	BufferY int
	U       float64
	V       float64
	Mip     int
	Page    PageID
}

// FeedbackBuffer is the CPU equivalent of a GPU UV/page-ID readback target.
type FeedbackBuffer struct {
	Width   int
	Height  int
	Stride  int
	Samples []FeedbackSample
	Counts  map[PageID]int
}

// BuildFeedback downsamples the camera view into page requests. Mip is derived
// from full-resolution pixel derivatives, as in a hardware feedback pass.
func BuildFeedback(cam Camera, frameW, frameH, stride int, info Info) *FeedbackBuffer {
	if stride < 1 {
		stride = 1
	}
	fbW := (frameW + stride - 1) / stride
	fbH := (frameH + stride - 1) / stride
	u0, v0, u1, v1 := cam.Bounds(float64(frameW) / float64(frameH))
	duDX := (u1 - u0) / float64(frameW)
	dvDY := (v1 - v0) / float64(frameH)
	mip := Mip(info, duDX, dvDY)

	fb := &FeedbackBuffer{
		Width:   fbW,
		Height:  fbH,
		Stride:  stride,
		Samples: make([]FeedbackSample, 0, fbW*fbH),
		Counts:  make(map[PageID]int),
	}
	for by := 0; by < fbH; by++ {
		py := (by*stride + stride/2)
		if py >= frameH {
			py = frameH - 1
		}
		v := v0 + (float64(py)+0.5)/float64(frameH)*(v1-v0)
		for bx := 0; bx < fbW; bx++ {
			px := bx*stride + stride/2
			if px >= frameW {
				px = frameW - 1
			}
			u := u0 + (float64(px)+0.5)/float64(frameW)*(u1-u0)
			uu, vv := ClampUV(u, v)
			page := info.PageAtUV(uu, vv, mip)
			sample := FeedbackSample{
				BufferX: bx, BufferY: by, U: uu, V: vv,
				Mip: mip, Page: page,
			}
			fb.Samples = append(fb.Samples, sample)
			fb.Counts[page]++
		}
	}
	return fb
}
