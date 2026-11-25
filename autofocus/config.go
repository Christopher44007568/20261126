package autofocus

import (
	"errors"
	"fmt"
)

// State is the externally visible state of the contrast-detect focus engine.
type State string

const (
	StateWaiting   State = "waiting_for_subject"
	StateCoarse    State = "coarse_search"
	StateFine      State = "fine_search"
	StateLocking   State = "locking"
	StateLocked    State = "locked"
	StateReacquire State = "reacquire"
)

// ROI describes the focus-measurement rectangle as fractions of the image.
// The default is a centered portrait rectangle.
type ROI struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Config controls both the AF search and the subject-presence gate.
//
// Positions are arbitrary integer lens steps. The defaults target a 0..64
// motor range, but MinPosition and MaxPosition are always honored.
type Config struct {
	MinPosition int `json:"min_position"`
	MaxPosition int `json:"max_position"`

	// CoarseStep is the initial hill-climb pitch. FineStep is the one-pass
	// refinement pitch. FineStep must be smaller than CoarseStep.
	CoarseStep int `json:"coarse_step"`
	FineStep   int `json:"fine_step"`

	// SettleFrames are discarded after the lens reaches a new target.
	SettleFrames int `json:"settle_frames"`

	// LockConfirmFrames validates the selected position before locking.
	LockConfirmFrames int `json:"lock_confirm_frames"`

	// LockHysteresis is the fractional sharpness drop tolerated while locked.
	// A value of 0.18 means a score down to 82% of the lock score is ignored.
	LockHysteresis float64 `json:"lock_hysteresis"`

	// UnlockFrames is the number of consecutive weak frames required before
	// a locked lens is allowed to search again.
	UnlockFrames int `json:"unlock_frames"`

	// PresenceMotion is the mean absolute luminance delta (0..1) that counts
	// as subject motion. PresenceFrames consecutive active frames arm a search.
	PresenceMotion float64 `json:"presence_motion"`
	PresenceFrames int     `json:"presence_frames"`
	AbsenceFrames  int     `json:"absence_frames"`

	// FineWindowSteps sizes the local fine-search bracket. The trailing side
	// spans this many coarse steps beyond the peak; the leading side spans half
	// as much. This bounds overshoot and prevents a second full sweep.
	FineWindowSteps int `json:"fine_window_steps"`

	ROI ROI `json:"roi"`
}

// DefaultConfig returns conservative, non-oscillating settings for a
// 0..64 step lens and a 50% center AF area.
func DefaultConfig() Config {
	return Config{
		MinPosition:       0,
		MaxPosition:       64,
		CoarseStep:        8,
		FineStep:          1,
		SettleFrames:      1,
		LockConfirmFrames: 2,
		LockHysteresis:    0.18,
		UnlockFrames:      8,
		PresenceMotion:    0.012,
		PresenceFrames:    2,
		AbsenceFrames:     12,
		FineWindowSteps:   1,
		ROI:               ROI{X: 0.25, Y: 0.20, W: 0.50, H: 0.60},
	}
}

func (c Config) validate() error {
	if c.MaxPosition <= c.MinPosition {
		return errors.New("max position must be greater than min position")
	}
	if c.CoarseStep <= 0 || c.FineStep <= 0 {
		return errors.New("search steps must be positive")
	}
	if c.FineStep >= c.CoarseStep {
		return errors.New("fine step must be smaller than coarse step")
	}
	if c.SettleFrames < 0 || c.LockConfirmFrames < 1 || c.UnlockFrames < 1 {
		return errors.New("frame counts are invalid")
	}
	if c.LockHysteresis < 0 || c.LockHysteresis >= 1 {
		return errors.New("lock hysteresis must be in [0, 1)")
	}
	if c.PresenceFrames < 1 || c.AbsenceFrames < 1 {
		return errors.New("presence frame counts must be positive")
	}
	if c.PresenceMotion <= 0 {
		return errors.New("presence motion threshold must be positive")
	}
	if c.FineWindowSteps < 1 {
		return errors.New("fine window must contain at least one coarse step")
	}
	if c.ROI.W <= 0 || c.ROI.H <= 0 || c.ROI.X < 0 || c.ROI.Y < 0 ||
		c.ROI.X+c.ROI.W > 1 || c.ROI.Y+c.ROI.H > 1 {
		return fmt.Errorf("ROI %+v must lie inside the image", c.ROI)
	}
	return nil
}
