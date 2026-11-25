package main

func buildIndex() string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>InstanceLab · 原型与实例审计</title>
<style>
:root { color-scheme: light; font-family: ui-sans-serif, system-ui, "Noto Sans CJK SC", sans-serif; }
body { margin: 0; background: #eef2ff; color: #0f172a; }
header { padding: 48px 6vw 24px; }
h1 { margin: 0 0 12px; font-size: clamp(30px, 5vw, 56px); letter-spacing: -.04em; }
header p { max-width: 920px; color: #475569; line-height: 1.7; }
main { padding: 0 4vw 64px; display: grid; gap: 24px; }
section { background: rgba(255,255,255,.9); border: 1px solid #dbe4f0; border-radius: 24px; padding: 22px; }
h2 { margin: 4px 8px 16px; font-size: 22px; }
img { width: 100%; height: auto; display: block; border-radius: 14px; border: 1px solid #e2e8f0; background: #f8fafc; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; }
.links { display: flex; flex-wrap: wrap; gap: 10px; padding: 0 8px 24px; }
a { border-radius: 999px; background: #0f172a; color: #fff; padding: 10px 16px; text-decoration: none; font-size: 14px; font-weight: 700; }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<header>
<h1>一个原型，128 个安全实例</h1>
<p>实例通过稳定 ID 引用原型，颜色与可见性覆盖保存在各自独立的 map 中。包含原型图、实例总览、错误共享颜色高亮、删除安全以及包围盒/渲染轮廓同步证据。</p>
</header>
<div class="links">
<a href="prototype_diagram.svg">原型图</a>
<a href="instances_overview.svg">实例总览</a>
<a href="shared_color_error.svg">错误共享高亮</a>
<a href="prototype_writeback_error.svg">回写原型故障</a>
<a href="fixed_color_isolation.svg">修复后隔离</a>
<a href="bounds_outline_sync.svg">几何同步</a>
<a href="audit.json">audit.json</a>
<a href="report.md">report.md</a>
</div>
<main>
<section><h2>原型图</h2><img src="prototype_diagram.svg" alt="原型图"></section>
<section><h2>实例总览（128 个实例）</h2><img src="instances_overview.svg" alt="实例总览"></section>
<section><h2>错误共享颜色高亮（共享覆盖表）</h2><img src="shared_color_error.svg" alt="错误共享颜色高亮"></section>
<div class="grid">
<section><h2>回写原型故障</h2><img src="prototype_writeback_error.svg" alt="回写原型"></section>
<section><h2>修复后颜色隔离</h2><img src="fixed_color_isolation.svg" alt="修复后隔离"></section>
</div>
<section><h2>包围盒与渲染轮廓同步</h2><img src="bounds_outline_sync.svg" alt="几何同步"></section>
</main>
</body>
</html>
`
}
