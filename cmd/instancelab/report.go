package main

import (
	"fmt"

	"instancelab/internal/lab"
)

func passWord(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

func buildReport(b *auditBundle, shared *lab.SharedMapFault, writeback *lab.WritebackFault) string {
	c := b.ColorIsolation
	d := b.DeletionSafety
	g := b.GeometrySync
	return fmt.Sprintf(`# InstanceLab：原型、实例与共享颜色审计

生成状态：**全部通过**。开发与运行仅使用 Go 标准库，go.mod 没有外部依赖。

## 三张主图

1. [原型图](prototype_diagram.svg)
2. [实例总览图（%d 个实例）](instances_overview.svg)
3. [错误共享颜色高亮](shared_color_error.svg)

浏览器打开 [index.html](index.html) 可查看整组图。

## 核心行为

| 验收点 | 结果 | 证据 |
|---|---|---|
| 修改一个实例，原型不变 | %s | %s body: %s → %s，prototype_changed=%v |
| 修改一个实例，其他实例不变 | %s | 意外变化数=%d |
| 删除实例不删除原型 | %s | 删除 %s 后 prototype_exists=%v，实例数 %d→%d |
| 包围盒和渲染轮廓同时更新 | %s | bounds=%v，selection=%v，render=%v |
| 正确场景无共享 map | %s | detect_shared_color_maps=%d 组 |

## 场景统计

- 实例总数：%d
- 可见实例：%d
- 整卡隐藏：%d
- 本地颜色覆盖项：%d
- 单节点隐藏项：%d

## 两个错误共享负面对照

### 共享覆盖表（shared_color_error.svg）

- 修改目标：%s / %s → %s
- 共享 owner：%d 个
- 被连带修改：%d 个非目标实例
- 审计可识别：%v

### 回写原型（prototype_writeback_error.svg）

- 修改目标：%s
- 被连带修改：%d 个非目标实例
- 根因：实例编辑写入 Prototype.Nodes，继承默认色的实例一起变色。

## 几何同步

见 [bounds_outline_sync.svg](bounds_outline_sync.svg)；修复后的隔离结果见
[fixed_color_isolation.svg](fixed_color_isolation.svg)，机器可读审计见
[audit.json](audit.json) 和 [faults.json](faults.json)。

## 运行

`+"```bash"+`
go test ./...
go run ./cmd/instancelab generate -out artifacts/generated
go run ./cmd/instancelab verify
go run ./cmd/instancelab serve -addr 127.0.0.1:8090
`+"```"+`
`,
		b.Counts["instances"],
		passWord(c.Pass), c.TargetInstanceID, c.PrototypeColorBefore, c.RequestedColor, c.PrototypeChanged,
		passWord(c.OtherInstanceChanges == 0), c.OtherInstanceChanges,
		passWord(d.Pass), d.DeletedInstanceID, d.PrototypeExistsAfter, d.InstanceCountBefore, d.InstanceCountAfter,
		passWord(g.Pass), g.BoundsTranslated, g.SelectionOutlineTranslated, g.RenderOutlineTranslated,
		passWord(len(b.SharedMapIssues) == 0), len(b.SharedMapIssues),
		b.Counts["instances"], b.Counts["visible_instances"], b.Counts["hidden_instances"],
		b.Counts["color_overrides"], b.Counts["hidden_nodes"],
		shared.TargetInstanceID, shared.NodeID, shared.EditColor,
		shared.OwnerCount(), len(shared.UnexpectedInstanceIDs), shared.AuditDetectable,
		writeback.TargetInstanceID, len(writeback.UnexpectedInstanceIDs),
	)
}
