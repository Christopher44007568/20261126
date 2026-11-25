# 瞬态纹理别名：寿命区间、屏障日志与污染定位

纯 Go 标准库实现，无第三方依赖、无 CGO、无图形 SDK。程序用 CPU 执行一条
六 Pass 的全屏效果链，比较三种物理资源分配：

1. `no_reuse`：每个瞬态纹理独占一个槽，作为逐像素基准。
2. `valid_reuse`：按真实读写寿命做别名，6 个纹理压到 3 个槽。
3. `error_reuse`：故意把“最后一次写”当成寿命终点，漏掉后续读取，导致
   `sky` 的槽在 Pass 2 被 `sun` 覆盖，Pass 3 读到了 `sun` 的旧颜色。

运行：

```sh
go test ./...
go run .
```

默认在 `artifacts/` 生成三张二进制 P6 PPM：

- `no_reuse.ppm`
- `valid_reuse.ppm`
- `error_reuse.ppm`

以及机器/人工可读记录：

- `intervals.log`：逐 Pass 读写表，以及正确和错误寿命区间。
- `aliases.log`：三种模式的槽位分配、共享资源、违例列表。
- `barriers.log`：RAW / WAR / WAW / ALIAS_* 屏障转换。
- `summary.txt`：一块式结果摘要。
- `result.json`：槽位、屏障、违例、哈希、首污染区和断言的结构化报告。

## 效果链

| Pass | 效果 | 读取 | 写入 |
| --- | --- | --- | --- |
| 1 | `sky-fill` | — | `sky` |
| 2 | `sun-glow` | `sky` | `sun` |
| 3 | `clouds` | `sky`, `sun` | `clouds` |
| 4 | `atmosphere` | `clouds` | `lit` |
| 5 | `grade` | `lit` | `graded` |
| 6 | `vignette` | `graded` | `final` |

真实寿命取首次访问到末次访问（读或写都算）：

```text
sky     [1,3]   sun     [2,3]   clouds  [3,4]
lit     [4,5]   graded  [5,6]   final   [6,6]
```

首适配分配器只允许“声明区间完全不相交”的资源共享同尺寸/同格式槽：

```text
slot 0: sky[1..3] -> lit[4..5] -> final[6..6]
slot 1: sun[2..3] -> graded[5..6]
slot 2: clouds[3..4]
```

Pass 内输入会先快照，再执行输出，因此合法别名不依赖执行时的覆盖顺序。
`valid_reuse.ppm` 与 `no_reuse.ppm` 的 PPM 文件逐字节相同，当然也逐像素相同。

## 错误寿命如何显形

错误模式把寿命改成“最后一次写”：

```text
sky[1,1] sun[2,2] clouds[3,3] lit[4,4] graded[5,5] final[6,6]
```

这些点区间让所有资源被分配到同一个槽。实际执行仍然要到 Pass 2 读
`sky`、Pass 3 再读 `sky`。Pass 2 结束时槽中已是 `sun`，所以 Pass 3 的
`sky` 输入实际是上一个效果的太阳颜色，云层合成随即出现错误。

程序按行主序比较错误图与基准图，第一个差异像素所在的 4 连通区域就是
“首个污染区域”。`error_reuse.ppm` 保留错误颜色，并用黑/黄双边框圈出该
区域，红白十字标出首个差异像素。默认 320×180 输出的首差异为
`(179,25)`，首连通区域边界为 `(65,25)-(236,43)`；精确值同时写入
`result.json`。

## 屏障与自检

屏障日志按实际槽位变化记录：

- `RAW`：同一资源先写后读。
- `WAR`：同一资源先读后写。
- `WAW`：同一资源连续写。
- `ALIAS_WAR`：旧纹理最后一次读后，把槽交给新纹理写。
- `ALIAS_RAW`：新纹理读取时槽里仍是前一个租户，错误模式会明确出现。
- `ALIAS_RAR`：同一 Pass 中两个不同资源占同一槽读取，即寿命冲突。

`go test ./...` 验证：

- 合法计划无区间交叠、无寿命违例，并确实减少槽数。
- 合法输出与无复用基准逐像素一致。
- 合法计划产生别名屏障。
- 错误计划有寿命违例，首个分叉 Pass 是 `clouds`。
- 首污染像素确实不同，连通边界和总差异数被正确记录。
- 三张 PPM 的 P6 头、尺寸、载荷长度正确；合法 PPM 与基准文件逐字节相等。

## 代码结构

- `internal/aliaslab/graph.go`：资源、Pass、访问序列和读写日志。
- `internal/aliaslab/plan.go`：寿命区间分析、首适配别名和违例检查。
- `internal/aliaslab/barrier.go`：槽位转换与屏障日志。
- `internal/aliaslab/run.go`：快照输入、写槽执行和逐 Pass 分叉检测。
- `internal/aliaslab/effects.go`、`scene.go`：六段确定性软件效果链。
- `internal/aliaslab/contamination.go`：首差异连通区和错误图标记。
- `internal/aliaslab/experiment.go`、`report.go`：三种模式编排、产物和 JSON。
- `internal/aliaslab/image.go`：RGB 缓冲与原子 P6 PPM 写入。
