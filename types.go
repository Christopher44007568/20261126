package main

import (
	"fmt"
	"math"
	"strings"
)

// CameraVersion, TextureVersion and DescriptorVersion are independent
// monotonic revisions. A resource is never edited in place after publication.
type CameraVersion uint64
type TextureVersion uint64
type DescriptorVersion uint64

// Mat3 is a row-major homography. It maps world coordinates to pixels:
//
//	x' = (m0*x + m1*y + m2) / w
//	y' = (m3*x + m4*y + m5) / w
//	w  = (m6*x + m7*y + m8)
type Mat3 [9]float64

// Vec2 is a point in world or pixel space, depending on context.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (v Vec2) add(o Vec2) Vec2 { return Vec2{X: v.X + o.X, Y: v.Y + o.Y} }
func (v Vec2) sub(o Vec2) Vec2 { return Vec2{X: v.X - o.X, Y: v.Y - o.Y} }
func (v Vec2) scale(s float64) Vec2 {
	return Vec2{X: v.X * s, Y: v.Y * s}
}
func (v Vec2) length() float64 { return math.Hypot(v.X, v.Y) }
func (v Vec2) midpoint(o Vec2) Vec2 {
	return Vec2{X: (v.X + o.X) / 2, Y: (v.Y + o.Y) / 2}
}

// CameraMatrix builds a rotated, non-uniformly scaled 2D camera with a small
// projective term. y increases downward in the target image.
func CameraMatrix(cx, cy, sx, sy, rotation, perspective float64) Mat3 {
	c, s := math.Cos(rotation), math.Sin(rotation)
	return Mat3{
		sx * c, -sy * s, cx,
		sx * s, -sy * c, cy,
		perspective * c, perspective * s, 1,
	}
}

// Transform projects a world point. The boolean is false when w is too close
// to zero or the point lies behind the projective plane.
func (m Mat3) Transform(p Vec2) (Vec2, bool) {
	w := m[6]*p.X + m[7]*p.Y + m[8]
	if math.Abs(w) < 1e-9 {
		return Vec2{}, false
	}
	x := (m[0]*p.X + m[1]*p.Y + m[2]) / w
	y := (m[3]*p.X + m[4]*p.Y + m[5]) / w
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return Vec2{}, false
	}
	return Vec2{X: x, Y: y}, true
}

// TransformPoints projects a polygon and fails if any vertex is invalid.
func (m Mat3) TransformPoints(points []Vec2) ([]Vec2, bool) {
	out := make([]Vec2, len(points))
	for i, p := range points {
		q, ok := m.Transform(p)
		if !ok {
			return nil, false
		}
		out[i] = q
	}
	return out, true
}

// Inverse returns the inverse homography, if it is numerically invertible.
func (m Mat3) Inverse() (Mat3, bool) {
	a, b, c := m[0], m[1], m[2]
	d, e, f := m[3], m[4], m[5]
	g, h, i := m[6], m[7], m[8]

	A := e*i - f*h
	B := c*h - b*i
	C := b*f - c*e
	D := f*g - d*i
	E := a*i - c*g
	F := c*d - a*f
	G := d*h - e*g
	H := b*g - a*h
	I := a*e - b*d

	det := a*A + b*D + c*G
	if math.Abs(det) < 1e-12 {
		return Mat3{}, false
	}
	invDet := 1 / det
	return Mat3{
		A * invDet, B * invDet, C * invDet,
		D * invDet, E * invDet, F * invDet,
		G * invDet, H * invDet, I * invDet,
	}, true
}

func (m Mat3) equal(o Mat3) bool {
	for i := range m {
		d := math.Abs(m[i] - o[i])
		scale := math.Max(1, math.Max(math.Abs(m[i]), math.Abs(o[i])))
		if d > 1e-9*scale {
			return false
		}
	}
	return true
}

// InvalidationFlags describes why a descriptor snapshot no longer matches the
// current resource revisions. The flags are advisory: a pinned snapshot keeps
// rendering with its exact historical resources until the material rebinds.
type InvalidationFlags uint32

const (
	// The camera revision changed after this descriptor was bound.
	InvalidateCameraChanged InvalidationFlags = 1 << iota
	// The texture revision changed after this descriptor was bound.
	InvalidateTextureChanged
	// The descriptor is not bound to the latest camera/texture pair.
	InvalidateDescriptorStale
	// A stored snapshot failed its resource identity check.
	InvalidateSnapshotMismatch
)

const InvalidateNone InvalidationFlags = 0

func (f InvalidationFlags) Has(flag InvalidationFlags) bool { return f&flag != 0 }

func (f InvalidationFlags) Names() []string {
	if f == InvalidateNone {
		return nil
	}
	names := []string{}
	for _, item := range []struct {
		flag InvalidationFlags
		name string
	}{
		{InvalidateCameraChanged, "camera_changed"},
		{InvalidateTextureChanged, "texture_changed"},
		{InvalidateDescriptorStale, "descriptor_stale"},
		{InvalidateSnapshotMismatch, "snapshot_mismatch"},
	} {
		if f.Has(item.flag) {
			names = append(names, item.name)
		}
	}
	return names
}

func (f InvalidationFlags) String() string {
	if f == InvalidateNone {
		return "fresh"
	}
	return strings.Join(f.Names(), "+")
}

// CameraResource is an immutable published camera revision.
type CameraResource struct {
	Version  CameraVersion `json:"version"`
	Label    string        `json:"label"`
	Matrix   Mat3          `json:"-"`
	Sequence uint64        `json:"sequence"`
}

// TextureResource is an immutable published texture revision. Pixels are
// copied on publication so a caller cannot mutate a live material.
type TextureResource struct {
	Version TextureVersion `json:"version"`
	Label   string         `json:"label"`
	Width   int            `json:"width"`
	Height  int            `json:"height"`
	Hash    string         `json:"hash"`
}

func (f InvalidationFlags) GoString() string { return fmt.Sprintf("%#x(%s)", uint32(f), f) }
