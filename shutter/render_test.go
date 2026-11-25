package shutter

import (
	"image/color"
	"math"
	"testing"
)

const (
	testW = 160
	testH = 120
	ppu   = 40.0 // pixels per world unit
)

func square(size float64, c RGB) []Polygon {
	h := size * 0.5
	return []Polygon{{
		Vertices: []Vec2{
			{-h, -h}, {h, -h}, {h, h}, {-h, h},
		},
		Color: c,
	}}
}

func movingSquare(speed float64) *Object {
	return &Object{
		Name:   "mover",
		Shapes: square(1.0, RGB{R: 0.9, G: 0.2, B: 0.1}),
		PoseAt: func(t float64) Pose {
			return Pose{Pos: Vec2{X: speed * t, Y: 0}}
		},
	}
}

func staticBackground() *Background {
	return &Background{Shapes: square(0.6, RGB{R: 0.2, G: 0.4, B: 0.8})}
}

func baseRequest(obj *Object, open, close float64, samples int) FrameRequest {
	return FrameRequest{
		Width:      testW,
		Height:     testH,
		Projection: OrthoProjection{PixelsPerUnit: ppu, Center: Vec2{}},
		OpenT:      open,
		CloseT:     close,
		Samples:    samples,
		CameraAt:   func(float64) Pose { return Pose{} },
		Objects:    []*Object{obj},
	}
}

// occupiedSpan measures, at the object's mid row, the width in pixels over
// which coverage exceeds the given threshold.
func occupiedSpan(t *testing.T, r *FrameResult, row int, thresh float64) int {
	t.Helper()
	first, last := -1, -1
	for x := 0; x < r.Width(); x++ {
		if r.Coverage[row*r.Width()+x] > thresh {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	if first < 0 {
		return 0
	}
	return last - first + 1
}

// TestBlurLengthLinearInShutter verifies that doubling the open interval
// doubles the on-screen blur length instead of relying on frame velocity.
func TestBlurLengthLinearInShutter(t *testing.T) {
	const speed = 2.0 // world units per second
	obj := movingSquare(speed)

	short := Render(baseRequest(obj, 0.0, 0.1, 41))
	long := Render(baseRequest(obj, 0.0, 0.2, 81))

	row := testH / 2
	// Object is 1 unit wide (40 px); expected span = 40 + speed*dt*ppu.
	wantShort := 40 + int(math.Round(speed*0.1*ppu))
	wantLong := 40 + int(math.Round(speed*0.2*ppu))

	gotShort := occupiedSpan(t, short, row, 0.05)
	gotLong := occupiedSpan(t, long, row, 0.05)

	if math.Abs(float64(gotShort-wantShort)) > 2 {
		t.Fatalf("short span = %d, want ~%d", gotShort, wantShort)
	}
	if math.Abs(float64(gotLong-wantLong)) > 2 {
		t.Fatalf("long span = %d, want ~%d", gotLong, wantLong)
	}
	if math.Abs(float64(gotLong)/float64(gotShort)-float64(wantLong)/float64(wantShort)) > 0.05 {
		t.Fatalf("blur did not scale linearly: %d vs %d", gotShort, gotLong)
	}
}

// TestStaticBackgroundNoCameraSmear moves the camera across the whole
// exposure and requires every background pixel to stay identical (sharp).
func TestStaticBackgroundNoCameraSmear(t *testing.T) {
	req := baseRequest(movingSquare(0), 0.0, 0.2, 41)
	req.Objects = nil
	req.Background = staticBackground()
	req.CameraAt = func(t float64) Pose {
		// Strong lateral camera ego-motion during the exposure.
		return Pose{Pos: Vec2{X: 3.0 * t, Y: 0.5 * t}}
	}

	still := req
	// Exactly the mid-exposure pose of the moving camera.
	still.CameraAt = func(float64) Pose { return Pose{Pos: Vec2{X: 3.0 * 0.1, Y: 0.5 * 0.1}} }

	moving := Render(req)
	fixed := Render(still)

	for i := range moving.Color {
		if moving.Color[i] != fixed.Color[i] {
			t.Fatalf("background pixel %d changed under camera motion: %+v vs %+v",
				i, moving.Color[i], fixed.Color[i])
		}
		if moving.Coverage[i] != fixed.Coverage[i] {
			t.Fatalf("background coverage %d changed: %v vs %v",
				i, moving.Coverage[i], fixed.Coverage[i])
		}
	}
}

// TestIntervalNotPreviousFrameVelocity places an object whose velocity jumps
// after the shutter closes; the rendered blur must follow in-interval motion.
func TestIntervalNotPreviousFrameVelocity(t *testing.T) {
	obj := &Object{
		Name:   "jumpy",
		Shapes: square(1.0, RGB{R: 0.1, G: 0.8, B: 0.2}),
		PoseAt: func(t float64) Pose {
			if t > 0.1+1e-9 {
				// Teleport far away only after the shutter has closed.
				return Pose{Pos: Vec2{X: 50, Y: 50}}
			}
			return Pose{Pos: Vec2{X: 3.0 * t, Y: 0}}
		},
	}
	req := baseRequest(obj, 0.0, 0.1, 41)
	res := Render(req)

	// All object pixels must remain within the central band of the image.
	row := testH / 2
	for x := 0; x < res.Width(); x++ {
		i := row*res.Width() + x
		if res.Coverage[i] > 0 && (x < 40 || x > 120) {
			t.Fatalf("object leaked to x=%d from post-interval teleport", x)
		}
	}
	if span := occupiedSpan(t, res, row, 0.05); span == 0 {
		t.Fatal("object missing from frame")
	}
}

// TestMidPosesRecorded checks both camera and object mid-interval poses.
func TestMidPosesRecorded(t *testing.T) {
	obj := movingSquare(4.0)
	req := baseRequest(obj, 0.2, 0.6, 21)
	req.CameraAt = func(t float64) Pose { return Pose{Pos: Vec2{Y: t}, Rot: t} }
	res := Render(req)

	if math.Abs(res.MidT-0.4) > 1e-12 {
		t.Fatalf("mid time = %v", res.MidT)
	}
	if math.Abs(res.MidCamera.Pos.Y-0.4) > 1e-12 || math.Abs(res.MidCamera.Rot-0.4) > 1e-12 {
		t.Fatalf("mid camera pose wrong: %+v", res.MidCamera)
	}
	if math.Abs(res.MidObject[0].Pos.X-1.6) > 1e-12 {
		t.Fatalf("mid object pose wrong: %+v", res.MidObject[0])
	}
}

// TestMotionVectorsMatchInterval verifies vector magnitude equals the
// shutter-edge displacement and that background pixels stay zero.
func TestMotionVectorsMatchInterval(t *testing.T) {
	const speed = 2.5
	obj := movingSquare(speed)
	req := baseRequest(obj, 0.0, 0.2, 9)
	req.Background = staticBackground()
	res := Render(req)

	want := speed * 0.2 * ppu // pixels over the full interval
	found := 0
	for i, mask := range res.Vectors.Mask {
		if mask {
			found++
			if math.Abs(res.Vectors.X[i]-want) > 1e-9 || math.Abs(res.Vectors.Y[i]) > 1e-9 {
				t.Fatalf("vector %d = (%v,%v), want (%v,0)", i, res.Vectors.X[i], res.Vectors.Y[i], want)
			}
		} else if res.Vectors.X[i] != 0 || res.Vectors.Y[i] != 0 {
			t.Fatalf("non-object pixel %d carries a vector", i)
		}
	}
	if found == 0 {
		t.Fatal("no motion vectors produced")
	}
}

// TestCameraMotionObjectRelativeBlur moves only the camera while the object
// stays world-static. In the mid-exposure canonical frame the object never
// moves, so it must render sharp with zero vectors even though the camera
// translates substantially across the exposure.
func TestCameraMotionObjectRelativeBlur(t *testing.T) {
	obj := &Object{
		Name:   "static-world-object",
		Shapes: square(1.0, RGB{R: 0.9, G: 0.2, B: 0.1}),
		PoseAt: func(float64) Pose { return Pose{} },
	}
	req := baseRequest(obj, 0.0, 0.2, 41)
	req.CameraAt = func(t float64) Pose { return Pose{Pos: Vec2{X: 3.0 * t, Y: 0.4 * t}} }
	res := Render(req)

	for i, mask := range res.Vectors.Mask {
		if mask {
			t.Fatalf("co-moving object produced a vector at %d", i)
		}
	}
	row := testH / 2
	for x := 0; x < res.Width(); x++ {
		c := res.Coverage[row*res.Width()+x]
		if c > 0 && c != 1 {
			t.Fatalf("co-moving object blurred at x=%d coverage=%v", x, c)
		}
	}
}

// TestSubframesQuarterTheExposure ensures four chronological subframes cover
// the interval exactly and each records its own mid poses.
func TestSubframesQuarterTheExposure(t *testing.T) {
	obj := movingSquare(2.0)
	req := baseRequest(obj, 0.0, 0.4, 16)
	set := RenderSubframes(req)

	quarter := 0.1
	for i, f := range set.Frames {
		wantOpen := quarter * float64(i)
		wantClose := quarter * float64(i+1)
		if math.Abs(f.OpenT-wantOpen) > 1e-12 || math.Abs(f.CloseT-wantClose) > 1e-12 {
			t.Fatalf("subframe %d interval = [%v,%v]", i, f.OpenT, f.CloseT)
		}
		if math.Abs(f.MidT-(wantOpen+wantClose)/2) > 1e-12 {
			t.Fatalf("subframe %d mid = %v", i, f.MidT)
		}
		if math.Abs(f.MidObject[0].Pos.X-2.0*f.MidT) > 1e-12 {
			t.Fatalf("subframe %d mid object pose wrong", i)
		}
	}
	montage := set.Montage()
	if len(montage) != (2*testW)*(2*testH) {
		t.Fatalf("montage size = %d", len(montage))
	}
}

// TestPNGRoundTrip ensures standard-library encoding succeeds and is a PNG.
func TestPNGRoundTrip(t *testing.T) {
	res := Render(baseRequest(movingSquare(1.0), 0, 0.1, 5))
	img := ToNRGBA(res.Width(), res.Height(), res.Color, res.Coverage, color.NRGBA{A: 255})
	var buf bytesBuffer
	if err := EncodePNG(&buf, img); err != nil {
		t.Fatal(err)
	}
	if string(buf.bytes[:4]) != "\x89PNG" {
		t.Fatal("output is not a PNG")
	}
}

type bytesBuffer struct{ bytes []byte }

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.bytes = append(b.bytes, p...)
	return len(p), nil
}

// TestRotatingCameraKeepsBackgroundSharp exercises the rotation branch of
// the rigid inverse transform (the sign-sensitive part of the math).
func TestRotatingCameraKeepsBackgroundSharp(t *testing.T) {
	req := baseRequest(movingSquare(0), 0.0, 0.3, 31)
	req.Objects = nil
	req.Background = staticBackground()
	req.CameraAt = func(t float64) Pose {
		return Pose{Pos: Vec2{X: 0.7 * t, Y: -0.3 * t}, Rot: 0.9 * t}
	}

	reference := req
	reference.CameraAt = func(float64) Pose {
		return req.CameraAt(req.MidTime())
	}

	moving := Render(req)
	fixed := Render(reference)
	for i := range moving.Color {
		if moving.Color[i] != fixed.Color[i] || moving.Coverage[i] != fixed.Coverage[i] {
			t.Fatalf("rotating camera altered background pixel %d", i)
		}
	}
}

// TestRotatingObjectProducesVectors ensures rotation-only motion yields
// nonzero vectors and the vector round-trip math stays consistent.
func TestRotatingObjectProducesVectors(t *testing.T) {
	obj := &Object{
		Name:   "spinner",
		Shapes: square(1.0, RGB{R: 0.6, G: 0.6, B: 0.1}),
		PoseAt: func(t float64) Pose { return Pose{Rot: 2.0 * t} },
	}
	req := baseRequest(obj, 0.0, 0.25, 21)
	res := Render(req)

	nonZero := 0
	for i, mask := range res.Vectors.Mask {
		if mask {
			if math.Hypot(res.Vectors.X[i], res.Vectors.Y[i]) > 0.5 {
				nonZero++
			}
		}
	}
	// A half-rotating unit square should have most pixels well off-center.
	if nonZero < (testW*testH)/100 {
		t.Fatalf("rotating object produced only %d significant vectors", nonZero)
	}
	if math.Abs(res.MidObject[0].Rot-0.25) > 1e-12 {
		t.Fatalf("mid rotation = %v", res.MidObject[0].Rot)
	}
}

// TestVectorBinaryRoundTrip checks the documented MVF1 serialization.
func TestVectorBinaryRoundTrip(t *testing.T) {
	res := Render(baseRequest(movingSquare(1.5), 0, 0.2, 7))
	var buf bytesBuffer
	if err := WriteVectorBinary(&buf, res.Vectors); err != nil {
		t.Fatal(err)
	}
	if string(buf.bytes[:4]) != "MVF1" {
		t.Fatal("bad magic")
	}
	want := 8 + 16*testW*testH
	if len(buf.bytes) != want {
		t.Fatalf("binary size = %d, want %d", len(buf.bytes), want)
	}
}
