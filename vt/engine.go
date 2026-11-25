package vt

import (
	"context"
	"time"
)

// Engine owns the virtual texturing feedback/update/render pipeline.
type Engine struct {
	Info           Info
	Table          *Table
	Scheduler      *Scheduler
	Loader         *AsyncLoader
	FeedbackStride int
	Feedback       *FeedbackBuffer
}

func NewEngine(info Info, table *Table, scheduler *Scheduler, loader *AsyncLoader, feedbackStride int) *Engine {
	if feedbackStride < 1 {
		feedbackStride = 8
	}
	return &Engine{
		Info:           info,
		Table:          table,
		Scheduler:      scheduler,
		Loader:         loader,
		FeedbackStride: feedbackStride,
	}
}

// Update builds UV feedback and submits all newly needed page requests.
func (e *Engine) Update(cam Camera, width, height int) *FeedbackBuffer {
	fb := BuildFeedback(cam, width, height, e.FeedbackStride, e.Info)
	e.Feedback = fb
	requests := e.Scheduler.Plan(fb, e.Table)
	// Keep every resident member of the current visible fallback chain out of
	// the next eviction batch. Missing IDs are ignored by Touch.
	e.Table.Touch(e.Scheduler.WantedIDs())
	for _, req := range requests {
		if e.Loader == nil {
			e.Scheduler.Finish(req.ID)
			continue
		}
		if !e.Loader.Submit(req) {
			e.Scheduler.Finish(req.ID)
		}
	}
	return fb
}

type PollReport struct {
	Applied      int
	StaleDiscard int
	Errors       int
	Evicted      []PageID
}

// Poll applies all currently available load results without blocking.
func (e *Engine) Poll() PollReport {
	var report PollReport
	if e.Loader == nil {
		return report
	}
	for {
		select {
		case result, ok := <-e.Loader.Results():
			if !ok {
				return report
			}
			e.Scheduler.Finish(result.Request.ID)
			if result.Err != nil || result.Page == nil {
				report.Errors++
				continue
			}
			if !e.Scheduler.Accept(result.Request.ID) {
				report.StaleDiscard++
				continue
			}
			evicted, err := e.Table.Install(result.Page)
			if err != nil {
				report.Errors++
				continue
			}
			report.Applied++
			if evicted != nil {
				report.Evicted = append(report.Evicted, evicted.ID)
			}
		default:
			return report
		}
	}
}

// WaitUntil polls completions until condition is true or ctx expires.
func (e *Engine) WaitUntil(ctx context.Context, condition func() bool) error {
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	for {
		e.Poll()
		if condition() {
			return nil
		}
		select {
		case <-ctx.Done():
			e.Poll()
			if condition() {
				return nil
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Idle reports that every scheduled request has produced a result.
func (e *Engine) Idle() bool {
	e.Poll()
	return e.Scheduler.InflightCount() == 0
}

// Render is a thin wrapper preserving the engine's info/table dependencies.
func (e *Engine) Render(cam Camera, width, height int) *Frame {
	return Render(cam, e.Info, e.Table, width, height)
}
