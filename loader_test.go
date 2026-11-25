package main

import (
	"sync/atomic"
	"testing"
	"time"
)

// fakeFetcher 统计远端调用次数，并用可控阻塞模拟在途请求。
type fakeFetcher struct {
	calls atomic.Int32
	block chan struct{}
}

func (f *fakeFetcher) Fetch(id string, seed uint32) (*Payload, error) {
	f.calls.Add(1)
	if f.block != nil {
		<-f.block
	}
	return generatePayload(id, seed), nil
}

func TestOnDemandLoadAndDedup(t *testing.T) {
	f := &fakeFetcher{}
	l := NewLoader(f)
	l.Register("T1", 7)

	if got := l.Status("T1"); got != StatusUnloaded {
		t.Fatalf("初始状态应为 UNLOADED，得到 %v", got)
	}
	l.EnsureLoaded("T1")
	l.EnsureLoaded("T1") // 加载中重复请求不应再次发起
	l.Wait()

	if got := f.calls.Load(); got != 1 {
		t.Fatalf("去重失败：远端被调用 %d 次，期望 1", got)
	}
	if l.Status("T1") != StatusLoaded || l.Cached("T1") == nil {
		t.Fatalf("加载后应为 LOADED 且缓存非空")
	}

	// 已缓存时再次 EnsureLoaded 必须命中缓存，不访问远端。
	l.EnsureLoaded("T1")
	if got := f.calls.Load(); got != 1 {
		t.Fatalf("缓存命中不应访问远端，调用数=%d", got)
	}
}

func TestUnloadInvalidatesInFlight(t *testing.T) {
	block := make(chan struct{})
	f := &fakeFetcher{block: block}
	l := NewLoader(f)
	l.Register("T1", 7)

	l.EnsureLoaded("T1")
	// 等待 goroutine 进入阻塞，确认请求在途。
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && f.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	if l.Status("T1") != StatusLoading {
		t.Fatalf("请求应处于 LOADING，得到 %v", l.Status("T1"))
	}

	// 在途期间卸载：状态立即回到未加载且缓存为空。
	l.Unload("T1")
	if l.Status("T1") != StatusUnloaded || l.Cached("T1") != nil {
		t.Fatalf("卸载后应为 UNLOADED 且无缓存")
	}
	close(block) // 放行在途请求
	l.Wait()

	// 在途结果因 generation 失效，不得写回缓存。
	if l.Status("T1") != StatusUnloaded || l.Cached("T1") != nil {
		t.Fatalf("过期的在途结果不得写回缓存，状态=%v", l.Status("T1"))
	}
}

func TestUnloadAllAndReload(t *testing.T) {
	f := &fakeFetcher{}
	l := NewLoader(f)
	l.Register("T1", 1)
	l.Register("T2", 2)
	l.EnsureLoaded("T1")
	l.EnsureLoaded("T2")
	l.Wait()

	l.UnloadAll()
	for _, id := range []string{"T1", "T2"} {
		if l.Status(id) != StatusUnloaded || l.Cached(id) != nil {
			t.Fatalf("%s 卸载后应为空", id)
		}
	}

	// 重新按需加载应产生新的远端调用并恢复完整状态。
	l.EnsureLoaded("T1")
	l.Wait()
	if l.Status("T1") != StatusLoaded {
		t.Fatalf("重新加载后应为 LOADED")
	}
	if got := f.calls.Load(); got != 3 {
		t.Fatalf("重新加载应新增一次远端调用，总调用数=%d，期望 3", got)
	}
}
