# 20261126

Repository initialized for 20261126.

## HDR 截图双路导出（Go，零第三方依赖）

导出 HDR 截图时若缺少亮度范围与色彩元数据，预览软件可能再次套用 Gamma，
导致画面发灰发白。本程序用纯 Go 标准库实现完整的双路导出与校验流程。

### 功能

- 合成确定性的线性光 HDR 场景，发光体峰值约 10，含大量超过 SDR 白
  （线性 1.0）的高光像素。
- 浮点 PPM 路：输出 PFM（`PF` 魔数、RGB float32、小端、bottom-up），
  保留完整线性光动态范围；PFM 本身无任何色彩元数据。
- 8-bit PNG 预览路：Reinhard 色调映射 + sRGB OETF 量化，PNG 内嵌
  `sRGB` / `gAMA` / `cHRM` / `tEXt` 块，明确声明像素已编码，查看器不会
  二次应用 Gamma。
- 预览图侧边面板：亮度刻度（线性刻度 + SDR 白标记）、场景最大码值、
  超白像素统计、逐步转换说明与 Gamma 安全说明。
- Reload 校验：把写盘的 PFM 重新解码，与内存原图逐像素比较，并生成
  before/after/差异 三联图；同页附带“正确 sRGB / 二次 Gamma / 原始线性”
  三条灰度演示条，直观显示二次 Gamma 的发白效果。

### 运行

```sh
go run . -out output
# 可选参数：-w 640 -h 400 -exposure 1.0
go test ./...
```

产物（默认在 `output/`）：

- `scene.pfm`：浮点 PPM 主数据。
- `scene_preview.png`：带注释面板的 8-bit sRGB 预览。
- `reload_compare.png`：reload 前后对比、差异图与 Gamma 演示。
- `conversion_report.txt`：中文转换说明与误差报告。

### 代码结构

- `internal/hdr`：float32 RGB 图像与亮度统计。
- `internal/scene`：合成 HDR 场景。
- `internal/pfm`：PFM 编解码（支持 PF/Pf、注释、大小端）。
- `internal/colorx`：sRGB 传递函数与 Reinhard 色调映射。
- `internal/pngx`：自带元数据块的零依赖 PNG 编码器。
- `internal/raster`、`internal/font`：8-bit 画布与内置 5x7 位图字体。
- `internal/preview`：LDR 渲染与注释面板。
- `internal/reload`：往返对比与 Gamma 演示页。
