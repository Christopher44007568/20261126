# 20261126

Repository initialized for 20261126.

## 快门区间运动模糊（Go 标准库，零外部依赖）

`shutter` 包把运动模糊建模为快门开合区间 `[OpenT, CloseT]` 上的时间积分，
而不是上一帧屏幕速度的有限差分。

### 核心设计

- **区间积分**：在快门区间内均匀采样，每次采样同时取对象与相机在该时刻的姿态。
- **单一规范图像平面**：所有采样先变换到快门中点 `MidT` 的相机坐标系。世界静止点
  在每个采样下都落在同一规范坐标，因此相机平移/旋转不会把静止背景抹成拖影。
- **相对运动才产生模糊**：模糊长度等于材质点在快门两端之间的相对位移；匀速情况下
  屏幕模糊长度与快门时长呈线性关系。
- **中段姿态记录**：`FrameResult.MidCamera` 与 `FrameResult.MidObject` 分别保存相机
  和每个对象在区间中点的姿态。
- **运动矢量场**：以快门开/关两端投影之差（像素单位）写入 `VectorField`，背景恒为
  零矢量；`vectors.png` 用色相编码方向、饱和度编码长度。
- **四倍子帧**：`RenderSubframes` 将区间四等分独立渲染，`Montage` 拼成 2×2 参考图。

### 产物

运行 `go run ./cmd/demo` 会在 `artifacts/` 下生成：

- `frame.png`：单帧运动模糊渲染；
- `vectors.png`：运动矢量色轮图；
- `vectors.mvf`：`MVF1` 头 + 行主序 float64 `(vx, vy)` 原始矢量；
- `subframes.png`：四倍子帧 2×2 参考（Q1–Q4 按时间顺序）。

### 命令

```sh
go test ./...      # 线性性、背景不拖影、矢量、子帧、序列化等测试
go vet ./...
go run ./cmd/demo  # 生成 artifacts/ 下的参考图
```
