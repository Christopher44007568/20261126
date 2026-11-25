package timeline

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Prepare sorts every track and validates playback settings. It is safe to
// call more than once.
func (s *Scene) Prepare() error {
	if len(s.Nodes) == 0 {
		return errors.New("timeline: scene has no nodes")
	}
	seen := make(map[string]bool, len(s.Nodes))
	primaryFound := s.PrimaryID == ""
	for ni := range s.Nodes {
		n := &s.Nodes[ni]
		if n.ID == "" {
			return fmt.Errorf("timeline: node %d has empty id", ni)
		}
		if seen[n.ID] {
			return fmt.Errorf("timeline: duplicate node id %q", n.ID)
		}
		seen[n.ID] = true
		if n.ID == s.PrimaryID {
			primaryFound = true
		}
		sort.SliceStable(n.Samples, func(i, j int) bool {
			return n.Samples[i].Time < n.Samples[j].Time
		})
		for i := 1; i < len(n.Samples); i++ {
			if n.Samples[i-1].Time == n.Samples[i].Time {
				return fmt.Errorf("timeline: node %q has duplicate sample time %.6g", n.ID, n.Samples[i].Time)
			}
		}
	}
	if !primaryFound {
		return fmt.Errorf("timeline: primary node %q not found", s.PrimaryID)
	}
	for i := range s.Sheets {
		sp := &s.Sheets[i]
		if sp.PlaybackFPS <= 0 {
			return fmt.Errorf("timeline: sheet %q has non-positive fps", sp.ID)
		}
		if sp.Retime.Rate <= 0 {
			return fmt.Errorf("timeline: sheet %q retime rate must be positive", sp.ID)
		}
		if sp.FrameCount <= 0 {
			return fmt.Errorf("timeline: sheet %q has no frames", sp.ID)
		}
		if sp.ReadMode != ModeStepHold && sp.ReadMode != ModeLinear {
			return fmt.Errorf("timeline: sheet %q has unsupported mode %q", sp.ID, sp.ReadMode)
		}
	}
	s.prepared = true
	return nil
}

// NodeByID returns a pointer to the named node.
func (s *Scene) NodeByID(id string) (*Node, bool) {
	for i := range s.Nodes {
		if s.Nodes[i].ID == id {
			return &s.Nodes[i], true
		}
	}
	return nil, false
}

// Evaluate reads one node at sourceTime. A track with no samples returns its
// own bind pose and StatusStatic; importantly it never consults another node.
func (n Node) Evaluate(sourceTime float64, mode ReadMode) NodeFrame {
	result := NodeFrame{
		ID:       n.ID,
		Kind:     n.Kind,
		Pose:     n.Bind,
		Status:   StatusStatic,
		Lo:       -1,
		Hi:       -1,
		Alpha:    0,
		ShownKey: -1,
	}
	if len(n.Samples) == 0 {
		return result
	}

	last := len(n.Samples) - 1
	firstTime := n.Samples[0].Time
	lastTime := n.Samples[last].Time

	if sourceTime < firstTime {
		result.Status = StatusBefore
		result.Pose = n.Samples[0].Pose
		result.Lo, result.Hi = 0, 0
		result.ShownKey = 0
		return result
	}
	if sourceTime > lastTime {
		result.Status = StatusAfter
		result.Pose = n.Samples[last].Pose
		result.Lo, result.Hi = last, last
		result.Alpha = 1
		result.ShownKey = last
		return result
	}
	if sourceTime == lastTime {
		result.Status = StatusIn
		result.Pose = n.Samples[last].Pose
		result.Lo, result.Hi = last, last
		result.Alpha = 1
		result.ShownKey = last
		return result
	}

	// Find first sample with time >= sourceTime. The source is known to be
	// inside the half-open range [first, last) here.
	hi := sort.Search(len(n.Samples), func(i int) bool {
		return n.Samples[i].Time >= sourceTime
	})
	if hi < len(n.Samples) && n.Samples[hi].Time == sourceTime {
		result.Status = StatusIn
		result.Pose = n.Samples[hi].Pose
		result.Lo, result.Hi = hi, hi
		result.ShownKey = hi
		return result
	}
	lo := hi - 1
	if lo < 0 {
		lo = 0
	}
	if hi > last {
		hi = last
	}
	left := n.Samples[lo]
	right := n.Samples[hi]
	span := right.Time - left.Time
	alpha := 0.0
	if span != 0 {
		alpha = (sourceTime - left.Time) / span
	}
	alpha = math.Max(0, math.Min(1, alpha))

	result.Status = StatusIn
	result.Lo, result.Hi = lo, hi
	result.BracketLo, result.BracketHi = lo, hi
	result.Alpha = alpha
	switch mode {
	case ModeLinear:
		result.Pose = LerpPose(left.Pose, right.Pose, alpha)
		if alpha == 0 {
			result.ShownKey = lo
		} else if alpha == 1 {
			result.ShownKey = hi
		}
	case ModeStepHold:
		result.Pose = left.Pose
		result.ShownKey = lo
	default:
		result.Pose = left.Pose
		result.ShownKey = lo
	}
	return result
}

// FrameAt evaluates the whole scene at playbackTime using spec's retime/mode.
func (s *Scene) FrameAt(index int, playbackTime float64, spec SheetSpec) Frame {
	sourceTime := spec.Retime.SourceTime(playbackTime)
	frame := Frame{
		Index:        index,
		PlaybackTime: playbackTime,
		SourceTime:   sourceTime,
		Nodes:        make([]NodeFrame, 0, len(s.Nodes)),
	}
	for _, n := range s.Nodes {
		frame.Nodes = append(frame.Nodes, n.Evaluate(sourceTime, spec.ReadMode))
	}
	if primary, ok := frame.Node(s.PrimaryID); ok {
		frame.PrimaryRangeStatus = primary.Status
	}
	return frame
}

// BuildFrames creates the fixed-frame-rate output sequence. Playback frame
// i is sampled at StartTime+i/FPS; retiming is applied independently.
func (s *Scene) BuildFrames(spec SheetSpec) []Frame {
	frames := make([]Frame, spec.FrameCount)
	for i := range frames {
		playbackTime := spec.StartTime + float64(i)/spec.PlaybackFPS
		frames[i] = s.FrameAt(i, playbackTime, spec)
	}
	return frames
}

// LerpPose linearly interpolates every joint channel.
func LerpPose(a, b Pose, t float64) Pose {
	return Pose{
		OffY:   lerp(a.OffY, b.OffY, t),
		Lean:   lerp(a.Lean, b.Lean, t),
		ArmL:   lerp(a.ArmL, b.ArmL, t),
		ArmR:   lerp(a.ArmR, b.ArmR, t),
		ElbowL: lerp(a.ElbowL, b.ElbowL, t),
		ElbowR: lerp(a.ElbowR, b.ElbowR, t),
		LegL:   lerp(a.LegL, b.LegL, t),
		LegR:   lerp(a.LegR, b.LegR, t),
		KneeL:  lerp(a.KneeL, b.KneeL, t),
		KneeR:  lerp(a.KneeR, b.KneeR, t),
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// FrameCountForRange returns a frame count that starts at time zero, includes
// the retimed last sample on or before its output frame, and adds tailFrames
// explicit out-of-range frames.
func FrameCountForRange(fps, rate, lastSampleTime float64, tailFrames int) int {
	endPlayback := lastSampleTime / rate
	lastFrame := int(math.Ceil(endPlayback * fps))
	if lastFrame < 0 {
		lastFrame = 0
	}
	return lastFrame + 1 + tailFrames
}
