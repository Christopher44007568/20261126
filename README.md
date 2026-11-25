# Wrap-safe Go mipmap lab

纯 Go 标准库实现的可平铺 mipmap 生成与验证工具。它使用周期环绕滤波、核权重/DC 能量归一和高度图法线重建，输出各级 mip 拼图、接缝放大对照图、平均色误差图与结构化报告。不使用 CGO、图形 SDK 或第三方图像依赖。

## 运行

```bash
go test ./...
go run . --out artifacts/mipwrap
go run ./cmd/mipwrap --out artifacts/mipwrap
```

常用参数：

- `-size 512`：程序化诊断纹理尺寸，默认 `256`。
- `-input texture.png`：使用自己的 PNG/JPEG 颜色图。
- `-height height.png`：指定高度图；未指定时从亮度重建高度。
- `-linear-light`：在线性光空间滤波；默认在 gamma 编码 RGB 空间统计，使 PNG 平均色误差直接可见地稳定。
- `-normal-strength 2.4`：Sobel 法线强度。

校验失败时命令以非零状态退出；阈值为工作色彩空间平均色 RMS 误差 `1e-9`。

## 产物

默认写入 `artifacts/mipwrap/`：

- `color_mip_montage.png`：颜色各级 mip 拼图。
- `normal_mip_montage.png`：每级从降采样高度重新重建的单位法线拼图。
- `seam_zoom.png`：接缝四角放大对照，列为朴素零填充、仅界内重归一、环绕归一。
- `normal_seam_zoom.png`：法线接缝放大图。
- `edge_probe_zoom.png`：恒定灰边缘探针，排除纹理内容影响，直接检查是否逐级变黑或先暗后亮。
- `average-color-error.png`：各级平均色 RMS 误差对数曲线。
- `average-color-error.csv`、`metrics.json`、`report.md`：平均色、亮度、接缝带误差和恒定灰探针数据。
- `levels/`：每一级独立的颜色与法线 PNG。

## 实现要点

- `internal/mipmap/image.go`：浮点 RGB 图像、周期坐标、sRGB/线性光转换和 PNG 编码。
- `internal/mipmap/filter.go`：`BoundaryWrap` 周期环绕、4×4 可分离盒式 footprint、朴素越界零填充对照、界内权重重归一和 `NormalizeEnergy`。
- `internal/mipmap/normal.go`：高度 mip 链、周期 Sobel 梯度、单位法线重建和无 gamma 的法线数据编码。
- `internal/mipmap/metrics.go`：平均工作色、输出 8-bit 平均色、线性光亮度、接缝带 MAE 和边缘亮度。
- `internal/mipmap/compose.go`、`montage.go`、`seam.go`、`chart.go`：全部使用 `image`/`image/draw` 生成拼图、放大图和图表。

## 稳定性口径

对于 2 的幂尺寸，环绕 4×4 footprint 在周期边界上恰好覆盖每个纹素相同次数，因此全局平均色保持到浮点误差；`NormalizeEnergy` 再处理奇数尺寸和累计舍入。恒定灰探针中，环绕链从原始尺寸到 `1×1` 的边缘/输入亮度比始终为 `1.0`，不会出现逐级变黑或先暗后亮。

朴素对照把越界抽头当作黑色且保留完整分母，因此接缝和 `1×1` 会逐级明显变黑；“仅界内重归一”能保住恒定色块，但会改变边界核形状，报告中保留其误差作为对照。
