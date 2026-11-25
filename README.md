# 20261126

Repository initialized for 20261126.

## 实例掩码与射线类型过滤渲染器（纯 Go，无第三方依赖）

每个实例携带 `MaskVisible | MaskShadow | MaskReflection` 三类掩码位，
求交时按射线类型（主相机 / 阴影 / 反射）过滤实例，被掩码排除的实例
不会遮挡其他实例，只计入排除热图。

- `go run .`：渲染并输出 `out/{primary,shadow,reflection,heatmap}.ppm`
- `go test ./...`：验证逐射线类型过滤、排除实例不遮挡、以及修改某实例
  单个掩码位只影响对应通道（其余通道逐像素一致）
