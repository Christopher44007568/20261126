// Package shutter implements exposure-interval based motion blur.
//
// Blur is produced by sampling poses ACROSS the open/close shutter interval,
// never from a finite difference against the previous frame. All geometry is
// projected through the camera pose at the middle of the interval into one
// canonical image, which keeps a static background sharp even while the
// camera moves.
package shutter

import "math"

// Vec2 is a 2D vector in world units.
type Vec2 struct {
	X float64
	Y float64
}

// Add returns v+w.
func (v Vec2) Add(w Vec2) Vec2 { return Vec2{v.X + w.X, v.Y + w.Y} }

// Sub returns v-w.
func (v Vec2) Sub(w Vec2) Vec2 { return Vec2{v.X - w.X, v.Y - w.Y} }

// Scale returns v*s.
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }

// Len returns the Euclidean length of v.
func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

// Pose is a rigid 2D transform: a rotation in radians followed by translation.
// A world point p maps to Pose.Point(p): R(rot)*p + pos.
type Pose struct {
	Pos Vec2
	Rot float64
}

// Point applies the rigid transform to p.
func (t Pose) Point(p Vec2) Vec2 {
	c, s := math.Cos(t.Rot), math.Sin(t.Rot)
	return Vec2{c*p.X - s*p.Y + t.Pos.X, s*p.X + c*p.Y + t.Pos.Y}
}

// Inverse returns the transform that undoes t.
func (t Pose) Inverse() Pose {
	c, s := math.Cos(t.Rot), math.Sin(t.Rot)
	return Pose{
		Rot: -t.Rot,
		Pos: Vec2{
			X: -(c*t.Pos.X + s*t.Pos.Y),
			Y: -(-s*t.Pos.X + c*t.Pos.Y),
		},
	}
}

// Compose returns the transform equivalent to applying outer after inner.
func Compose(outer, inner Pose) Pose {
	c, s := math.Cos(outer.Rot), math.Sin(outer.Rot)
	return Pose{
		Rot: outer.Rot + inner.Rot,
		Pos: Vec2{
			c*inner.Pos.X - s*inner.Pos.Y + outer.Pos.X,
			s*inner.Pos.X + c*inner.Pos.Y + outer.Pos.Y,
		},
	}
}

// RelativeTo returns the pose of target expressed in reference's local
// frame, i.e. reference.Inverse() composed with target. It is the correct
// way to remove camera ego-motion from a world-space object pose.
func RelativeTo(reference, target Pose) Pose {
	return Compose(reference.Inverse(), target)
}

// RGB is a linear color with channels in [0,1].
type RGB struct {
	R, G, B float64
}

// Polygon is a closed convex shape given in an object's local coordinates.
type Polygon struct {
	Vertices []Vec2
	Color    RGB
}

// Object is a rigid body whose pose is defined by a pose function of time.
type Object struct {
	Name   string
	Shapes []Polygon
	PoseAt func(t float64) Pose
}

// Background is static world geometry rendered through the mid-exposure
// camera pose only, so camera motion can never smear it.
type Background struct {
	Shapes []Polygon
}

// FrameRequest describes one exposure interval.
type FrameRequest struct {
	// Width and height of the output image in pixels.
	Width, Height int
	// Projection maps mid-camera-frame world coordinates to pixel positions.
	Projection OrthoProjection
	// OpenT and CloseT are the world times of the shutter edges.
	OpenT, CloseT float64
	// Samples is the number of evenly spaced sub-interval samples used to
	// integrate the exposure. Must be >= 1.
	Samples int
	// CameraAt returns the camera pose at a world time.
	CameraAt func(t float64) Pose
	// Scene objects and optional static background.
	Objects    []*Object
	Background *Background
}

// OrthoProjection is an orthographic mapping from camera-frame world units
// to pixels. The camera-frame point Center lands on the image center; camera
// +Y maps to image -Y so that world up is screen up.
type OrthoProjection struct {
	// PixelsPerUnit scales world distances into pixel distances.
	PixelsPerUnit float64
	// Center is the camera-frame point shown at the image center.
	Center Vec2
}

// Project converts a camera-frame point to floating point pixel coordinates
// for an image of the given dimensions.
func (p OrthoProjection) Project(pt Vec2, width, height int) (x, y float64) {
	d := pt.Sub(p.Center)
	x = float64(width)*0.5 + d.X*p.PixelsPerUnit
	y = float64(height)*0.5 - d.Y*p.PixelsPerUnit
	return
}

// MidTime returns the time at the middle of the exposure interval.
func (r FrameRequest) MidTime() float64 { return (r.OpenT + r.CloseT) * 0.5 }

// Duration returns the shutter duration CloseT-OpenT.
func (r FrameRequest) Duration() float64 { return r.CloseT - r.OpenT }

// SampleTimes returns the evenly spaced integration times, including both
// shutter edges. With Samples n there are n times; n=1 yields the mid time.
func (r FrameRequest) SampleTimes() []float64 {
	n := r.Samples
	if n < 1 {
		n = 1
	}
	if n == 1 {
		return []float64{r.MidTime()}
	}
	times := make([]float64, n)
	for i := 0; i < n; i++ {
		times[i] = r.OpenT + r.Duration()*float64(i)/float64(n-1)
	}
	return times
}
