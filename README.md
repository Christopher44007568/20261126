# refproof · 引用链资产变更验证

纯 Go 标准库（外部依赖为零）实现的 Showroom 引用链 proof-board。构造
**材质 / 几何 / 摄影机**三类 reference，沿
`instances → models → materials|geometry|cameras` 引用链解析最终绝对路径与
逐跳来源，并输出「引用前 / 引用后 / 无资产占位」三张 1080×720 PNG 与一份
`manifest.json`。

## 直接运行

```sh
go run .                 # 以当前目录为仓库根，重建 fixtures 与 artifacts
go run . -root /path/to/uid5-0022
go test ./...            # 引用链 / 错误契约 / 文本截断单测
```

运行结束会：

1. 把内置的干净数据集写入 `fixtures/showroom/`，并同步到工作树
   `artifacts/work/showroom/`（每次运行先重置，保证可重复）。
2. 依次生成三张图与 `artifacts/manifest.json`（含每张图的字节数与 sha256）。
3. 在标准输出打印每次变更的 `changedPath / finalPath / source`。
4. 跑内置自检（哈希一致、错误被记录、错误红像素确实出现），失败时退出码为 1。
5. 把工作树恢复成引用前的干净状态。

## 结论：是否只更新引用位置？

**是。** 修改沿引用链落到被引用的源资产，只有真正引用该资产的实例场景会更新；
独立拷贝指向自己的 `archive.json` 材质，完全不变。

- 更新：`instances/live.json -> models/live.json -> materials/live.json`
- 不变：`instances/archive.json -> models/archive.json -> materials/archive.json`

`manifest.json` 的 `impact.onlyReferencingPathsUpdate = true` 记录该结论。

## 三张图与错误契约

- `artifacts/before.png`：引用前，live 与 archive 都解析到各自的 coral 材质，
  两条链路均 `ALL REFERENCES RESOLVED`。
- `artifacts/after.png`：沿链更新 live 材质 `baseColor #F06A5D -> #6C7BFF`。
  该阶段刻意模拟外部保存丢失了必填的 `kind` 鉴别字段，解析器给出
  `reference ".../materials/live.json" has no kind`，视口叠加红色斜纹与
  `ERROR OVERLAY / MISSING MATERIAL` 横幅；物体仍绘制，不是空场景。
- `artifacts/missing.png`：把 `models/live.json` 的
  `references.geometry` 改指向不存在的 `showcase_missing.json`，末跳
  `exists=false`，错误为真实的 `open ...: no such file or directory`。
  视口替换为**可读占位面板**（文件图标、`MISSING GEOMETRY`、`PATH`、
  截断的 `SOURCE` 链与 “SCENE IS NOT EMPTY” 说明）+ 红色描边/斜纹错误覆盖色；
  下方仍列出 MAT/GEO/CAM 的最终路径来源。archive 卡片始终健康。

缺失/失效引用永远不会被静默吞掉：每个 slot 都有 `chain[].exists`、
`finalPath`、`displayPath`、`source` 与非空 `error`，画面上也有对应状态色。

## 每次变更的最终路径来源

`manifest.json -> changes[]` 与标准输出给出：

- `material-field`：`materials/live.json` 的 `baseColor` 前后值，
  `finalPath` 指向解析后的材质绝对路径，`source` 为完整引用链。
- `reference-change`：`models/live.json` 的 `references.geometry`
  从 `showcase.json` 改为 `showcase_missing.json`，`finalPath` 是新引用的
  绝对目标（此文件缺失），`chain` 最后一跳 `exists:false`。

## 代码结构

- `seed.go`：内置 Showroom JSON 数据集（唯一数据真相）。
- `workspace.go`：fixture/工作树物化、重置与 JSON 改写。
- `chain.go`：相对引用解析、逐跳 chain、最终路径/source、错误包装。
- `pipeline.go`：before/after/missing 三态编排、changes 与 impact。
- `model.go`：manifest 数据结构（字段顺序即 JSON 键顺序）。
- `canvas.go` / `shapes.go` / `text.go`：零依赖 RGBA 画布、抗锯齿图元、PNG 编码。
- `font_data.go` / `font_extra.go`：内嵌 5×7 位图字体（A–Z、0–9、标点）。
- `art.go` / `panels.go` / `board.go` / `theme.go`：展示物、缺失占位与 proof-board 排版。
- `verify.go`：产物哈希与“错误覆盖色可见”像素自检。
- `chain_test.go`：`go test ./...` 单测。

## 零外部依赖

`go.mod` 没有任何 require，全部 import 均为标准库；
`GOPROXY=off go build ./...` 可离线构建。
