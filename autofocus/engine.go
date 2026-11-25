package autofocus

import (
	"fmt"
	"image"
)

// Sample is one focus measurement at a known lens position.
type Sample struct {
	Frame     int     `json:"frame"`
	Position  int     `json:"position"`
	Sharpness float64 `json:"sharpness"`
	Contrast  float64 `json:"contrast"`
	Phase     string  `json:"phase"`
}

// Decision is returned for every frame. MoveTo is -1 when no motor command is
// needed. A caller should apply MoveTo before supplying the next frame.
type Decision struct {
	Frame          int         `json:"frame"`
	State          State       `json:"state"`
	Position       int         `json:"position"`
	MoveTo         int         `json:"move_to"`
	Measurement    Measurement `json:"measurement"`
	BestPosition   int         `json:"best_position"`
	BestSharpness  float64     `json:"best_sharpness"`
	LockScore      float64     `json:"lock_score"`
	SubjectPresent bool        `json:"subject_present"`
	Reason         string      `json:"reason"`
}

// Snapshot is a copy of the engine state suitable for telemetry or reports.
type Snapshot struct {
	State          State    `json:"state"`
	Position       int      `json:"position"`
	BestPosition   int      `json:"best_position"`
	BestSharpness  float64  `json:"best_sharpness"`
	LockedPosition int      `json:"locked_position"`
	LockScore      float64  `json:"lock_score"`
	Frame          int      `json:"frame"`
	Moves          int      `json:"moves"`
	Reversals      int      `json:"reversals"`
	MovesAfterLock int      `json:"moves_after_lock"`
	SubjectPresent bool     `json:"subject_present"`
	Samples        []Sample `json:"samples"`
}

// Engine is a contrast-detect autofocus controller. It contains no camera or
// graphics SDK dependency; callers provide luminance images and execute the
// integer motor command returned in Decision.MoveTo.
type Engine struct {
	cfg   Config
	state State
	gate  subjectGate

	frame    int
	position int
	target   int
	settle   int

	samples   []Sample
	best      Sample
	hasBest   bool
	coarseDir int
	coarseEnd int
	belowBest int

	fineNext int
	fineEnd  int
	fineStep int

	confirmLeft  int
	confirmSum   float64
	confirmCount int

	lockedPosition int
	lockScore      float64
	weakFrames     int

	moves          int
	reversals      int
	movesAfterLock int
	lastMoveDir    int
	everLocked     bool
}

// New validates cfg and creates an idle engine.
func New(cfg Config) (*Engine, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	e := &Engine{
		cfg:            cfg,
		state:          StateWaiting,
		target:         -1,
		position:       cfg.MinPosition,
		lockedPosition: -1,
		gate:           newSubjectGate(cfg),
	}
	return e, nil
}

// Update consumes one frame at the supplied current lens position. The caller
// is expected to move the lens to Decision.MoveTo before the next call.
func (e *Engine) Update(img image.Image, currentPosition int) Decision {
	e.frame++
	e.position = currentPosition

	measurement, luma := CalculateMeasurement(img, e.cfg.ROI, e.gate.previous)
	subjectPresent := e.gate.update(measurement.Motion, luma)

	d := Decision{
		Frame:          e.frame,
		State:          e.state,
		Position:       currentPosition,
		MoveTo:         -1,
		Measurement:    measurement,
		BestPosition:   e.best.Position,
		BestSharpness:  e.best.Sharpness,
		LockScore:      e.lockScore,
		SubjectPresent: subjectPresent,
	}

	if e.target >= 0 {
		if currentPosition != e.target {
			d.Reason = fmt.Sprintf("lens moving to %d", e.target)
			return d
		}
		if e.settle > 0 {
			e.settle--
			d.Reason = "settling"
			return d
		}
		e.target = -1
	}

	switch e.state {
	case StateWaiting:
		if subjectPresent {
			e.beginCoarse(currentPosition)
			d = e.commandOrSample(d, currentPosition, "subject acquired; start bounded step search")
		} else {
			d.Reason = "waiting for subject motion"
		}
	case StateCoarse:
		d = e.stepCoarse(d, measurement)
	case StateFine:
		d = e.stepFine(d, measurement)
	case StateLocking:
		d = e.stepLocking(d, measurement)
	case StateLocked:
		d = e.monitorLock(d, measurement, subjectPresent)
	case StateReacquire:
		if subjectPresent {
			e.beginCoarse(currentPosition)
			d.State = e.state
			d = e.commandOrSample(d, currentPosition, "sharpness loss persisted; bounded reacquire")
		} else {
			e.state = StateWaiting
			d.State = e.state
			d.Reason = "subject absent; waiting before reacquire"
		}
	}

	d.State = e.state
	d.Position = currentPosition
	d.BestPosition = e.best.Position
	d.BestSharpness = e.best.Sharpness
	d.LockScore = e.lockScore
	return d
}

// NotifySubject injects an external subject-presence decision. It is useful
// when the application already has a person/object detector and does not want
// to rely only on frame-difference motion. A true value immediately arms the
// gate; false releases it.
func (e *Engine) NotifySubject(present bool) {
	e.gate.present = present
	e.gate.activeRun = 0
	e.gate.quietRun = 0
	if present {
		e.gate.activeRun = e.cfg.PresenceFrames
	}
}

func (e *Engine) commandOrSample(d Decision, current int, reason string) Decision {
	if current != e.coarseEnd {
		return e.moveToward(d, e.coarseEnd, reason)
	}
	d.Reason = reason
	return d
}

func (e *Engine) moveToward(d Decision, target int, reason string) Decision {
	target = clamp(target, e.cfg.MinPosition, e.cfg.MaxPosition)
	if target == e.position {
		return d
	}
	dir := sign(target - e.position)
	if e.lastMoveDir != 0 && dir != e.lastMoveDir {
		e.reversals++
	}
	if e.everLocked {
		e.movesAfterLock++
	}
	e.lastMoveDir = dir
	e.moves++
	e.target = target
	e.settle = e.cfg.SettleFrames
	d.MoveTo = target
	d.Reason = reason
	return d
}

// Snapshot returns an immutable copy of search telemetry.
func (e *Engine) Snapshot() Snapshot {
	samples := make([]Sample, len(e.samples))
	copy(samples, e.samples)
	return Snapshot{
		State:          e.state,
		Position:       e.position,
		BestPosition:   e.best.Position,
		BestSharpness:  e.best.Sharpness,
		LockedPosition: e.lockedPosition,
		LockScore:      e.lockScore,
		Frame:          e.frame,
		Moves:          e.moves,
		Reversals:      e.reversals,
		MovesAfterLock: e.movesAfterLock,
		SubjectPresent: e.gate.present,
		Samples:        samples,
	}
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
