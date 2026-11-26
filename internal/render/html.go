package render

import (
	"fmt"
	"strings"
)

func indexHTML(report MetricsReport) string {
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>10/12-bit 渐变管线到 8-bit 显示</title>
<style>
:root{color-scheme:dark}
*{box-sizing:border-box}
body{margin:0;background:#040a14;color:#dceaf5;font:15px/1.6 ui-sans-serif,system-ui,sans-serif}
main{max-width:1500px;margin:auto;padding:30px 22px 70px}
h1{font-size:clamp(26px,4vw,48px);letter-spacing:.02em;margin:0 0 6px}
h2{font-size:22px;margin:38px 0 12px;color:#aeeF5b}
p{color:#8da7bd;max-width:980px}
.badge{display:inline-block;border:1px solid #24435e;background:#091a2c;border-radius:999px;padding:3px 10px;margin:2px 6px 2px 0;color:#67e8f9}
.cards{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin:22px 0}
.card{border:1px solid #223a53;background:linear-gradient(160deg,#0b1b2d,#071220);border-radius:14px;padding:14px}
.card small{display:block;color:#7391a8;letter-spacing:.08em}
.card b{font-size:26px;color:#b6f35c}
.tabs{position:sticky;top:0;z-index:2;background:rgba(4,10,20,.94);padding:14px 0;border-bottom:1px solid #1b3148}
button{border:1px solid #385a78;background:#0b1d30;color:#dceaf5;border-radius:10px;padding:9px 15px;margin:0 8px 8px 0;cursor:pointer}
button.active{border-color:#43e0ee;color:#67e8f9;box-shadow:0 0 0 1px #43e0ee inset}
img{display:block;width:100%;height:auto;border:1px solid #223a53;border-radius:14px;background:#071220}
.pair{display:grid;grid-template-columns:1fr 1fr;gap:14px}
table{border-collapse:collapse;width:100%;font-size:13px;margin-top:14px}
th,td{border-bottom:1px solid #1b3148;padding:7px 8px;text-align:right;white-space:nowrap}
th:first-child,td:first-child,th:nth-child(2),td:nth-child(2){text-align:left}
th{color:#7391a8;font-weight:500}
code{color:#67e8f9}
.note{border-left:3px solid #43e0ee;padding:10px 14px;background:#071624;border-radius:0 10px 10px 0}
@media(max-width:900px){.cards{grid-template-columns:1fr 1fr}.pair{grid-template-columns:1fr}}
</style>
</head>
<body><main>
`)
	b.WriteString(`<h1>10-bit / 12-bit 渐变管线 → 8-bit 显示</h1>
<p>同一份浮点源在三种管线中只按位深分叉：暗部中性坡、连续 HSV 色相坡、以及按 R→G→B 顺序截断的 HDR 高光坡。10/12-bit 保留 8-bit 小数坐标后再用 4×4 Bayer 抖动，因此额外层级以空间密度出现，而不只是 JSON 里的数字。</p>
<div>`)
	b.WriteString(`<span class="badge">Go 标准库</span><span class="badge">4×4 Bayer</span><span class="badge">色相路径锁定</span><span class="badge">16-bit master</span></div>`)
	b.WriteString(`<div class="cards">`)
	for _, p := range Pipelines {
		fmt.Fprintf(&b, `<div class="card"><small>%s 梯子密度</small><b>%d / 16</b></div>`,
			p.Short, report.LadderDensities[p.Short])
	}
	maxHue := 0.0
	for _, hs := range report.HueSwitches {
		if hs.MaxDeg > maxHue {
			maxHue = hs.MaxDeg
		}
	}
	fmt.Fprintf(&b, `<div class="card"><small>切换位深最大色相差</small><b>%.2f°</b></div>`, maxHue)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="tabs">`)
	b.WriteString(`<button data-src="sheet_8bit.png">8-bit 直出</button>`)
	b.WriteString(`<button class="active" data-src="sheet_10bit_to_8bit.png">10-bit → 8-bit</button>`)
	b.WriteString(`<button data-src="sheet_12bit_to_8bit.png">12-bit → 8-bit</button>`)
	b.WriteString(`<span id="tip" style="color:#7391a8;font-size:13px">几何与色相路径完全共用</span></div>`)
	b.WriteString(`<img id="sheet" src="sheet_10bit_to_8bit.png" alt="三种场景的 10-bit 到 8-bit 输出">`)

	b.WriteString(`<h2>三场景总拼图</h2>
<p>每行对应暗部阶梯、彩色坡和高光截断；每列只改变管线位深。洋红斜线为源截断区，R/G/B 虚线为通道截断位置，热条为逐列平均量化误差。</p>
<img src="bit_gradient_puzzle.png" alt="暗部阶梯、彩色坡、高光截断位深拼图">`)

	b.WriteString(`<div class="pair">`)
	b.WriteString(`<section><h2>亚码抖动梯子</h2><img src="subcode_dither_ladder.png" alt="亚码抖动梯子"></section>`)
	b.WriteString(`<section><h2>管线数值</h2>`)
	b.WriteString(`<div class="note">硬 8-bit 每通道最多 256 个码字。10/12-bit 的可见收益来自更准确的舍入和 4×4 Bayer 的空间平均；12-bit 在两个相邻码之间能形成 16 个不同密度，10-bit 为 4–5 个，8-bit 只有 2 个。</div>`)
	b.WriteString(`<table><tr><th>场景</th><th>管线</th><th>硬色数</th><th>QMAE</th><th>块 RMSE</th><th>色相最大</th><th>截断损失</th></tr>`)
	for _, sc := range report.Scenes {
		for _, p := range sc.Pipelines {
			hue := "—"
			if p.HueApplicable {
				hue = fmt.Sprintf("%.2f°", p.HueMax)
			}
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%d-bit</td><td>%d</td><td>%.3f</td><td>%.3f</td><td>%s</td><td>%.1f</td></tr>`,
				sc.Name, p.Bits, p.UniqueHardColors, p.HardQuantMAE, p.DitherBlockRMSE, hue, p.HardClipLoss)
		}
	}
	b.WriteString(`</table></section></div>`)
	b.WriteString(`<h2>16-bit 主数据</h2><div class="pair">
<img src="master_10bit_16bit.png" alt="10-bit 主数据">
<img src="master_12bit_16bit.png" alt="12-bit 主数据">
</div>`)
	b.WriteString(`<p>主数据 PNG 使用 16-bit 通道承载精确的 10/12-bit 整数层级；8-bit sheet 与总拼图均由同一源函数和同一几何重新采样，因此按钮切换不会改变色相路径或截断位置。</p>`)
	b.WriteString(`<p>完整指标见 <code>metrics.json</code>。重新生成：<code>go run . -out artifacts</code></p>`)
	b.WriteString(`</main><script>
const img=document.getElementById('sheet');
const buttons=[...document.querySelectorAll('button[data-src]')];
buttons.forEach(b=>b.addEventListener('click',()=>{
 buttons.forEach(x=>x.classList.remove('active'));b.classList.add('active');img.src=b.dataset.src;
}));
['sheet_8bit.png','sheet_10bit_to_8bit.png','sheet_12bit_to_8bit.png'].forEach(s=>{const i=new Image();i.src=s});
</script></body></html>`)
	return b.String()
}
