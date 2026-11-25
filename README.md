# BD4X：4×4 基色 + 增量块压缩

纯 Go、仅使用标准库的 4×4 RGBA 纹素块压缩演示。每个块使用 RGB565 基色，
每个纹素每通道保存 2 位增量；透明度按块选择四种记录：

| alpha 模式 | 记录长度 | 用途 |
| --- | ---: | --- |
| `opaque` | 17 B | 16 个纹素全不透明 |
| `mask1` | 19 B | 二值透明，16 位掩码逐像素精确复原 |
| `alpha4` | 25 B | 半透明，每纹素 4 位 alpha |
| `empty` | 1 B | 整块完全透明 |

块内布局：

```text
header(1) = 0xD0 | alpha_mode
base(2)   = RGB565
spans(2)  = R/G/B 各 5 位跨度
codes(12) = 16 个纹素 × 3 通道 × 2 位增量
alpha     = mask1 时 2 字节；alpha4 时 8 字节
```

## 运行

```sh
go run . -out artifacts
go test ./...
```

程序生成：

- `artifacts/overview.png`：行是平滑渐变、锐边、透明洞；列是原图、解码图、RGBA 误差 ×8。
- `artifacts/original.png`、`decoded.png`、`error_x8.png`：三类场景的横向展开图。
- `artifacts/<scenario>_compare.png`：单场景三联图。
- `artifacts/streams/*.bd4`：实际压缩流；`.hex.txt` 为完整字节 dump。
- `artifacts/blocks_*.txt`：每个 4×4 块的坐标、模式、基色、跨度与原始十六进制。
- `artifacts/report.html`：图表、全部块字节和指标的报告。
- `artifacts/report.txt`、`metrics.json`、`compressed_hex.txt`：文本与机器可读结果。

终端也会打印代表块的压缩字节。误差图中洋红色专门表示
可见/透明分类翻转，而不是普通 RGB 误差。

## 透明边缘审计

RGB MAE 只在源图像可见纹素上统计。透明破裂单独报告：

- `alpha_class_errors`：可见与透明分类发生翻转的纹素数。
- `hole_fill_errors` / `hole_erosion_errors`：空洞被填上 / 可见边缘被挖掉。
- `boundary_alpha_mae`：源 alpha 边界上的 alpha 平均绝对误差。

测试中还有一个刻意构造的用例：只翻转一个洞边缘纹素时，可见 RGB MAE
仍为 `0`，但 `alpha_class_errors` 会立即变为 `1`。因此低均值不能掩盖透明边缘破裂。
