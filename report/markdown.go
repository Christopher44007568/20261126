package report

import (
	"fmt"
	"strings"

	"nsmigrate/audit"
	"nsmigrate/scene"
)

// BuildMarkdown renders the human-readable Chinese audit report.
func BuildMarkdown(m scene.Mapping, before, correct, naive *audit.Report) string {
	var b strings.Builder

	b.WriteString("# 命名空间迁移校验报告（reference / 材质绑定 / 相机目标）\n\n")
	b.WriteString("技术栈：仅 Go 标准库，零外部依赖。所有图片由程序自行光栅化并写出二进制 P6 PPM。\n\n")

	b.WriteString("## 1. 迁移映射\n\n")
	b.WriteString("| 旧命名空间（旧路径前缀） | 新路径（重根后） |\n|---|---|\n")
	fmt.Fprintf(&b, "| `%s` | `%s` |\n\n", m.OldRoot, m.NewRoot)
	b.WriteString("正确迁移按**路径组件边界**重写整棵子树：根节点与其全部父子节点路径、" +
		"`references`、`rel material:binding`、`camera.target` 一起改写到新根。\n\n")

	b.WriteString("## 2. 三种状态的结论\n\n")
	b.WriteString("| 状态 | 不变量 | ok | broken | default_retained | 父子路径不一致 |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	stateRow(&b, "迁移前 before", before)
	stateRow(&b, "正确迁移 after（路径/父子感知）", correct)
	stateRow(&b, "只替换字符串 string-only（反例）", naive)
	b.WriteString("\n")
	b.WriteString("- 主交付的两张 PPM：`artifacts/migration_before.ppm` 与 `artifacts/migration_after.ppm`。\n")
	b.WriteString("- 反例图：`artifacts/naive_string_replace_after.ppm`，专门复现“只替换字符串、未更新父子路径”。\n")
	b.WriteString("- 每个红框的像素坐标、类别与旧路径见 `artifacts/red_boxes.json`。\n\n")

	b.WriteString("## 3. 正确迁移后仍然成立的链接\n\n")
	b.WriteString("| ID | 类型 | 持有对象 | 迁移后目标 |\n|---|---|---|---|\n")
	for _, f := range correct.Findings {
		if f.Status != audit.StatusOK {
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | `%s` | `%s` |\n", f.LinkID, kindCN(f.Kind), f.OwnerPath, f.Target)
	}
	b.WriteString("\n参考（R1）、材质绑定（M1/M2）、相机目标（C1）在新路径下都能解析到真实节点。\n\n")

	b.WriteString("## 4. 红框：断链对象与保留的默认值\n\n")
	b.WriteString("红框分两类（`red_boxes.json` 中 `category` 分别为 `broken_link` 与 `retained_default`）：\n\n")
	b.WriteString("- **断链对象**：引用/相机目标解析不到节点，或材质绑定失效。\n")
	b.WriteString("- **保留的默认值**：材质绑定失效后，渲染器没有崩溃，而是保留内建默认材质 " +
		"`" + scene.DefaultMaterialName + "`（RGB `#B9BCC6`），并用红框标出该默认值。\n\n")
	b.WriteString("正确迁移后的图中仍刻意保留两个边界用例，避免把问题“静默洗掉”：\n\n")
	b.WriteString("| ID | 类别 | 持有对象 | 具体旧路径 | 迁移后目标 | 结论 |\n|---|---|---|---|---|---|\n")
	for _, f := range correct.Findings {
		if f.Status == audit.StatusOK {
			continue
		}
		label := "断链"
		if f.Status == audit.StatusDefault {
			label = "绑定失效→保留默认材质"
		}
		fmt.Fprintf(&b, "| %s | %s | `%s` | `%s` | `%s` | %s |\n",
			f.LinkID, kindCN(f.Kind), f.OwnerPath, f.OldPath, f.Target, label)
	}
	b.WriteString("\n默认值明细：\n\n")
	for _, d := range correct.Defaults {
		fmt.Fprintf(&b, "- `%s` 持有对象 `%s` 的 `%s` 未解析到 `%s`，保留默认值 `%s` (#%02X%02X%02X)。\n",
			d.LinkID, d.OwnerPath, d.Property, d.Because, d.ValueName,
			d.ValueRGB.R, d.ValueRGB.G, d.ValueRGB.B)
	}
	b.WriteString("\n")

	b.WriteString("## 5. 只替换字符串、未更新父子路径：给出具体旧路径\n\n")
	fmt.Fprintf(&b, "下面的反例中，脚本只对链接字符串做了字面字符串替换 "+
		"`strings.ReplaceAll(s, %q, %q)`，并改了根节点名字符串，"+
		"但没有沿父子链重写子孙节点的绝对路径。结果链接字符串指向新前缀，",
		m.OldRoot, m.NewRoot)
	b.WriteString(
		"节点却仍注册在旧的父子路径上，于是全部悬空。诊断不只报“未解析”，" +
			"而是给出每条链接的**具体旧路径**以及残留的旧父子路径：\n\n")
	b.WriteString("| ID | 类型 | 具体旧路径 old_path | 字符串替换后的目标 | 残留旧父子路径 stale_parent_child_path | 状态 |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, f := range naive.Findings {
		stale := f.StalePath
		if stale == "" {
			stale = "—（该目标在旧场景中本就不存在）"
		}
		fmt.Fprintf(&b, "| %s | %s | `%s` | `%s` | `%s` | %s |\n",
			f.LinkID, kindCN(f.Kind), f.OldPath, f.Target, stale, f.Status)
	}
	b.WriteString("\n父子路径一致性检查：\n\n")
	if len(naive.ParentChild) == 0 {
		b.WriteString("- 无不一致。\n")
	}
	for _, pc := range naive.ParentChild {
		fmt.Fprintf(&b, "- 节点 id `%s`（父 `%s`）路径为 `%s`，按父节点链重算应为 `%s`。\n",
			pc.PrimID, pc.ParentID, pc.Path, pc.ExpectedPath)
	}
	b.WriteString("\n修复方式：使用路径组件感知的前缀重写（`paths.RewritePrefix`）" +
		"遍历整棵父子树，而不是裸字符串替换；这同时避免 `" + m.OldRoot +
		"Archive` 这类同前缀旁支被误伤。\n\n")

	b.WriteString("## 6. 复现\n\n")
	b.WriteString("```sh\n")
	b.WriteString("go run .          # 重新生成 artifacts/ 下的 PPM/PNG/JSON/MD\n")
	b.WriteString("go test ./...     # 锁定路径重写、迁移不变量、旧路径引用与红框像素\n")
	b.WriteString("```\n\n")
	b.WriteString("PPM 可直接查看，或用 `pnmtopng migration_after.ppm` / ImageMagick `convert` 转 PNG。\n")
	return b.String()
}

func stateRow(b *strings.Builder, name string, r *audit.Report) {
	inv := "PASS"
	if !r.InvariantOK {
		inv = "**FAIL**"
	}
	fmt.Fprintf(b, "| %s | %s | %d | %d | %d | %d |\n",
		name, inv, r.Counts["ok"], r.Counts["broken"], r.Counts["default_retained"], len(r.ParentChild))
}

func kindCN(k scene.LinkKind) string {
	switch k {
	case scene.KindReference:
		return "reference"
	case scene.KindMaterialBinding:
		return "材质绑定"
	case scene.KindCameraTarget:
		return "相机目标"
	}
	return string(k)
}
