package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aelab/internal/pipeline"
)

func WriteReport(path string, result *pipeline.Result) error {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>AE dark-to-window temporal responses</title><style>`)
	b.WriteString(reportCSS())
	b.WriteString(`</style></head><body><main>`)
	b.WriteString(`<h1>相机从暗处转向窗口：自动曝光时间响应</h1>`)
	fmt.Fprintf(&b, `<p class="lead">程序化 HDR 场景，%.0f 秒、%d FPS、共 %d 个连续帧/响应；每帧均导出 480×270 JPEG 缩略图。黄带表示摇镜过程。</p>`, result.Duration, result.FPS, result.FPS*int(result.Duration))

	b.WriteString(`<section class="cards">`)
	for _, p := range result.Profiles {
		recommended := p.Profile.ID == "fast_smooth"
		cardClass := "card"
		badge := ""
		if recommended {
			cardClass += " recommended"
			badge = `<span class="badge">推荐</span>`
		}
		fmt.Fprintf(&b, `<article class="%s" style="border-top-color:%s">%s<h2>%s</h2><p>%s</p>`, cardClass, p.Profile.Color, badge, p.Profile.Label, p.Profile.Description)
		fmt.Fprintf(&b, `<dl><div><dt>收敛时刻</dt><dd>%.2f s</dd></div>`, p.Metrics.SettleTimeS)
		fmt.Fprintf(&b, `<div><dt>最大过冲</dt><dd>%.3f EV</dd></div>`, p.Metrics.MaxOvershootEV)
		fmt.Fprintf(&b, `<div><dt>EV 穿越次数</dt><dd>%d</dd></div>`, p.Metrics.ZeroCrossings)
		fmt.Fprintf(&b, `<div><dt>最大亮度跳变</dt><dd>%.4f</dd></div>`, p.Metrics.MaxLumaJump)
		fmt.Fprintf(&b, `<div><dt>暗部均值/标准差</dt><dd>%.3f / %.3f</dd></div>`, p.Metrics.ShadowMean, p.Metrics.ShadowStd)
		fmt.Fprintf(&b, `<div><dt>窗部边缘能量</dt><dd>%.4f</dd></div></dl>`, p.Metrics.WindowEdge)
		fmt.Fprintf(&b, `<p class="status">无整屏突变：%s；平滑收敛：%s</p>`, passText(p.Metrics.PassesNoFlash), passText(p.Metrics.PassesSmoothConverge))
		b.WriteString(`</article>`)
	}
	b.WriteString(`</section>`)

	b.WriteString(`<section><h2>曝光值曲线</h2><img class="chart" src="exposure_curves.png" alt="Exposure EV curves"><p>矢量版本：<a href="exposure_curves.svg"><code>exposure_curves.svg</code></a></p></section>`)
	b.WriteString(`<section><h2>过冲 / 收敛曲线</h2><img class="chart" src="overshoot_curves.png" alt="Overshoot curves"><p>矢量版本：<a href="overshoot_curves.svg"><code>overshoot_curves.svg</code></a></p></section>`)
	b.WriteString(`<section><h2>逐响应连续帧总览</h2>`)
	for _, p := range result.Profiles {
		fmt.Fprintf(&b, `<figure><figcaption><span class="swatch" style="background:%s"></span>%s — 每 10 帧取样，顺序从左到右、从上到下；完整序列见 <code>frames/%s/</code></figcaption>`, p.Profile.Color, p.Profile.Label, p.Profile.ID)
		fmt.Fprintf(&b, `<img class="sheet" src="contact_sheet_%s.jpg" alt="Contact sheet for %s"></figure>`, p.Profile.ID, p.Profile.ID)
		fmt.Fprintf(&b, `<p><a href="all_frames_sheet_%s.jpg">打开 %s 的全部 300 帧拼图</a></p>`, p.Profile.ID, p.Profile.Label)
		fmt.Fprintf(&b, `<div class="keytimes"><img src="frames/%s/frame_0001.jpg" alt="first frame"><img src="frames/%s/frame_0061.jpg" alt="frame 61"><img src="frames/%s/frame_0091.jpg" alt="frame 91"><img src="frames/%s/frame_0151.jpg" alt="frame 151"><img src="frames/%s/frame_0300.jpg" alt="final frame"></div>`, p.Profile.ID, p.Profile.ID, p.Profile.ID, p.Profile.ID, p.Profile.ID)
	}
	b.WriteString(`</section>`)

	b.WriteString(`<section class="files"><h2>数据文件</h2><ul>`)
	b.WriteString(`<li><a href="exposure_values.csv"><code>exposure_values.csv</code></a>：3×300 帧的 EV、增益、测光和 ROI 统计</li>`)
	b.WriteString(`<li><a href="overshoot_curves.csv"><code>overshoot_curves.csv</code></a>：三种响应的实际 EV、过冲量和收敛百分比</li>`)
	b.WriteString(`<li><a href="ae_metrics.csv"><code>ae_metrics.csv</code></a>：稳定时间、过冲、穿越次数、闪烁/细节指标</li>`)
	b.WriteString(`<li><a href="exposure_curves.png"><code>exposure_curves.png</code></a> / <a href="exposure_curves.svg"><code>.svg</code></a>；<a href="overshoot_curves.png"><code>overshoot_curves.png</code></a> / <a href="overshoot_curves.svg"><code>.svg</code></a></li>`)
	b.WriteString(`</ul></section></main></body></html>`)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func passText(v bool) string {
	if v {
		return `<strong class="pass">通过</strong>`
	}
	return `<strong class="fail">对照/未通过</strong>`
}

func reportCSS() string {
	return `
:root{color-scheme:light}*{box-sizing:border-box}body{margin:0;background:#f3f0e8;color:#202124;font:15px/1.55 -apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans CJK SC","Microsoft YaHei",sans-serif}
main{max-width:1280px;margin:0 auto;padding:34px 28px 70px}h1{font-size:30px;margin:0 0 8px}.lead{color:#5f6368;margin:0 0 26px}h2{font-size:21px;margin:28px 0 14px}
.cards{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.card{background:#fff;border:1px solid #ded9cd;border-top:5px solid #999;border-radius:14px;padding:18px;box-shadow:0 8px 24px rgba(60,50,30,.07);position:relative}.card.recommended{outline:3px solid rgba(31,157,85,.18)}
.badge{position:absolute;right:16px;top:16px;background:#e6f6ec;color:#147a40;border-radius:999px;padding:3px 10px;font-size:12px;font-weight:700}.card h2{margin:0 70px 8px 0;font-size:18px}.card p{color:#5f6368}.card dl{display:grid;grid-template-columns:1fr 1fr;gap:8px 14px;margin:14px 0}.card dt{font-size:12px;color:#777}.card dd{margin:0;font-weight:700}.status{margin-bottom:0}.pass{color:#147a40}.fail{color:#b42318}
.chart,.sheet,figure img{width:100%;height:auto;background:#fff;border:1px solid #d8d4cc;border-radius:12px}.chart{padding:8px}.sheet{display:block}figure{margin:18px 0}figcaption{margin:0 0 8px;font-weight:650}.swatch{display:inline-block;width:12px;height:12px;border-radius:3px;margin-right:7px}.keytimes{display:grid;grid-template-columns:repeat(5,1fr);gap:8px;margin-top:8px}.keytimes img{width:100%;border-radius:8px;border:1px solid #d8d4cc}
.files{background:#fff;border:1px solid #ded9cd;border-radius:14px;padding:8px 20px}.files a{color:#0b66c3;text-decoration:none}.files a:hover{text-decoration:underline}code{background:#f1eee6;padding:2px 5px;border-radius:5px}
@media(max-width:900px){.cards{grid-template-columns:1fr}.keytimes{grid-template-columns:repeat(2,1fr)}}`
}
