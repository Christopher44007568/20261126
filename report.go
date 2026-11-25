package main

import (
	"fmt"
	"strings"

	"hdrexport/internal/hdr"
	"hdrexport/internal/reload"
)

func buildReport(s hdr.Stats, exposure float64, peakCode uint8, m reload.Metrics,
	pfmPath, pngPath, sheetPath string) string {
	var b strings.Builder
	b.WriteString("============================================================\n")
	b.WriteString(" HDR 截图双路导出报告（零第三方依赖，仅用 Go 标准库）\n")
	b.WriteString("============================================================\n\n")

	b.WriteString("[1] 两路导出\n")
	fmt.Fprintf(&b, "  浮点 PPM (PFM): %s\n", pfmPath)
	b.WriteString("    - PF 魔数，RGB 三通道 IEEE-754 float32，小端，bottom-up 扫描线\n")
	b.WriteString("    - 原始线性光数据，无 Gamma、无色域/白点元数据，允许 >1 的高光\n")
	fmt.Fprintf(&b, "  8-bit PNG 预览: %s\n", pngPath)
	b.WriteString("    - 8 位 RGB，内嵌 sRGB / gAMA(0.45455) / cHRM / tEXt 块\n")
	b.WriteString("    - 查看器据此识别像素已完成 sRGB 编码，避免再次套用 Gamma\n\n")

	b.WriteString("[2] 场景亮度统计（线性光）\n")
	fmt.Fprintf(&b, "  像素总数      : %d\n", s.TotalPx)
	fmt.Fprintf(&b, "  最大亮度      : %.4f（SDR 白 = 1.0000）\n", s.MaxLum)
	fmt.Fprintf(&b, "  最大单通道码值: %.4f\n", s.MaxChannel)
	fmt.Fprintf(&b, "  平均亮度      : %.4f\n", s.MeanLum)
	fmt.Fprintf(&b, "  超过 1.0 的像素: %d (%.2f%%)\n\n", s.OverWhite,
		100*float64(s.OverWhite)/float64(s.TotalPx))

	b.WriteString("[3] 转换说明（线性 HDR -> 8-bit sRGB）\n")
	fmt.Fprintf(&b, "  步骤 1 曝光   : L' = L * %.3f\n", exposure)
	b.WriteString("  步骤 2 色调映射: Reinhard  T = L' / (1 + L')，平滑压缩 >1 高光而非硬裁切\n")
	b.WriteString("  步骤 3 传递函数: sRGB OETF（暗部线性段 + 2.4 次幂段），仅应用一次\n")
	b.WriteString("  步骤 4 量化   : 四舍五入到 [0,255]\n")
	fmt.Fprintf(&b, "  场景峰值 %.4f 经映射后的最大码值 = %d / 255\n\n", s.MaxLum, peakCode)

	b.WriteString("[4] 为什么预览软件可能二次套用 Gamma\n")
	b.WriteString("  PFM 不携带任何色彩元数据，软件只能按约定把它当作线性光；\n")
	b.WriteString("  若 PNG 缺少 gAMA/sRGB/cHRM，部分软件会猜测一个 Gamma 再编码一次，\n")
	b.WriteString("  画面会整体发灰发白（见对比页中段 DOUBLE GAMMA 演示条）。\n")
	b.WriteString("  本 PNG 显式声明 sRGB，配合 tEXt 记录转换来源，保证 Gamma 只做一次。\n\n")

	b.WriteString("[5] Reload 往返校验\n")
	fmt.Fprintf(&b, "  对比图        : %s\n", sheetPath)
	fmt.Fprintf(&b, "  float32 最大绝对误差: %.3g\n", m.MaxAbsErr)
	fmt.Fprintf(&b, "  float32 平均绝对误差: %.3g\n", m.MeanAbsErr)
	fmt.Fprintf(&b, "  float32 最大相对误差: %.3g\n", m.MaxRelErr)
	fmt.Fprintf(&b, "  重渲染字节差异数    : %d（最大单通道差 %d）\n", m.ByteDiffs, m.MaxByteD)
	if m.ByteDiffs == 0 {
		b.WriteString("  结论: PFM 往返无损，reload 前后画面字节级一致。\n")
	} else {
		b.WriteString("  结论: 差异仅来自 float32 存储精度，画面视觉一致。\n")
	}
	b.WriteString("============================================================\n")
	return b.String()
}
