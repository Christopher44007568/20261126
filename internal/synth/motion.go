package synth

import (
	"math"

	"deshake/internal/geom"
)

// Frame holds everything a stabilization pipeline needs for one captured image.
// RowH is the per-row source homography (world->pixel) induced by the rolling
// shutter; MidPose is the pose at the central exposure instant.
type Frame struct {
	Index   int
	W, H    int
	Img     *RGB
	MidPose geom.Pose
	RefPose geom.Pose // smoothed, jitter-free reference pose
	RowH    []geom.Mat3
	MidH    geom.Mat3
}

// RGB is a packed row-major image with linear channels.
type RGB struct {
	W, H int
	Pix  []float64
}

func NewRGB(w, h int) *RGB { return &RGB{W: w, H: h, Pix: make([]float64, w*h*3)} }

func (im *RGB) Set(x, y int, r, g, b float64) {
	i := (y*im.W + x) * 3
	im.Pix[i], im.Pix[i+1], im.Pix[i+2] = r, g, b
}

func (im *RGB) At(x, y int) (r, g, b float64) {
	i := (y*im.W + x) * 3
	return im.Pix[i], im.Pix[i+1], im.Pix[i+2]
}

// Motion generates handheld camera poses over time, including a slow pan,
// a deliberately fast turn (worst case for rolling shutter) and hand jitter.
type Motion struct {
	Cam       geom.Camera
	N         int
	ReadStart float64 // readout window start within a frame interval
	ReadEnd   float64
}

func NewMotion(cam geom.Camera, n int) *Motion {
	return &Motion{Cam: cam, N: n, ReadStart: 0.0, ReadEnd: 1.0}
}

// intendedYaw is the jitter-free operator motion: a slow pan, a sharp turn
// around frame 13, then another gentle pan.
func intendedYaw(t float64) float64 {
	slow := geom.Deg(7.0) * t
	turn := geom.Deg(20.0) * 0.5 * (1 + math.Tanh((t-0.42)/0.022))
	return -geom.Deg(10) + slow + turn
}

func intendedPitch(t float64) float64 {
	return geom.Deg(3.5) * math.Sin(2*math.Pi*t*1.3)
}

// jitter returns high-frequency handheld perturbation at continuous time.
func jitter(t float64) geom.Pose {
	return geom.Pose{
		Yaw:   geom.Deg(3.6)*math.Sin(2*math.Pi*5.1*t) + geom.Deg(1.6)*math.Sin(2*math.Pi*11.7*t),
		Pitch: geom.Deg(3.1)*math.Sin(2*math.Pi*4.3*t+1.2) + geom.Deg(1.3)*math.Sin(2*math.Pi*9.2*t),
		Roll:  geom.Deg(1.7) * math.Sin(2*math.Pi*3.7*t+0.6),
		Tx:    16.0 * math.Sin(2*math.Pi*4.8*t),
		Ty:    13.0 * math.Sin(2*math.Pi*4.0*t+2.0),
	}
}

// PoseAt returns the actual handheld pose at continuous time t in frames.
func (m *Motion) PoseAt(t float64) geom.Pose {
	j := jitter(t / float64(m.N))
	tt := t / float64(m.N)
	return geom.Pose{
		Yaw:   intendedYaw(tt) + j.Yaw,
		Pitch: intendedPitch(tt) + j.Pitch,
		Roll:  j.Roll,
		Tx:    24 + j.Tx,
		Ty:    j.Ty,
	}
}

// RefPoseAt returns the smoothed intended pose used as the stabilization target.
func (m *Motion) RefPoseAt(t float64) geom.Pose {
	tt := t / float64(m.N)
	return geom.Pose{
		Yaw:   intendedYaw(tt),
		Pitch: intendedPitch(tt),
		Tx:    24,
	}
}

// ReadTime maps a fractional row to continuous capture time for frame index fi.
func (m *Motion) ReadTime(fi int, rowFrac float64) float64 {
	return float64(fi) + m.ReadStart + (m.ReadEnd-m.ReadStart)*rowFrac
}
