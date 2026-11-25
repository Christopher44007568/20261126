package autofocus

// subjectGate turns frame-to-frame ROI luminance motion into a hysteretic
// presence signal. Blur changes from focusing are tolerated while the signal
// is latched; a quiet period is required before the subject can be considered
// absent and a new acquisition can be armed.
type subjectGate struct {
	cfg       Config
	previous  []float64
	present   bool
	activeRun int
	quietRun  int
}

func newSubjectGate(cfg Config) subjectGate {
	return subjectGate{cfg: cfg}
}

func (g *subjectGate) update(motion float64, luma []float64) bool {
	if motion >= g.cfg.PresenceMotion {
		g.activeRun++
		g.quietRun = 0
		if g.activeRun >= g.cfg.PresenceFrames {
			g.present = true
		}
	} else {
		g.quietRun++
		g.activeRun = 0
		if g.quietRun >= g.cfg.AbsenceFrames {
			g.present = false
		}
	}
	g.previous = append(g.previous[:0], luma...)
	return g.present
}
