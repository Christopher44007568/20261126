package vt

import (
	"context"
	"sync"
)

// LoadRequest is a prioritized asynchronous page load.
type LoadRequest struct {
	ID         PageID
	Priority   int
	Generation uint64
	Reason     string
}

// LoadResult carries either a freshly loaded page or an error.
type LoadResult struct {
	Request LoadRequest
	Page    *Page
	Err     error
}

// LoadFunc generates/reads one page. It receives context so demos can gate
// high-resolution loads without sleeping through shutdown.
type LoadFunc func(ctx context.Context, id PageID) (*Page, error)

// AsyncLoader is a fixed-size worker pool with bounded request/result queues.
type AsyncLoader struct {
	jobs    chan LoadRequest
	results chan LoadResult
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	load    LoadFunc
	closed  bool
	mu      sync.Mutex
}

func NewAsyncLoader(parent context.Context, workers, queueSize int, load LoadFunc) *AsyncLoader {
	if workers < 1 {
		workers = 1
	}
	if queueSize < workers {
		queueSize = workers
	}
	ctx, cancel := context.WithCancel(parent)
	l := &AsyncLoader{
		jobs:    make(chan LoadRequest, queueSize),
		results: make(chan LoadResult, queueSize),
		ctx:     ctx,
		cancel:  cancel,
		load:    load,
	}
	for i := 0; i < workers; i++ {
		l.wg.Add(1)
		go l.worker()
	}
	go func() {
		l.wg.Wait()
		close(l.results)
	}()
	return l
}

func (l *AsyncLoader) worker() {
	defer l.wg.Done()
	for {
		select {
		case <-l.ctx.Done():
			return
		case req := <-l.jobs:
			page, err := l.load(l.ctx, req.ID)
			result := LoadResult{Request: req, Page: page, Err: err}
			select {
			case l.results <- result:
			case <-l.ctx.Done():
				return
			}
		}
	}
}

// Submit enqueues one request. It returns false when the bounded queue is full.
func (l *AsyncLoader) Submit(req LoadRequest) bool {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()
	if closed {
		return false
	}
	select {
	case <-l.ctx.Done():
		return false
	case l.jobs <- req:
		return true
	default:
		return false
	}
}

func (l *AsyncLoader) Results() <-chan LoadResult { return l.results }

// Close cancels blocked work and closes the result channel after workers exit.
func (l *AsyncLoader) Close() {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.closed = true
	l.mu.Unlock()
	l.cancel()
	l.wg.Wait()
}
