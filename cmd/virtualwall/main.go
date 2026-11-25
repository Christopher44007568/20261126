package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	vt "virtualwall/vt"
)

type config struct {
	out     string
	width   int
	height  int
	cache   int
	stride  int
	workers int
	delay   time.Duration
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.out, "out", "artifacts/virtualwall", "artifact output directory")
	flag.IntVar(&cfg.width, "width", 600, "camera frame width")
	flag.IntVar(&cfg.height, "height", 400, "camera frame height")
	flag.IntVar(&cfg.cache, "cache", 12, "physical page cache capacity")
	flag.IntVar(&cfg.stride, "feedback-stride", 8, "UV feedback block stride in pixels")
	flag.IntVar(&cfg.workers, "workers", 4, "async loader workers")
	flag.DurationVar(&cfg.delay, "load-delay", 8*time.Millisecond, "simulated page load delay")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "virtualwall:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	info := vt.DefaultInfo()
	texture := vt.NewProceduralTexture(info)
	table := vt.NewTable(cfg.cache)
	scheduler := vt.NewScheduler(info)

	// Tokens make the asynchronous sequence deterministic for artifacts:
	// drain after warm-up, permit two high-detail pages for the partial mask,
	// then permit all remaining pages.
	mip0Tokens := make(chan struct{}, 64)
	for i := 0; i < cap(mip0Tokens); i++ {
		mip0Tokens <- struct{}{}
	}
	load := func(loadCtx context.Context, id vt.PageID) (*vt.Page, error) {
		if id.Mip == 0 {
			select {
			case <-mip0Tokens:
			case <-loadCtx.Done():
				return nil, loadCtx.Err()
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		select {
		case <-time.After(cfg.delay):
		case <-loadCtx.Done():
			return nil, loadCtx.Err()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return texture.LoadPage(id)
	}
	loader := vt.NewAsyncLoader(ctx, cfg.workers, 128, load)
	defer loader.Close()
	engine := vt.NewEngine(info, table, scheduler, loader, cfg.stride)

	if err := os.MkdirAll(cfg.out, 0o755); err != nil {
		return err
	}

	// Width UV = 0.375 (three mip0 pages), height UV = 0.25 (two pages).
	aspect := float64(cfg.width) / float64(cfg.height)
	viewHeight := 0.25
	initial := vt.NewCameraWithViewHeight(0.3125, 0.25, viewHeight).ClampToWall(aspect)
	moved := vt.NewCameraWithViewHeight(0.6875, 0.75, viewHeight).ClampToWall(aspect)

	var points []vt.MetricPoint
	checks := []verificationCheck{}

	engine.Update(initial, cfg.width, cfg.height)
	if err := engine.WaitUntil(ctx, engine.Idle); err != nil {
		return fmt.Errorf("warm-up load: %w", err)
	}
	engine.Poll()
	frame := engine.Render(initial, cfg.width, cfg.height)
	if err := saveStage(cfg, texture, engine, "01_settled", frame, engine.Feedback, &points, &checks); err != nil {
		return err
	}

	drainTokens(mip0Tokens)
	engine.Update(moved, cfg.width, cfg.height)
	if err := engine.WaitUntil(ctx, func() bool { return ancestorsResident(info, table, engine.Feedback) }); err != nil {
		return fmt.Errorf("ancestor load: %w", err)
	}
	engine.Poll()
	parentFrame := engine.Render(moved, cfg.width, cfg.height)
	if err := saveStage(cfg, texture, engine, "02_parent_fallback", parentFrame, engine.Feedback, &points, &checks); err != nil {
		return err
	}

	mip0Tokens <- struct{}{}
	mip0Tokens <- struct{}{}
	if err := engine.WaitUntil(ctx, func() bool { return exactCount(table, engine.Feedback) >= 2 }); err != nil {
		return fmt.Errorf("partial load: %w", err)
	}
	engine.Poll()
	partialFrame := engine.Render(moved, cfg.width, cfg.height)
	if err := saveStage(cfg, texture, engine, "03_partial", partialFrame, engine.Feedback, &points, &checks); err != nil {
		return err
	}

	for i := 0; i < 16; i++ {
		select {
		case mip0Tokens <- struct{}{}:
		default:
		}
	}
	if err := engine.WaitUntil(ctx, func() bool {
		return exactCount(table, engine.Feedback) == len(engine.Feedback.Counts)
	}); err != nil {
		return fmt.Errorf("full load: %w", err)
	}
	engine.Poll()
	completeFrame := engine.Render(moved, cfg.width, cfg.height)
	if err := saveStage(cfg, texture, engine, "04_complete", completeFrame, engine.Feedback, &points, &checks); err != nil {
		return err
	}

	if err := writeCanonicalAndReports(cfg, engine, points, checks); err != nil {
		return err
	}
	fmt.Printf("virtualwall artifacts written to %s\n", cfg.out)
	fmt.Printf("verified %d frames; final resident=%d/%d evictions=%d\n",
		len(checks), table.Count(), cfg.cache, table.EvictionCount())
	return nil
}

func drainTokens(ch chan struct{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func ancestorsResident(info vt.Info, table *vt.Table, fb *vt.FeedbackBuffer) bool {
	snapshot := table.Snapshot()
	for exact := range fb.Counts {
		chain := info.Chain(exact)
		for _, ancestor := range chain[1:] {
			if _, ok := snapshot[ancestor]; !ok {
				return false
			}
		}
	}
	return true
}

func exactCount(table *vt.Table, fb *vt.FeedbackBuffer) int {
	snapshot := table.Snapshot()
	count := 0
	for id := range fb.Counts {
		if _, ok := snapshot[id]; ok {
			count++
		}
	}
	return count
}
