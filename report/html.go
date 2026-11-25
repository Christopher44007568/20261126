package report

import (
	"fmt"
	"strings"
)

// BuildHTML renders a compact Chinese-language index for the PNG sheets.
func BuildHTML(r Report) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">`)
	b.WriteString(`<title>动画时间采样与重定时 · Go 零依赖</title><style>`)
	b.WriteString(`body{margin:0;background:#e8edf4;color:#1f2a37;font:15px/1.7 -apple-system,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}`)
	b.WriteString(`header{background:#101826;color:#fff;padding:28px 34px}h1{margin:0 0 6px;font-size:24px}`)
	b.WriteString(`header p{margin:4px 0;color:#b8c4d6}main{max-width:1500px;margin:0 auto;padding:24px 20px 60px}`)
	b.WriteString(`section{background:#fff;border:1px solid #c4cdd9;border-radius:12px;padding:18px 20px;margin:22px 0;box-shadow:0 2px 10px rgba(20,40,70,.06)}`)
	b.WriteString(`h2{margin:0 0 4px;font-size:19px}.meta{color:#687486;font-size:13px;margin-bottom:12px}`)
	b.WriteString(`img{width:100%;height:auto;border:1px solid #d7dee8;border-radius:8px;display:block}`)
	b.WriteString(`.ok{color:#167440;font-weight:700}.bad{color:#a52626;font-weight:700}`)
	b.WriteString(`.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px;margin-top:10px}`)
	b.WriteString(`.card{background:#f4f8fb;border:1px solid #d7dee8;border-radius:10px;padding:12px 14px;font-size:13px}`)
	b.WriteString(`code{background:#eef2f7;padding:1px 6px;border-radius:5px}`)
	b.WriteString(`table{border-collapse:collapse;font-size:13px;margin-top:8px}td,th{border:1px solid #d7dee8;padding:5px 10px}th{background:#f4f8fb;text-align:left}`)
	b.WriteString(`</style></head><body><header>`)
	b.WriteString(`<h1>动画时间采样、帧率失配与重定时</h1>`)
	b.WriteString(`<p>每个节点独立读取自己的时间样本；无样本节点始终使用自身绑定姿态，不借用邻近对象。</p>`)
	b.WriteString(fmt.Sprintf(`<p>主角样本区间 <code>%.2fs – %.2fs</code>；播放时钟 6fps；Go 标准库实现，零外部依赖。</p>`,
		r.HeroSampleRangeSeconds[0], r.HeroSampleRangeSeconds[1]))
	b.WriteString(`</header><main><section><h2>读图说明</h2><div class="grid">`)
	b.WriteString(`<div class="card"><strong>琥珀色边框</strong><br>查询时间早于首个样本：<code>before_sample_range</code>，钳制到首姿态。</div>`)
	b.WriteString(`<div class="card"><strong>红色边框</strong><br>查询时间晚于末个样本：<code>after_sample_range</code>，钳制到末姿态。</div>`)
	b.WriteString(`<div class="card"><strong>绿色/蓝色菱形</strong><br>绿色=该帧完整呈现关键姿态；蓝色=线性括号覆盖该样本。红叉=招牌姿态被整帧跳过。</div>`)
	b.WriteString(`<div class="card"><strong>STATIC</strong><br>BEACON 与 BUDDY 没有时间样本，所有 45 帧保持各自绑定姿态。</div>`)
	b.WriteString(`</div></section>`)

	for _, sheet := range r.Sheets {
		statusClass := "ok"
		statusText := "无招牌姿态遗漏"
		if len(sheet.SkippedSignatureKeys) > 0 {
			statusClass = "bad"
			statusText = "跳过招牌姿态：" + strings.Join(sheet.SkippedSignatureKeys, ", ")
		}
		fmt.Fprintf(&b, `<section><h2>%s</h2>`, sheet.CNTitle)
		fmt.Fprintf(&b, `<div class="meta">文件 <code>%s</code> · 模式 <code>%s</code> · %.0ffps · %d 帧 · source = %.2f + %.2f × playback · <span class="%s">%s</span></div>`,
			sheet.Image, sheet.Spec.ReadMode, sheet.Spec.PlaybackFPS, len(sheet.Frames),
			sheet.Spec.Retime.Offset, sheet.Spec.Retime.Rate, statusClass, statusText)
		fmt.Fprintf(&b, `<img src="%s" alt="%s"></section>`, sheet.Image, sheet.ID)
	}

	b.WriteString(`<section><h2>静态节点不变量</h2><table><tr><th>节点</th><th>状态</th><th>校验帧数</th><th>与绑定姿态一致</th></tr>`)
	for _, n := range r.StaticPolicy.Nodes {
		ok := "true"
		cls := "ok"
		if !n.IdenticalToBindPose {
			ok = "false"
			cls = "bad"
		}
		fmt.Fprintf(&b, `<tr><td><code>%s</code></td><td>%s</td><td>%d</td><td class="%s">%s</td></tr>`,
			n.ID, n.Status, n.FramesChecked, cls, ok)
	}
	b.WriteString(`</table><p class="meta">完整逐帧数值见 <code>timeline_report.json</code>；总拼图见 <code>animation_frame_sheets_all.png</code>。</p></section>`)
	b.WriteString(`</main></body></html>`)
	return b.String()
}
