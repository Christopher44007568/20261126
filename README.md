# 20261126

渲染管线 Pass 依赖图演示：声明式资源依赖、拓扑排序、循环检测，
以及不合法输入下“拒绝运行、不产生半张画面”的保证。仅使用 Go 标准库。

## 构建与运行

```bash
go test ./...

# 正确接线：生成 artifacts/frame.ppm
go run ./cmd/render valid artifacts/frame.ppm

# 两条边接反：检测到完整循环链并拒绝运行
go run ./cmd/render swapped

# 缺失输入资源：静态校验失败并拒绝运行
go run ./cmd/render missing
```

## Pass 与资源

| Pass | 读取 | 写入 |
| --- | --- | --- |
| `geometry` | — | `depth` |
| `shadow` | `depth` | `shadow` |
| `color` | — | `color` |
| `lighting` | `color`, `shadow` | `lit` |
| `postprocess` | `lit` | `final_color` |
| `output` | `final_color` | `ppm_color` |

阴影 Pass 依赖深度，光照/后处理依赖颜色。边的方向是
`生产者 -> 消费者`，建图后用 Kahn 算法做确定性拓扑排序。

## 失败处理

- 重复写同一缓冲、读取无生产者的缓冲：`BuildGraph` 静态拒绝。
- 依赖成环：三色 DFS 返回首尾闭合的完整路径（如
  `lighting -> postprocess -> lighting`），`Plan` 返回 `CycleError`。
- 任何校验失败都在执行第一个 Pass 之前发生，不会生成帧或 PPM。
- PPM 写入采用临时文件 + `rename` 原子替换，失败时不留半张画面。

## 代码结构

- `internal/pipeline/graph.go`：有向图、环检测、拓扑排序、DOT/邻接表渲染。
- `internal/pipeline/pipeline.go`：Pass/Pipeline、资源校验、执行计划。
- `internal/pipeline/frame.go`：帧缓冲、P6 PPM 编码与原子落盘。
- `internal/pipeline/passes.go`：六个具体渲染 Pass。
- `cmd/render/main.go`：`valid` / `swapped` / `missing` 三个场景。
