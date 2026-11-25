package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Report struct {
	Schema           string             `json:"schema"`
	Canvas           string             `json:"canvas"`
	ViewportPixels   int                `json:"viewportPixels"`
	SourceTexture    int                `json:"sourceTexture"`
	MaxLOD           int                `json:"maxLod"`
	ErrorThresholdPx float64            `json:"errorThresholdPx"`
	TransferBudgetKB int                `json:"transferBudgetKBPerSimFrame"`
	Strategies       []StrategyReport   `json:"strategies"`
	Verification     VerificationReport `json:"verification"`
	ColorLegend      []ColorLegendItem  `json:"colorLegend"`
}

type StrategyReport struct {
	Name   string         `json:"name"`
	Title  string         `json:"title"`
	Frames []FrameMetrics `json:"frames"`
}

type ColorLegendItem struct {
	State   string `json:"state"`
	RGB     string `json:"rgb"`
	Meaning string `json:"meaning"`
}

func NewReport() Report {
	return Report{
		Schema:           "texture-stream-comparison/v1",
		Canvas:           fmt.Sprintf("%dx%d P6 PPM", frameW, frameH),
		ViewportPixels:   mainViewport().Size,
		SourceTexture:    sourceSize,
		MaxLOD:           maxLOD,
		ErrorThresholdPx: errorThreshold,
		TransferBudgetKB: frameBudget / 1024,
		ColorLegend: []ColorLegendItem{
			{State: "EXACT", RGB: "#22C55E", Meaning: "target LOD or finer tile is resident"},
			{State: "LOADING", RGB: "#F59E0B", Meaning: "request is in flight/queued; textured ancestor stays visible"},
			{State: "COARSE_PARENT", RGB: "#38BDF8", Meaning: "intermediate mip parent is the current fallback"},
			{State: "ROOT_PARENT", RGB: "#818CF8", Meaning: "committed root texture is the final fallback"},
		},
	}
}

func WriteReports(outDir string, report Report) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	jsonPath := filepath.Join(outDir, "streaming_report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	mdPath := filepath.Join(outDir, "streaming_report.md")
	return os.WriteFile(mdPath, []byte(RenderMarkdownReport(report)), 0o644)
}

func RenderMarkdownReport(r Report) string {
	var b strings.Builder
	b.WriteString("# 流式纹理策略对比\n\n")
	b.WriteString("纯 Go 标准库实现，输出二进制 P6 PPM。所有策略在子块到达前都通过同一 UV 网格采样已提交的父级纹理；叠色只表示加载状态，不替换成空白画布。\n\n")

	b.WriteString("## 状态叠色\n\n")
	b.WriteString("| 状态 | 颜色 | 含义 |\n| --- | --- | --- |\n")
	for _, item := range r.ColorLegend {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", item.State, item.RGB, item.Meaning)
	}
	b.WriteString("\n")

	b.WriteString("## 三帧指标\n\n")
	b.WriteString("| 策略 | 细节帧 | 已传输 | 目标 LOD 覆盖 | 加载中 | 粗父级 | 根父级 | 平均屏幕误差 |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, strategy := range r.Strategies {
		for _, f := range strategy.Frames {
			fmt.Fprintf(&b, "| `%s` | %s | %d KiB | %.1f%% | %d px | %d px | %d px | %.3f px |\n",
				strategy.Name,
				titleCase(f.Detail),
				f.BytesLoaded/1024,
				f.TargetCoverage,
				f.LoadingPixels,
				f.CoarsePixels,
				f.RootPixels,
				f.MeanError,
			)
		}
	}
	b.WriteString("\n> 注：`加载中` 是琥珀斜纹层，可与 `粗父级/根父级` 重叠；父级列表示斜纹下方实际采样的回退纹理。\n")

	b.WriteString("\n## 连续性校验\n\n")
	fmt.Fprintf(&b, "- 连续相机采样：`%d` 帧\n", r.Verification.FramesChecked)
	fmt.Fprintf(&b, "- 纹理网格像素：`%d`\n", r.Verification.MeshPixels)
	fmt.Fprintf(&b, "- 空白像素：`%d`\n", r.Verification.BlankPixels)
	fmt.Fprintf(&b, "- 纯黑/纯白闪烁像素：`%d`\n", r.Verification.BlackWhiteFlashPixels)
	fmt.Fprintf(&b, "- 同相机加载前后几何遮罩一致：`%t`\n", r.Verification.GeometryMaskStable)
	fmt.Fprintf(&b, "- 最大帧均亮度跳变：`%.2f`\n", r.Verification.MaxMeanLuminanceJump)
	conclusion := "PASS"
	if !r.Verification.Passed {
		conclusion = "FAIL"
	}
	fmt.Fprintf(&b, "- 结论：`%s`\n\n", conclusion)

	b.WriteString("## 产物\n\n")
	b.WriteString("- `low.ppm`、`medium.ppm`、`high.ppm`：低/中/高细节三帧三联对比（规范文件名）。\n")
	b.WriteString("- `comparison_low.ppm`、`comparison_medium.ppm`、`comparison_high.ppm`：低/中/高细节三帧三联对比。\n")
	b.WriteString("- `full_<detail>.ppm`、`chunked_<detail>.ppm`、`screen-error_<detail>.ppm`：各策略单帧。\n")
	b.WriteString("- `streaming_report.json`：机器可读指标。\n\n")

	b.WriteString("## 策略解释\n\n")
	b.WriteString("- `full`：等待整张高分辨率 payload；传输量固定且未完成时只有根父级，但不会清空几何。\n")
	b.WriteString("- `chunked`：按固定行主序请求全部 8×8 L3 分块；镜头移动到底部热点时，预算可能花在不可见或低误差块上。\n")
	b.WriteString("- `screen-error`：按投影后纹素尺寸选择 LOD，优先请求可见且误差最大的节点，并取消偏离视野的同层旧请求；因此用更少字节达到高覆盖。\n")
	return b.String()
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
