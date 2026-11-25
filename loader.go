package main

import (
	"sync"
	"time"
)

// Status 是单个子树 payload 的加载状态。
type Status int

const (
	StatusUnloaded Status = iota // 未加载：缓存中没有任何 payload
	StatusLoading                // 加载中：已发起远端请求，尚未完成
	StatusLoaded                 // 完整：payload 已在缓存中
)

func (s Status) String() string {
	switch s {
	case StatusUnloaded:
		return "UNLOADED"
	case StatusLoading:
		return "LOADING"
	default:
		return "LOADED"
	}
}

// Payload 是从“远端”取回的子树内容。生成成本高、体积大，
// 因此需要按需加载、缓存并在离开视口时卸载。
type Payload struct {
	SubtreeID string
	Seed      uint32
	// Rows 为确定性生成的图案行数据，渲染器据此绘制完整内容。
	Rows []string
}

// Fetcher 抽象远端数据源，便于在测试中注入假实现。
type Fetcher interface {
	Fetch(id string, seed uint32) (*Payload, error)
}

// RemoteStore 模拟带网络延迟的远端 payload 服务。
// 不持有任何客户端缓存；延迟与内容均为确定性。
type RemoteStore struct {
	Latency map[string]time.Duration
	Default time.Duration
}

func (s *RemoteStore) Fetch(id string, seed uint32) (*Payload, error) {
	d, ok := s.Latency[id]
	if !ok {
		d = s.Default
	}
	time.Sleep(d)
	return generatePayload(id, seed), nil
}

type entry struct {
	status     Status
	payload    *Payload
	generation uint64 // 每次请求递增；卸载会使在途结果失效
}

// Loader 管理所有子树 payload 的按需加载、缓存与卸载。
// 渲染器只能通过它读取状态/缓存，本身不保存像素。
type Loader struct {
	fetcher Fetcher
	mu      sync.Mutex
	entries map[string]*entry
	seeds   map[string]uint32
	wg      sync.WaitGroup
}

func NewLoader(f Fetcher) *Loader {
	return &Loader{
		fetcher: f,
		entries: make(map[string]*entry),
		seeds:   make(map[string]uint32),
	}
}

// Register 登记子树及其内容种子，初始为未加载。
func (l *Loader) Register(id string, seed uint32) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[id] = &entry{status: StatusUnloaded}
	l.seeds[id] = seed
}

// Status 返回当前加载状态。
func (l *Loader) Status(id string) Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.entries[id].status
}

// Cached 返回缓存中的完整 payload；未加载或加载中返回 nil。
func (l *Loader) Cached(id string) *Payload {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries[id].status == StatusLoaded {
		return l.entries[id].payload
	}
	return nil
}

// EnsureLoaded 按需触发加载：已加载或加载中均不重复发起请求。
func (l *Loader) EnsureLoaded(id string) {
	l.mu.Lock()
	e := l.entries[id]
	if e.status != StatusUnloaded {
		l.mu.Unlock()
		return
	}
	e.generation++
	gen := e.generation
	seed := l.seeds[id]
	e.status = StatusLoading
	l.mu.Unlock()

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		p, err := l.fetcher.Fetch(id, seed)
		if err != nil {
			l.mu.Lock()
			if e.generation == gen {
				e.status = StatusUnloaded
			}
			l.mu.Unlock()
			return
		}
		l.mu.Lock()
		// 仅当请求未被卸载/重新发起取代时才写入缓存。
		if e.generation == gen {
			e.payload = p
			e.status = StatusLoaded
		}
		l.mu.Unlock()
	}()
}

// Unload 卸载单个子树：清空缓存并使任何在途结果失效。
func (l *Loader) Unload(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[id]
	e.generation++
	e.payload = nil
	e.status = StatusUnloaded
}

// UnloadAll 卸载全部子树（模拟离开整个场景视口）。
func (l *Loader) UnloadAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.entries {
		e.generation++
		e.payload = nil
		e.status = StatusUnloaded
	}
}

// Wait 阻塞直到当前所有在途请求结算。
func (l *Loader) Wait() { l.wg.Wait() }

// Snapshot 返回所有子树状态的只读副本。
func (l *Loader) Snapshot() map[string]Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]Status, len(l.entries))
	for id, e := range l.entries {
		out[id] = e.status
	}
	return out
}
