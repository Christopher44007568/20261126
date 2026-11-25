package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	logFile, err := os.Create(filepath.Join(outDir, "run.log"))
	if err != nil {
		log.Fatalf("创建日志失败: %v", err)
	}
	defer logFile.Close()
	logger := log.New(ioMultiWriter(os.Stdout, logFile), "", log.LstdFlags)

	frames := []FrameSpec{
		{Index: 1, Rect: Rect{X0: 0, Y0: 0}}, // 左上四页
		{Index: 2, Rect: Rect{X0: 4, Y0: 4}}, // 跳到右下，与第一帧不相交
		{Index: 3, Rect: Rect{X0: 0, Y0: 0}}, // 回到左上，部分已被淘汰
	}

	sim := NewSimulator()
	failed := false
	for _, spec := range frames {
		res := sim.RunFrame(spec)
		logger.Printf("===== 第 %d 帧 相机视口 页[%d..%d) x [%d..%d) =====",
			spec.Index, spec.Rect.X0, spec.Rect.X0+CameraPages,
			spec.Rect.Y0, spec.Rect.Y0+CameraPages)
		logKinds(logger, res)
		for _, ev := range res.Events {
			logger.Println(DescribeEvent(ev))
		}
		logger.Printf("悬空掩码统计: 冷缺失=%d 驻留命中=%d 悬空=%d",
			countKind(res.Kinds, HitCold),
			countKind(res.Kinds, HitResident),
			countKind(res.Kinds, HitDangling))
		logger.Printf("错误页表危害: 当前视口 %d 个纹素会读到别的虚拟页内容", res.Hazard)

		if msg := sim.Table.CheckInvariants(); msg != "" {
			logger.Printf("不变量校验失败: %s", msg)
			failed = true
		}
		if err := verifyCameraMatchesCanonical(sim, spec.Rect); err != nil {
			logger.Printf("相机内容校验失败: %v", err)
			failed = true
		}

		writeFrameImages(logger, outDir, res)
	}

	if failed {
		logger.Fatal("存在校验失败项，退出码 1")
	}
	logger.Println("全部不变量与内容校验通过：被淘汰内容回视野后已重新加载，未读到其他纹理。")
}

func logKinds(logger *log.Logger, res FrameResult) {
	for _, vp := range pagesInRect(res.Spec.Rect) {
		logger.Printf("  虚拟页(%d,%d) 加载前状态: %s", vp.X, vp.Y, kindName(res.Kinds[vp]))
	}
}

func kindName(k HitKind) string {
	switch k {
	case HitCold:
		return "冷缺失（首次加载）"
	case HitResident:
		return "驻留命中"
	default:
		return "悬空（物理页已被淘汰回收）"
	}
}

func countKind(kinds map[VP]HitKind, want HitKind) int {
	n := 0
	for _, k := range kinds {
		if k == want {
			n++
		}
	}
	return n
}

func writeFrameImages(logger *log.Logger, outDir string, res FrameResult) {
	idx := res.Spec.Index
	outputs := []struct {
		name  string
		img   imageImage
		scale int
	}{
		{fmt.Sprintf("frame%d_camera.png", idx), res.Camera, OutputScale},
		{fmt.Sprintf("frame%d_mask.png", idx), res.Mask, OutputScale},
		{fmt.Sprintf("frame%d_state.png", idx), res.StateGrid, 1},
		{fmt.Sprintf("frame%d_stale_hazard.png", idx), res.StaleCamera, OutputScale},
	}
	for _, o := range outputs {
		if err := savePNG(outDir, o.name, o.img, o.scale); err != nil {
			logger.Printf("写出 %s 失败: %v", o.name, err)
		}
	}
}
