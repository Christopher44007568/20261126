package autofocus

// beginCoarse selects the nearest motor endpoint and then scans strictly in
// one direction. Starting at an endpoint removes the classic overshoot loop:
// there is no full-range back-and-forth sweep.
func (e *Engine) beginCoarse(current int) {
	e.samples = e.samples[:0]
	e.best = Sample{}
	e.hasBest = false
	e.belowBest = 0
	e.coarseDir = 0

	toMin := current - e.cfg.MinPosition
	toMax := e.cfg.MaxPosition - current
	if toMin <= toMax {
		e.coarseEnd = e.cfg.MinPosition
		e.coarseDir = 1
	} else {
		e.coarseEnd = e.cfg.MaxPosition
		e.coarseDir = -1
	}
	e.state = StateCoarse
}

func (e *Engine) stepCoarse(d Decision, m Measurement) Decision {
	s := e.recordSample(m, "coarse")
	e.observeBest(s)

	if e.belowBest >= 2 || e.atCoarseBoundary() {
		e.beginFine()
		d.State = e.state
		return e.stepFine(d, m)
	}

	next := e.position + e.coarseDir*e.cfg.CoarseStep
	next = clamp(next, e.cfg.MinPosition, e.cfg.MaxPosition)
	if next == e.position {
		e.beginFine()
		d.State = e.state
		return e.stepFine(d, m)
	}
	return e.moveToward(d, next, "monotonic coarse sweep")
}

func (e *Engine) atCoarseBoundary() bool {
	next := e.position + e.coarseDir*e.cfg.CoarseStep
	return next < e.cfg.MinPosition || next > e.cfg.MaxPosition
}

func (e *Engine) beginFine() {
	trail := e.cfg.FineWindowSteps * e.cfg.CoarseStep
	lead := trail / 2
	if lead < e.cfg.FineStep {
		lead = e.cfg.FineStep
	}
	if e.coarseDir < 0 {
		e.fineNext = clamp(e.best.Position-trail, e.cfg.MinPosition, e.cfg.MaxPosition)
		e.fineEnd = clamp(e.best.Position+lead, e.cfg.MinPosition, e.cfg.MaxPosition)
		e.fineStep = e.cfg.FineStep
	} else {
		e.fineNext = clamp(e.best.Position+trail, e.cfg.MinPosition, e.cfg.MaxPosition)
		e.fineEnd = clamp(e.best.Position-lead, e.cfg.MinPosition, e.cfg.MaxPosition)
		e.fineStep = -e.cfg.FineStep
	}
	e.state = StateFine
}

func (e *Engine) stepFine(d Decision, m Measurement) Decision {
	if e.position != e.fineNext {
		return e.moveToward(d, e.fineNext, "single-pass fine refinement")
	}

	s := e.recordSample(m, "fine")
	e.observeBest(s)
	e.fineNext += e.fineStep

	inRange := e.fineStep > 0 && e.fineNext <= e.fineEnd
	inRange = inRange || (e.fineStep < 0 && e.fineNext >= e.fineEnd)
	if inRange {
		return e.moveToward(d, e.fineNext, "single-pass fine refinement")
	}
	e.beginLocking()
	d.State = e.state
	if e.position != e.best.Position {
		return e.moveToward(d, e.best.Position, "return to best focus once")
	}
	return d
}

func (e *Engine) beginLocking() {
	e.state = StateLocking
	e.confirmLeft = e.cfg.LockConfirmFrames
	e.confirmSum = 0
	e.confirmCount = 0
}

func (e *Engine) stepLocking(d Decision, m Measurement) Decision {
	if e.position != e.best.Position {
		return e.moveToward(d, e.best.Position, "move to focus candidate")
	}

	e.recordSample(m, "confirm")
	e.confirmSum += m.Sharpness
	e.confirmCount++
	e.confirmLeft--
	if e.confirmLeft > 0 {
		d.Reason = "confirming best position"
		return d
	}

	e.lockScore = e.confirmSum / float64(e.confirmCount)
	e.lockedPosition = e.best.Position
	e.everLocked = true
	e.weakFrames = 0
	e.state = StateLocked
	d.Reason = "focus locked; hysteresis active"
	return d
}

func (e *Engine) monitorLock(d Decision, m Measurement, subjectPresent bool) Decision {
	floor := e.lockScore * (1 - e.cfg.LockHysteresis)
	if e.lockScore > 0 && m.Sharpness < floor {
		e.weakFrames++
	} else {
		e.weakFrames = 0
	}

	if e.weakFrames < e.cfg.UnlockFrames {
		d.Reason = "locked; holding position"
		return d
	}
	e.weakFrames = 0
	if subjectPresent {
		e.state = StateReacquire
		d.Reason = "persistent sharpness loss; reacquire requested"
	} else {
		e.state = StateWaiting
		d.Reason = "subject absent; lock released"
	}
	return d
}

func (e *Engine) recordSample(m Measurement, phase string) Sample {
	s := Sample{
		Frame:     e.frame,
		Position:  e.position,
		Sharpness: m.Sharpness,
		Contrast:  m.Contrast,
		Phase:     phase,
	}
	e.samples = append(e.samples, s)
	return s
}

func (e *Engine) observeBest(s Sample) {
	if !e.hasBest || s.Sharpness > e.best.Sharpness {
		e.best = s
		e.hasBest = true
		e.belowBest = 0
		return
	}
	if e.best.Sharpness > 0 && s.Sharpness < e.best.Sharpness*0.97 {
		e.belowBest++
	} else {
		e.belowBest = 0
	}
}
