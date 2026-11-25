// Package timeline evaluates self-timed animation tracks against an
// independent playback clock. It has no third-party dependencies.
package timeline

// Pose is the small joint model used by the demo character. Angles are in
// degrees; OffY moves the hip in screen coordinates (positive is upward).
type Pose struct {
	OffY   float64 `json:"off_y"`
	Lean   float64 `json:"lean"`
	ArmL   float64 `json:"arm_l"`
	ArmR   float64 `json:"arm_r"`
	ElbowL float64 `json:"elbow_l"`
	ElbowR float64 `json:"elbow_r"`
	LegL   float64 `json:"leg_l"`
	LegR   float64 `json:"leg_r"`
	KneeL  float64 `json:"knee_l"`
	KneeR  float64 `json:"knee_r"`
}

// Sample is one irregularly spaced time sample on a node track.
type Sample struct {
	Time      float64 `json:"time"`
	Name      string  `json:"name,omitempty"`
	Signature bool    `json:"signature,omitempty"`
	Pose      Pose    `json:"pose"`
}

// Node is an independently evaluated scene object. Samples may be empty; in
// that case Bind is the only pose this node is allowed to use.
type Node struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Bind    Pose     `json:"bind_pose"`
	Samples []Sample `json:"samples,omitempty"`
}

// Retime maps playback time to source/sample time:
//
//	source = Offset + Rate*playback
type Retime struct {
	Rate   float64 `json:"rate"`
	Offset float64 `json:"offset"`
}

// SourceTime applies the retiming curve.
func (r Retime) SourceTime(playbackTime float64) float64 {
	return r.Offset + r.Rate*playbackTime
}

// ReadMode selects sample reading behaviour between exact samples.
type ReadMode string

const (
	ModeStepHold ReadMode = "step_hold"
	ModeLinear   ReadMode = "linear"
)

// RangeStatus explains where a queried source time fell relative to a track.
type RangeStatus string

const (
	StatusStatic RangeStatus = "no_samples_static"
	StatusBefore RangeStatus = "before_sample_range"
	StatusIn     RangeStatus = "in_range"
	StatusAfter  RangeStatus = "after_sample_range"
)

// NodeFrame is the fully resolved value for one node at one output frame.
type NodeFrame struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	Pose      Pose        `json:"pose"`
	Status    RangeStatus `json:"status"`
	Lo        int         `json:"lo"`
	Hi        int         `json:"hi"`
	Alpha     float64     `json:"alpha"`
	ShownKey  int         `json:"shown_key"`
	BracketLo int         `json:"-"`
	BracketHi int         `json:"-"`
}

// Frame contains the clock times and every independently evaluated node.
type Frame struct {
	Index              int         `json:"index"`
	PlaybackTime       float64     `json:"playback_time"`
	SourceTime         float64     `json:"source_time"`
	PrimaryRangeStatus RangeStatus `json:"primary_range_status"`
	Nodes              []NodeFrame `json:"nodes"`
}

// Node returns the evaluation for id, or false when absent.
func (f Frame) Node(id string) (NodeFrame, bool) {
	for _, n := range f.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return NodeFrame{}, false
}

// SheetSpec describes one contact sheet / playback configuration.
type SheetSpec struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	CNTitle     string   `json:"cn_title"`
	Blurb       string   `json:"blurb"`
	PlaybackFPS float64  `json:"playback_fps"`
	FrameCount  int      `json:"frame_count"`
	StartTime   float64  `json:"start_time"`
	Retime      Retime   `json:"retime"`
	ReadMode    ReadMode `json:"read_mode"`
	Cols        int      `json:"cols"`
	CellW       int      `json:"cell_w"`
	CellH       int      `json:"cell_h"`
}

// Scene is the self-contained demo input.
type Scene struct {
	Title     string      `json:"title"`
	PrimaryID string      `json:"primary_id"`
	Nodes     []Node      `json:"nodes"`
	Sheets    []SheetSpec `json:"sheets"`
	prepared  bool
}
