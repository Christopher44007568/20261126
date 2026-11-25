# 薄透镜景深与顺序遮挡

纯 Go 标准库实现的薄透镜景深渲染器，不使用图形 SDK、图像库或外部依赖。
景深不是高斯后处理：每条相机光线从多边形光圈上采样，并收敛到焦平面；
散景形状、弥散圆和近远遮挡都由三维求交结果产生。

## 生成 PPM

```sh
go run .
```

默认在 `artifacts/` 写出三张二进制 `P6` PPM：

- `artifacts/highlight_spheres.ppm`：不同深度的高光球，近/远焦外呈现六边形散景。
- `artifacts/thin_foreground_rods.ppm`：细前景杆遮挡后方高光；杆后散景被切开，最近求交保证远光不会画到杆前。
- `artifacts/near_far_ruler.ppm`：从近到远的深度标尺，焦平面刻度最清晰，两侧按弥散圆虚化。

常用参数：

```sh
go run . -samples 256                 # 更低噪点
go run . -fstop 16 -out artifacts/f16 # 小光圈：整组画面回到近清晰图
go run . -scene rods                  # 只渲染 highlight|rods|ruler
go test ./...
```

## 实现要点

- `lens.go`：多边形光圈的内部判定、均匀采样，以及薄透镜公式的有符号/无符号弥散圆。
- `camera.go`：薄透镜光线；光圈采样点与焦平面对应点相连，焦平面上的点严格收敛。
- `geometry.go`：球、闭合细圆柱杆和带 UV 的四边形；每个像素选择最近命中。
- `render.go`：并行 Monte Carlo 渲染、直接光照、一次反射、HDR ACES 色调映射和 PPM 输出。
- `ruler.go`：纯代码生成刻度/数字纹理，再由光线采样，因此刻度也遵循同一景深。

`SignedCircleOfConfusion` 的符号可区分焦点前/后；普通 `CircleOfConfusion`
返回直径，`CoCRadiusPixels` 换算为像素半径。测试覆盖焦平面零 CoC、
近远符号、光圈形状、小光圈接近针孔图，以及近景杆阻断远景高光。

Repository initialized for 20261126.
