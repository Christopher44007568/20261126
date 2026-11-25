package main

import (
	"bytes"
	"strings"
	"testing"
)

func newTestLoader() *Loader {
	l := NewLoader(&fakeFetcher{})
	for _, n := range BuildScene().Leaves() {
		l.Register(n.ID, n.Seed)
	}
	return l
}

func TestRenderThreeStates(t *testing.T) {
	root := BuildScene()
	l := newTestLoader()
	r := NewRenderer(l)

	unloaded := r.Render(root, "U")
	for _, n := range root.Leaves() {
		if l.Status(n.ID) != StatusUnloaded {
			t.Fatalf("%s 初始应为 UNLOADED", n.ID)
		}
	}

	for _, n := range root.Leaves() {
		l.EnsureLoaded(n.ID)
	}
	l.Wait()
	full := r.Render(root, "F")

	// 完整帧与未加载帧在每个子树区域都必须不同。
	for _, n := range root.Leaves() {
		hu := unloaded.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		hf := full.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		if hu == hf {
			t.Fatalf("%s 未加载与完整帧像素不应相同", n.ID)
		}
	}
}

func TestRevisitAfterUnloadHasNoStalePixels(t *testing.T) {
	root := BuildScene()
	l := newTestLoader()
	r := NewRenderer(l)

	for _, n := range root.Leaves() {
		l.EnsureLoaded(n.ID)
	}
	l.Wait()
	full := r.Render(root, "FULL")

	// 离开视口触发卸载，切回时重新渲染。
	l.UnloadAll()
	revisit := r.Render(root, "REVISIT")

	for _, n := range root.Leaves() {
		hf := full.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		hr := revisit.RegionHash(n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H)
		if hf == hr {
			t.Fatalf("%s 切回已卸载区域时复用了旧像素", n.ID)
		}
		if l.Cached(n.ID) != nil {
			t.Fatalf("%s 卸载后缓存应清空", n.ID)
		}
	}
}

func TestPPMWriter(t *testing.T) {
	im := NewImage(4, 2)
	im.Set(0, 0, RGB(1, 2, 3))
	var buf bytes.Buffer
	if err := im.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "P6\n") {
		t.Fatalf("PPM 必须以 P6 魔数开头，得到 %q", buf.String()[:4])
	}
	// 头 3 行 + 4*2*3 像素字节。
	if got := buf.Len(); got < 24 {
		t.Fatalf("PPM 字节数异常: %d", got)
	}
}
