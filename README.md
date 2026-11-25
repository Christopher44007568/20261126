# 加速结构代际复用与安全退役

纯 Go、无第三方依赖的确定性模拟：多个光照路径查询跨帧读取同一组加速结构槽位时，通过**资源代际 + 每查询引用 + 退役队列**阻止槽位过早复用。

## 运行

```bash
go run .
go test ./...
go vet ./...
```

默认输出到 `out/`，也可指定尺寸和目录：

```bash
go run . -out out -width 480 -height 270
```

## 输出

| 文件 | 内容 |
| --- | --- |
| `out/frame_001.ppm` | 第 1 帧，P6 RGB |
| `out/frame_002.ppm` | 第 2 帧，P6 RGB |
| `out/frame_003.ppm` | 第 3 帧，P6 RGB |
| `out/reuse_intervals.ppm` | 槽位、查询和退役时间线；黄色区间标出复用等待 |
| `out/dangling_references.ppm` | 查询 → 代际 → 槽位引用图；悬空边为红色虚线 |
| `out/report.json` | 代际、引用、槽位、复用、事件和指标 |
| `out/dangling_references.dot` | Graphviz DOT 版本的引用图 |

当前交付指标：

```text
frames: 3, ghost pixels: 0
safe slot reuses: 4/4, dangling refs: 0, premature releases: 0
builds overlapped with active queries: 2
```

## 调度

共享 arena 固定为 8 个槽位，每帧的加速结构占 4 个槽位；每帧有 A/B 两条异步光照路径查询。

| tick | 事件 |
| --- | --- |
| T0 | G1 建到槽位 0–3；F1-A/B 获取 G1 引用 |
| T1 | G1 退役入队但仍有 2 个引用；G2 建到槽位 4–7 |
| T3 | F1-A 完成；G2 退役；G3 请求复用，但 0 个空闲槽位，请求被阻止 |
| T4 | F1-B 完成，G1 引用归零并释放槽位 0–3；G3 立即安全复用这 4 个槽位 |
| T5 | F2-B 完成，G2 引用归零并释放槽位 4–7 |
| T7 | F3-B 完成；G3 最后一个查询结束 |
| T8 | G3 退役并释放槽位 |

时间线图中的黄色 `T3–T4` 区间就是 G3 的复用等待区间：请求已经发出，但旧代际仍被 F1-B 读取，因此不能写入。

## 安全不变量

- 每个槽位带 `Generation` 标签；查询绑定槽位集合与期望代际，读取时若槽位代际不符就记录悬空引用。
- 每次路径查询独立持有引用；最后一个引用释放后，退役队列才会归还槽位。
- `ReusedAt >= LastFromRefDone` 且复用瞬间 `ActiveRefsAtReuse == 0`。
- 每帧渲染结果与使用当前场景重新渲染的 oracle 逐像素比较；任何旧物体残影都会计入 `ghost_pixels`。
- 发生悬空读、提前释放、残影或不安全复用时，程序仍写出诊断文件，但以非零状态退出。

报告中的释放/最后查询时刻为：

```text
G1 release T4 >= last query T4
G2 release T5 >= last query T5
G3 release T8 >= last query T7
```

## 代码结构

- `arena.go`：代际、槽位、引用、退役队列和悬空读检测。
- `simulation.go`：三帧确定性调度、异步 A/B 路径、残影比较和报告。
- `geometry.go`：最小光线追踪与三帧场景。
- `diagram.go`、`graph.go`：复用时间线和引用图绘制。
- `canvas.go`：纯 Go PPM 画布和位图字体。
- `output.go`：PPM、JSON、DOT 输出及交付前安全校验。
