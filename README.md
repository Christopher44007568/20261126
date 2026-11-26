# 20261126

Repository initialized for 20261126.

## 材质 Over 覆盖引擎（纯 Go，零外部依赖）

对材质写 `over` 覆盖而不是重复定义 mesh：覆盖补丁只能引用原始资产里已
存在的 mesh，且只允许改它的 `material.color` 与 `material.opacity`。几何
体、绘制顺序和未被覆盖的 mesh 完全不动。

### 运行

```sh
go test ./...          # 单元 + 端到端测试
go run .               # 跑内置 demo，产物写入 out/
go run . scene.json override.json outdir   # 单个自定义用例
```

### 目录

- `scene/`：原始资产结构（mesh、geometry、material）与深拷贝。
- `over/`：覆盖补丁的严格校验与原子应用，输出被覆盖属性清单。
- `render/`：零依赖软件光栅器（4x SSAA、直线 alpha、source-over 合成）。
- `demo/`：示例资产、合法覆盖、含各类非法条目的覆盖文件。
- `out/<case>/`：`base.png`、`over.png`、`final.png` 与 `report.json`。

### 三张图的含义

- `base.png`：原始资产渲染结果，只渲染一次。
- `over.png`：仅包含被覆盖 mesh 的透明层，使用覆盖后的颜色/透明度；不
  包含背景与未覆盖 mesh，也不复制几何。
- `final.png`：背景 + 未覆盖 mesh + over 层按 source-over 合成，局部颜色
  与透明度正确替换原材质。

### 覆盖格式与拒绝规则

合法路径只有两种：

```json
{"meshId":"badge","path":"meshes.badge.material.color","value":{"r":0.1,"g":0.8,"b":0.45}}
{"meshId":"badge","path":"meshes.badge.material.opacity","value":0.4}
```

以下修改会被拒绝，且整份补丁原子回滚、原画面逐字节保留
（`over.png`/`final.png` 与 `base.png` 相同）：

- 目标 mesh 在原资产中不存在（超出目标路径）。
- 路径指向 `geometry.*` 或其他非材质属性。
- 属性值类型不符（如透明度传字符串、颜色传数组）。
- 数值越界（颜色通道与透明度必须在 `[0,1]`）。
- 未知 JSON 字段或重复键。
