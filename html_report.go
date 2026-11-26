package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteHTMLReport(path string, report PipelineReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var html strings.Builder
	html.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	html.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	html.WriteString(`<title>射线参考与简化加速结构对比</title><style>`)
	html.WriteString(`:root{color-scheme:dark}body{margin:0;background:#0b1020;color:#e8eeff;font:15px/1.5 system-ui,"Noto Sans CJK SC",sans-serif}main{max-width:1500px;margin:0 auto;padding:28px}.pass{color:#79f29a}.fail{color:#ff5b7f}.grid4{display:grid;grid-template-columns:repeat(4,1fr);gap:12px}.grid2{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}figure{margin:0;background:#121a2e;border:1px solid #263454;border-radius:12px;padding:10px}img{width:100%;height:auto;display:block;border-radius:8px}figcaption{padding-top:8px;color:#b8c6e8}table{border-collapse:collapse;width:100%;margin:12px 0}td,th{border-bottom:1px solid #263454;padding:6px 8px;text-align:left}code{color:#9ee7ff}.banner{padding:14px 18px;border-radius:12px;background:#141d35;border:1px solid #2b3b63;margin:16px 0}.critical{color:#ff4d6d;font-weight:700}.ok{color:#79f29a}.legend span{display:inline-block;margin-right:14px}.swatch{width:12px;height:12px;border-radius:3px;display:inline-block;margin-right:5px}</style></head><main>`)

	statusClass := "pass"
	statusText := "全部逐像素与探针射线一致"
	if !report.Pass {
		statusClass = "fail"
		statusText = "存在局部不一致；不得用平均误差替代修复"
	}
	fmt.Fprintf(&html, `<h1>软件射线参考 × 简化加速结构</h1><div class="banner"><strong class="%s">%s</strong><br>分辨率 <code>%d×%d</code>；每组额外探针随机射线 <code>%d</code> 条。所有路径共用同一 float64 三角形求交与同一相机/灯光。</div>`, statusClass, statusText, report.Width, report.Height, report.ProbeRays)

	html.WriteString(`<h2>三组总览（行：擦边 / 平行 / 大三角形；列：参考 / 简化 / 交点数 / 误差热图）</h2>`)
	html.WriteString(`<figure><img src="side_by_side.png" alt="三组并排总览"><figcaption>每行使用完全相同的相机和灯光；若加速结构漏三角形，误差热图会在对应像素形成局部红/白点，而不是只影响平均值。</figcaption></figure>`)
	html.WriteString(`<div class="legend banner"><span><i class="swatch" style="background:#ff264a"></i>漏命中</span><span><i class="swatch" style="background:#ffb020"></i>假命中</span><span><i class="swatch" style="background:#e840ff"></i>错误三角形</span><span><i class="swatch" style="background:#20e2ff"></i>交点数不一致</span></div>`)

	fmt.Fprintf(&html, `<h2>负面对照：故意漏掉一个三角形</h2><figure><img src="fault_injection/side_by_side.png" alt="故障注入对照"><figcaption>这个面板不是交付的简化 BVH；它证明热图和严格报告会把局部漏命中标成红色区域。检测状态：<strong class="%s">%t</strong>，漏命中像素 <code>%d</code>，平均颜色误差仅 <code>%.6f</code>，但 <code>pass=false</code>。</figcaption></figure>`,
		passClass(report.FaultControlDetected), report.FaultControlDetected, report.FaultControl.MissedPixels, report.FaultControl.MeanColorError)

	for i, suite := range report.PixelSuites {
		ray := report.RaySuites[i]
		fmt.Fprintf(&html, `<h2 id="%s">%s</h2>`, suite.Name, suite.Name)
		html.WriteString(`<div class="grid4">`)
		for _, item := range []struct{ file, caption string }{
			{"reference.png", "软件暴力参考"},
			{"simplified.png", "保守 BVH 简化加速"},
			{"intersection_count.png", "每射线有效交点数"},
			{"error_heatmap.png", "误差热图"},
		} {
			fmt.Fprintf(&html, `<figure><img src="suites/%s/%s" alt="%s"><figcaption>%s</figcaption></figure>`, suite.Name, item.file, item.caption, item.caption)
		}
		html.WriteString(`</div>`)
		fmt.Fprintf(&html, `<div class="banner"><strong class="%s">像素校验：%t</strong>；不一致像素 <code>%d</code>，漏三角形 <code class="critical">%d</code>，假命中 <code>%d</code>，错误三角形 <code>%d</code>，交点数错误 <code>%d</code>。平均颜色误差 <code>%.8g</code>，最大局部误差 <code>%.8g</code>。<br><strong class="%s">探针校验：%t</strong>；探针 <code>%d</code> 条，失败 <code>%d</code> 条。最大交点数 <code>%d</code>，BVH 平均三角形测试 <code>%.2f</code> / <code>%d</code>。</div>`,
			passClass(suite.Pass), suite.Pass, suite.MismatchPixels, suite.MissedPixels, suite.FalseHitPixels, suite.WrongTrianglePixels, suite.CountMismatchPixels, suite.MeanColorError, suite.MaxColorError,
			passClass(ray.Pass), ray.Pass, ray.Rays, len(ray.Mismatches), suite.MaxIntersectionCount, suite.MeanSimplifiedTests, suite.ReferenceTriangles)

		if len(suite.Regions) > 0 {
			html.WriteString(`<table><thead><tr><th>错误类型</th><th>像素数</th><th>局部包围盒</th></tr></thead><tbody>`)
			for _, region := range suite.Regions {
				fmt.Fprintf(&html, `<tr><td><code>%s</code></td><td>%d</td><td>x=%d..%d, y=%d..%d</td></tr>`, region.Kind, region.Count, region.MinX, region.MaxX, region.MinY, region.MaxY)
			}
			html.WriteString(`</tbody></table>`)
		}
	}

	html.WriteString(`<p>机器可读结果见 <a href="report.json"><code>report.json</code></a>；分类遮罩为 <code>error_mask.png</code>。</p></main></body></html>`)
	return os.WriteFile(path, []byte(html.String()), 0o644)
}

func passClass(pass bool) string {
	if pass {
		return "ok"
	}
	return "fail"
}
