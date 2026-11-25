package main

import (
	"fmt"
	"strings"
)

// verify 对三帧与“切回”帧做状态与像素级断言，返回人类可读报告。
func verify(
	root *Node,
	loader *Loader,
	unloaded, loading, full, revisit *Image,
) string {
	var b strings.Builder
	pass := true

	check := func(name string, ok bool, detail string) {
		mark := "PASS"
		if !ok {
			mark = "FAIL"
			pass = false
		}
		fmt.Fprintf(&b, "[%s] %s %s\n", mark, name, detail)
	}

	leaves := root.Leaves()

	// 帧 1：所有子树必须处于未加载，缓存为空。
	allUnloaded := true
	for _, n := range leaves {
		if loader.Status(n.ID) != StatusUnloaded || loader.Cached(n.ID) != nil {
			// 注意：此检查在最终卸载后执行，覆盖“卸载清空缓存”。
			allUnloaded = false
		}
	}
	check("unload clears cache", allUnloaded, "all subtrees back to UNLOADED with nil payload")

	// 完整帧与切回帧的每个子树区域像素必须不同：没有复用旧像素。
	reused := 0
	var diffs []string
	for _, n := range leaves {
		hFull := full.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		hRevisit := revisit.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		hUnloaded := unloaded.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		if hFull == hRevisit {
			reused++
		}
		diffs = append(diffs, fmt.Sprintf("%s(full=%016x revisit=%016x unloaded=%016x)",
			n.ID, hFull, hRevisit, hUnloaded))
	}
	check("no stale pixels on revisit", reused == 0,
		"region hashes differ from fully-loaded frame: "+strings.Join(diffs, " "))

	// 切回帧与首次未加载帧在同一子树区域应一致（都走占位渲染路径）。
	samePlaceholder := true
	for _, n := range leaves {
		if revisit.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H) !=
			unloaded.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H) {
			samePlaceholder = false
		}
	}
	check("revisit renders placeholder", samePlaceholder,
		"revisit frame matches first UNLOADED placeholder pixels")

	// 帧尺寸与非空像素校验。
	check("frame geometry", full.W == FrameW && full.H == FrameH,
		fmt.Sprintf("%dx%d PPM frames", FrameW, FrameH))

	// 加载中帧与完整帧必须存在差异（确实捕获了中间态）。
	check("loading frame distinct", loading.RegionHash(0, 0, FrameW, FrameH) !=
		full.RegionHash(0, 0, FrameW, FrameH), "in-flight frame differs from complete frame")

	if pass {
		b.WriteString("RESULT: ALL CHECKS PASSED\n")
	} else {
		b.WriteString("RESULT: FAILURES DETECTED\n")
	}
	return b.String()
}
