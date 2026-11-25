# Sensor Noise Lab

纯 Go 标准库实现的传感器噪声与降噪实验，不依赖图形 SDK 或第三方包。

## 运行

```bash
go test ./...
go run .
```

所有结果写入 `output/`，随机数使用固定种子，运行结果可复现。

## 传感器模型

每个 RGB 通道独立模拟，蓝通道增益和读出噪声更高，以体现彩色通道差异：

- 光子散粒噪声：对期望电子数采样泊松分布，`variance = mean`，亮部幅度随信号升高。
- 读出噪声：按 ISO 调整尺度的零均值高斯噪声，低曝光高 ISO 最明显。
- 固定模式噪声：像素、列、行三个分量叠加，列噪声形成可见竖纹。
- 信号链包含通道增益、满阱裁切、零值截断和 sRGB 显示转换。

三档画面：

- 低曝光：ISO 3200，强调暗部、彩噪和固定模式噪声。
- 中曝光：ISO 800，散粒噪声和读出噪声相对均衡。
- 高曝光：ISO 200，噪声最低，亮部更接近满阱。

## 输出

- `component_photon_shot.png`：光子散粒噪声残差。
- `component_read_noise.png`：读出噪声残差。
- `component_fixed_pattern.png`：固定模式噪声绝对幅度。
- `sheet_noise_components.png`：三张分量图总览。
- `exposure_low.png`、`exposure_middle.png`、`exposure_high.png`：三档带噪曝光。
- `denoised_*.png`：对应降噪结果。
- `detail_reference_vs_denoised.png`：参考纹理、带噪图、降噪图三栏放大对照。
- `sheet_exposures.png`：参考和三档曝光总览。

## 降噪与保真

降噪先削弱行/列固定模式，再组合双边滤波和边缘感知的可分离方向滤波。方向滤波沿竖纹方向平滑随机噪声，同时限制跨边缘权重，避免只追求平坦暗部。

终端输出两个指标：

- `texture-retention`：参考显著边缘与降噪结果的 Sobel 方向一致性。
- `flat-noise-reduction`：平坦区域局部 RMS 的下降比例。

测试同时要求高 ISO 下显著降低平坦噪声，并保留至少 68% 的参考纹理方向。
