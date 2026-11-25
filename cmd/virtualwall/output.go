package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vt "virtualwall/vt"
)

type verificationCheck struct {
	Stage string `json:"stage"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func saveStage(
	cfg config,
	texture *vt.Texture,
	engine *vt.Engine,
	stage string,
	frame *vt.Frame,
	fb *vt.FeedbackBuffer,
	points *[]vt.MetricPoint,
	checks *[]verificationCheck,
) error {
	out := cfg.out
	table := engine.Table
	info := engine.Info

	cameraPath := filepath.Join(out, "camera_"+stage+".png")
	maskPath := filepath.Join(out, "miss_mask_"+stage+".png")
	statusPath := filepath.Join(out, "status_mask_"+stage+".png")
	residencyPath := filepath.Join(out, "residency_"+stage+".png")
	if err := vt.WritePNG(cameraPath, frame.Image); err != nil {
		return err
	}
	if err := vt.WritePNG(maskPath, frame.MissMask); err != nil {
		return err
	}
	if err := vt.WritePNG(statusPath, frame.StatusMask); err != nil {
		return err
	}
	residency := vt.RenderResidencyMap(info, table.Snapshot(), fb.Counts, 28)
	if err := vt.WritePNG(residencyPath, residency); err != nil {
		return err
	}

	if stage == "02_parent_fallback" {
		feedback := vt.RenderFeedback(fb, 4)
		if err := vt.WritePNG(filepath.Join(out, "uv_feedback.png"), feedback); err != nil {
			return err
		}
		if err := vt.WritePNG(filepath.Join(out, "camera_view.png"), frame.Image); err != nil {
			return err
		}
		if err := vt.WritePNG(filepath.Join(out, "miss_mask.png"), frame.MissMask); err != nil {
			return err
		}
		if err := vt.WritePNG(filepath.Join(out, "resident_set.png"), residency); err != nil {
			return err
		}
	}
	if stage == "04_complete" {
		if err := vt.WritePNG(filepath.Join(out, "resident_set_complete.png"), residency); err != nil {
			return err
		}
		if err := vt.WritePNG(filepath.Join(out, "physical_cache.png"), vt.RenderPhysicalSlots(table)); err != nil {
			return err
		}
	}

	verifyErr := frame.Verify(texture)
	check := verificationCheck{Stage: stage, OK: verifyErr == nil}
	if verifyErr != nil {
		check.Error = verifyErr.Error()
	}
	*checks = append(*checks, check)
	if verifyErr != nil {
		return verifyErr
	}

	snapshot := table.Snapshot()
	_, _, feedbackMiss := vt.FeedbackCoverage(fb, snapshot)
	*points = append(*points, vt.MetricPoint{
		Stage:               stage,
		Frame:               len(*points) + 1,
		ExactPixels:         frame.Metrics.ExactPixels,
		FallbackPixels:      frame.Metrics.FallbackPixels,
		MissingPixels:       frame.Metrics.MissingPixels,
		PixelMissRate:       frame.Metrics.MissRate,
		FeedbackMissRate:    feedbackMiss,
		ResidentPages:       len(snapshot),
		CacheCapacity:       cfg.cache,
		InflightRequests:    engine.Scheduler.InflightCount(),
		CumulativeEvictions: table.EvictionCount(),
	})
	return nil
}

func writeCanonicalAndReports(cfg config, engine *vt.Engine, points []vt.MetricPoint, checks []verificationCheck) error {
	if err := vt.WriteMetricsSVG(filepath.Join(cfg.out, "metrics.svg"), points); err != nil {
		return err
	}
	allOK := true
	for _, check := range checks {
		allOK = allOK && check.OK
	}
	report := map[string]any{
		"config": map[string]any{
			"width": cfg.width, "height": cfg.height, "cache_pages": cfg.cache,
			"feedback_stride": cfg.stride, "workers": cfg.workers,
			"load_delay_ms": cfg.delay.Milliseconds(),
		},
		"all_verified": allOK,
		"checks":       checks,
		"metrics":      points,
	}
	if err := vt.WriteJSON(filepath.Join(cfg.out, "report.json"), report); err != nil {
		return err
	}
	if err := vt.WriteJSON(filepath.Join(cfg.out, "metrics.json"), points); err != nil {
		return err
	}
	return writeHTMLReport(filepath.Join(cfg.out, "index.html"), points, checks)
}

func writeHTMLReport(path string, points []vt.MetricPoint, checks []verificationCheck) error {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"zh-CN\"><meta charset=\"utf-8\">")
	b.WriteString("<title>Go 虚拟纹理反馈与异步替换</title>")
	b.WriteString("<style>body{background:#0b1220;color:#e2e8f0;font:15px system-ui;margin:32px;max-width:1180px}" +
		"img{max-width:100%;border:1px solid #334155;border-radius:10px;background:#0f172a}" +
		".grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px}" +
		"table{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}" +
		"td,th{border-bottom:1px solid #334155;padding:7px 9px;text-align:right}td:first-child,th:first-child{text-align:left}" +
		".ok{color:#4ade80}.bad{color:#f87171}code{color:#7dd3fc}</style>")
	b.WriteString("<h1>软件虚拟纹理：UV 反馈、页表与异步替换</h1>")
	b.WriteString("<p>高倍率相机只请求可见 mip0 页；移动后高分辨率页未到时，页表沿 mip 链使用最近驻留父页，并将采样钳制在该页自身纹素内。</p>")
	b.WriteString("<div class=grid>")
	addFigure(&b, "camera_view.png", "移动后：父级 mip 回退相机视图")
	addFigure(&b, "miss_mask.png", "缺页掩码：白色=请求的精确页未驻留")
	addFigure(&b, "resident_set.png", "驻留集合：各 mip 页表/物理槽位")
	addFigure(&b, "uv_feedback.png", "UV 反馈缓冲：颜色=请求 PageID")
	addFigure(&b, "03_camera_partial.png", "两页到达：部分精确、其余父级")
	addFigure(&b, "03_miss_mask_03_partial.png", "部分到达时的缺页掩码")
	addFigure(&b, "04_camera_complete.png", "全部到达：精确 mip0 视图")
	addFigure(&b, "physical_cache.png", "物理缓存槽位（槽间留边）")
	b.WriteString("</div>")
	b.WriteString("<h2>缺页率曲线</h2><img src=\"metrics.svg\" alt=\"metrics\">")
	b.WriteString("<h2>指标</h2><table><tr><th>阶段</th><th>像素缺页率</th><th>反馈缺页率</th><th>驻留页</th><th>累计淘汰</th></tr>")
	for _, p := range points {
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%.3f</td><td>%.3f</td><td>%d/%d</td><td>%d</td></tr>",
			p.Stage, p.PixelMissRate, p.FeedbackMissRate, p.ResidentPages, p.CacheCapacity, p.CumulativeEvictions))
	}
	b.WriteString("</table><h2>逐帧内容校验</h2><ul>")
	for _, c := range checks {
		status := "<span class=ok>PASS</span>"
		if !c.OK {
			status = "<span class=bad>FAIL</span>"
		}
		b.WriteString(fmt.Sprintf("<li>%s: %s %s</li>", c.Stage, status, c.Error))
	}
	b.WriteString("</ul>")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func addFigure(b *strings.Builder, src, caption string) {
	b.WriteString("<figure><img src=\"" + src + "\" alt=\"" + caption + "\">")
	b.WriteString("<figcaption>" + caption + "</figcaption></figure>")
}
