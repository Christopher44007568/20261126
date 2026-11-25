// Package shutter renders exposure-interval motion blur using only the Go
// standard library.
//
// # Design
//
// Motion blur is an integral over the open shutter interval [OpenT, CloseT],
// sampled at evenly spaced times. Every sample transforms object and camera
// geometry into ONE canonical image plane: the camera frame at the middle of
// the interval (MidT). This has two consequences:
//
//   - A world-static point maps to the same canonical coordinates at every
//     sample, so static background and world-static objects stay sharp
//     regardless of camera translation or rotation.
//   - Only motion RELATIVE to the camera contributes to the blur and to the
//     motion vectors, matching how a real camera exposes film.
//
// Blur length therefore depends on how far each material point travels
// between the shutter edges, not on a velocity derived from the previous
// frame. With constant relative velocity the on-screen blur length is linear
// in the shutter duration.
//
// # Outputs
//
// Render returns a FrameResult containing:
//   - the accumulated blurred image (Color) and per-pixel Coverage;
//   - the mid-interval camera pose and every object's mid-interval pose;
//   - a camera-relative VectorField (close-open displacement in pixels),
//     zero everywhere a moving object does not cover.
//
// RenderSubframes splits the interval into four equal quarters and renders
// each independently; Montage composes them into a 2x2 reference image.
//
// EncodePNG and VectorFieldNRGBA produce viewable images via image/png;
// WriteVectorBinary serializes exact float64 vectors in the MVF1 format.
package shutter
