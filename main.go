package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := run(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "lazyscene:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	root := BuildScene()
	leaves := root.Leaves()

	// 远端服务：T1 最快，其余较慢，从而稳定捕获“加载中”中间态。
	store := &RemoteStore{
		Default: 120 * time.Millisecond,
		Latency: map[string]time.Duration{
			"T1": 25 * time.Millisecond,
			"T2": 200 * time.Millisecond,
			"T3": 300 * time.Millisecond,
			"T4": 400 * time.Millisecond,
		},
	}
	loader := NewLoader(store)
	for _, n := range leaves {
		loader.Register(n.ID, n.Seed)
	}
	renderer := NewRenderer(loader)

	// 帧 1：先卸载远端 payload（此时缓存为空），再尝试渲染。
	loader.UnloadAll()
	frameUnloaded := renderer.Render(root, "FRAME 1 UNLOADED REMOTE PAYLOAD EVICTED")
	if err := writePPM(filepath.Join(outDir, "frame1_unloaded.ppm"), frameUnloaded); err != nil {
		return err
	}

	// 按需加载：对视口内子树发起请求（已在途/已缓存的不会重复请求）。
	for _, n := range leaves {
		loader.EnsureLoaded(n.ID)
	}

	// 帧 2：等待 T1 完成而其余仍在途，得到加载中与完整并存的一帧。
	if !waitFor(loader, func(s map[string]Status) bool {
		return s["T1"] == StatusLoaded &&
			s["T2"] == StatusLoading &&
			s["T3"] == StatusLoading &&
			s["T4"] == StatusLoading
	}, time.Second) {
		return fmt.Errorf("未能在超时内捕获加载中状态: %v", loader.Snapshot())
	}
	frameLoading := renderer.Render(root, "FRAME 2 LOADING ON DEMAND FETCH IN FLIGHT")
	if err := writePPM(filepath.Join(outDir, "frame2_loading.ppm"), frameLoading); err != nil {
		return err
	}

	// 帧 3：所有远端请求结算，缓存命中后渲染完整内容。
	loader.Wait()
	frameFull := renderer.Render(root, "FRAME 3 LOADED CACHED PAYLOAD")
	if err := writePPM(filepath.Join(outDir, "frame3_loaded.ppm"), frameFull); err != nil {
		return err
	}

	// 切回已卸载区域：再次卸载全部 payload 后用全新画布渲染。
	loader.UnloadAll()
	frameRevisit := renderer.Render(root, "REVISIT UNLOADED AGAIN NO STALE PIXELS")

	report := verify(root, loader, frameUnloaded, frameLoading, frameFull, frameRevisit)
	fmt.Print(report)
	return nil
}

// waitFor 轮询状态快照直到 cond 成立或超时。
func waitFor(loader *Loader, cond func(map[string]Status) bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond(loader.Snapshot()) {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return cond(loader.Snapshot())
}

func writePPM(path string, im *Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return im.WritePPM(f)
}
