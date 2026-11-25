# 20261126

Repository initialized for 20261126.

## InstanceLab：原型与实例颜色/可见性隔离

纯 Go（仅标准库，零外部依赖）实现的原型模式审计项目：一个 `Beacon Card`
原型被 128 个实例通过稳定 ID 引用，实例各自保存颜色与可见性覆盖。

### 设计要点

- 实例只持有 `PrototypeID`，不复制也不回写原型节点树（`internal/lab/model.go`）。
- 颜色覆盖 `ColorOverrides` 与节点隐藏 `HiddenNodes` 是每个实例独立的 map，
  修改一个实例不会改原型，也不会连带兄弟实例。
- 删除实例只动实例注册表，原型独立存活（`Document.DeleteInstance`）。
- 每次 `Snapshot` 在同一遍计算里重算包围盒、选中轮廓和渲染轮廓，三者不会漂移
  （`internal/lab/snapshot.go`）。
- 两个负面对照复现真实故障：共享覆盖表、实例编辑回写原型，并可被审计识别
  （`internal/lab/faults.go`）。

### 运行

```bash
go test ./...
go run ./cmd/instancelab generate -out artifacts/generated
go run ./cmd/instancelab verify
go run ./cmd/instancelab serve -addr 127.0.0.1:8090
```

### 产物（`artifacts/generated/`）

- `prototype_diagram.svg`：原型图（5 个节点与默认颜色）
- `instances_overview.svg`：128 实例总览（覆盖/隐藏标记）
- `shared_color_error.svg`：错误共享颜色高亮（共享 map 反例）
- `prototype_writeback_error.svg`：回写原型反例
- `fixed_color_isolation.svg`：修复后仅目标实例变色
- `bounds_outline_sync.svg`：包围盒与渲染轮廓同步
- `audit.json` / `faults.json`：机器可读审计；`report.md` / `index.html`：报告
