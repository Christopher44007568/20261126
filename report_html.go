package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

func writeHTMLReport(path string, results []ScenarioResult, rowsByID map[string][]BlockRow) error {
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>BD4X 4x4 base + delta</title>
<style>
:root{color-scheme:dark;--bg:#11141b;--panel:#1b202a;--line:#343b49;--ink:#edf2f7;--muted:#9da9ba;--accent:#58d6c9;--magenta:#ff41c7}
body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 ui-sans-serif,system-ui,sans-serif}
main{max-width:1440px;margin:0 auto;padding:28px}
h1,h2,h3{margin:0 0 12px}h1{font-size:28px}h2{font-size:20px;margin-top:34px}h3{font-size:16px}
p{margin:8px 0;color:var(--muted)}
.panel{background:var(--panel);border:1px solid var(--line);border-radius:10px;padding:18px}
.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}
.grid figure{margin:0}.grid img,.wide img{width:100%;image-rendering:pixelated;border-radius:6px;border:1px solid var(--line)}
.grid figcaption{color:var(--muted);font-size:13px;margin-top:6px}
.wide img{max-height:none}
table{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}
th,td{border-bottom:1px solid var(--line);padding:7px 9px;text-align:right;white-space:nowrap}
th:first-child,td:first-child{text-align:left}
th{color:var(--muted);font-weight:600}
.warn{color:var(--magenta)}.ok{color:var(--accent)}
code,pre{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
pre{overflow:auto;background:#0b0e14;border:1px solid var(--line);padding:12px;border-radius:8px;color:#c9d5e3}
details{margin-top:10px}summary{cursor:pointer;color:var(--accent);padding:6px 0}
.small{font-size:13px}.kv{display:flex;gap:24px;flex-wrap:wrap;margin:10px 0}
.kv b{color:var(--ink)}
</style></head><body><main>
`)
	b.WriteString("<h1>BD4X · 4×4 基色 + 增量块压缩</h1>\n")
	b.WriteString(`<p>基色为 RGB565；每个纹素的 RGB 使用每通道 2 位增量。透明块分为 opaque、1 位 alpha mask、4 位 alpha、empty 四种记录。误差图为最大通道差放大 ×8，洋红色表示可见/透明分类翻转。</p>`)
	b.WriteString("\n<h2>三类纹理总览</h2>\n")
	b.WriteString(`<p>行：平滑渐变 / 锐边 / 透明洞。列：原图 / 解码图 / 误差 ×8。透明区域使用检查板背景。</p>`)
	b.WriteString(`<div class="wide"><img src="overview.png" alt="overview"></div>`)
	b.WriteString("\n<h2>指标</h2>\n<div class=\"panel\"><table><thead><tr>")
	headers := []string{"场景", "可见 RGB MAE", "PSNR dB", "Alpha MAE", "透明分类错误", "空洞填充", "空洞侵蚀", "边界 Alpha MAE", "最差块 RGB MAE", "压缩比"}
	for _, h := range headers {
		fmt.Fprintf(&b, "<th>%s</th>", html.EscapeString(h))
	}
	b.WriteString("</tr></thead><tbody>\n")
	for _, r := range results {
		m := r.Metrics
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%.3f</td><td>%.2f</td><td>%.3f</td>", html.EscapeString(r.ID), m.VisibleRGBMAE, m.RGBPSNR, m.AlphaMAE)
		class := "ok"
		if m.AlphaClassErrors > 0 {
			class = "warn"
		}
		fmt.Fprintf(&b, `<td class="%s">%d</td>`, class, m.AlphaClassErrors)
		fmt.Fprintf(&b, "<td>%d</td><td>%d</td><td>%.3f</td><td>%.3f @ (%d,%d)</td><td>%.2fx</td></tr>\n",
			m.HoleFillErrors, m.HoleErosionErrors, m.BoundaryAlphaMAE,
			m.WorstBlockRGBMAE, m.WorstBlockX, m.WorstBlockY, m.CompressionRatio)
	}
	b.WriteString("</tbody></table></div>\n")
	b.WriteString(`<p class="small">透明审计是独立计数，不参与 RGB MAE。因此平均值再低，只要空洞边缘出现分类翻转，仍会在此表中直接暴露。</p>`)

	for _, r := range results {
		fmt.Fprintf(&b, "\n<h2>%s <span class=\"small\">(%s)</span></h2>\n", html.EscapeString(r.Title), html.EscapeString(r.ID))
		fmt.Fprintf(&b, `<div class="wide"><img src="%s_compare.png" alt="%s"></div>`+"\n",
			html.EscapeString(r.ID), html.EscapeString(r.ID))
		fmt.Fprintf(&b, `<div class="kv"><span>尺寸 <b>%dx%d</b></span><span>块数 <b>%d</b></span><span>流字节 <b>%d</b></span><span>块模式 <b>opaque %d / mask1 %d / alpha4 %d / empty %d</b></span><span><a href="%s">下载 .bd4</a></span></div>`+"\n",
			r.Width, r.Height, r.BlockCount, r.StreamBytes,
			r.OpaqueBlocks, r.MaskBlocks, r.Alpha4Blocks, r.EmptyBlocks,
			html.EscapeString(filepath.ToSlash(r.StreamPath)))
		b.WriteString("<details><summary>展开全部 4×4 块的压缩字节</summary><table><thead><tr>")
		for _, h := range []string{"块坐标", "字节", "alpha 模式", "基色", "跨度 R/G/B", "原始十六进制"} {
			fmt.Fprintf(&b, "<th>%s</th>", h)
		}
		b.WriteString("</tr></thead><tbody>\n")
		for _, row := range rowsByID[r.ID] {
			info := row.Info
			base := "—"
			spans := "—"
			if info.AlphaMode != alphaEmpty {
				base = fmt.Sprintf("#%02x%02x%02x", info.Base.R, info.Base.G, info.Base.B)
				spans = fmt.Sprintf("%d/%d/%d", info.SpanR, info.SpanG, info.SpanB)
			}
			fmt.Fprintf(&b, "<tr><td>(%d,%d)</td><td>%d</td><td>%s</td><td><code>%s</code></td><td>%s</td><td><code>%s</code></td></tr>\n",
				row.X, row.Y, len(row.Data), alphaModeName(info.AlphaMode), base, spans, hexBytes(row.Data))
		}
		b.WriteString("</tbody></table></details>\n")
	}
	b.WriteString("\n<h2>压缩流</h2>\n")
	b.WriteString("<p>每行 24 字节。完整十六进制清单：<a href=\"compressed_hex.txt\">compressed_hex.txt</a>；指标 JSON：<a href=\"metrics.json\">metrics.json</a>；文本摘要：<a href=\"report.txt\">report.txt</a>。</p>\n")
	b.WriteString("</main></body></html>\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
