# TextureLab：Go 实现的 KTX 风格纹理容器

纯 Go 标准库实现，不链接 OpenGL、Vulkan、DirectX、Metal 或其他图形 SDK；PNG 输出只使用 `image/png`。

一次 `demo` 会生成：

- `512x512`、`256x256`、`128x128` 三档完整 mip 链。
- sRGB 标记的反照率容器与线性标记的法线容器。
- 一个故意把同一法线字节标成 sRGB 的错误容器。
- 三档 mip 拼图、法线拼图和放大的逐层色差图。
- CSV / JSON / Markdown 元数据表，包含层级、拼图坐标、文件偏移、大小、CRC32 和 SHA256。
- 重载材质的线性指纹报告，以及法线误标 sRGB 的逐层稳定色差报告。

## 运行

```bash
go test ./...
go run . demo -out artifacts/generated
go run . inspect --json artifacts/generated/normal_linear.ktxc
```

也可以使用标准入口：

```bash
go run ./cmd/texcontainer demo -out artifacts/generated
```

## 关键产物

```text
artifacts/generated/albedo_srgb.ktxc
artifacts/generated/normal_linear.ktxc
artifacts/generated/normal_mislabeled_srgb.ktxc
artifacts/generated/albedo_mip_atlas.png
artifacts/generated/normal_mip_atlas.png
artifacts/generated/normal_srgb_delta_atlas.png
artifacts/generated/mip_metadata.csv
artifacts/generated/mip_metadata.json
artifacts/generated/mip_metadata.md
artifacts/generated/roundtrip_report.json
artifacts/generated/normal_srgb_diagnostic.json
artifacts/generated/normal_srgb_diagnostic.md
```

## 不偏色的保证

- 容器逐字节保存 RGBA8 mip 纹素，并保留 `linear` / `srgb` 标记。
- sRGB 反照率在 mip 降采样时先转线性光求 2x2 平均，再编码回 sRGB，避免 gamma 空间平均导致发暗或偏色。
- 法线图按线性向量数据处理：解码为 XYZ、求平均并重新归一化。
- 材质采样时，只有 `srgb` 标记的 RGB 会走 sRGB EOTF；alpha 和 `linear` 数据不做传递曲线转换。
- 重载后逐层比较原始纹素 SHA256 与线性材质采样 SHA256；`roundtrip_report.json` 中为 `"no_color_shift": true`。

## 法线误标 sRGB

法线必须保持 `linear` / data 标记。把完全相同的法线字节改标为 `srgb` 后，材质会错误地对 XY 通道执行 sRGB 解码。中间值 `128` 的线性值会从约 `0.501961` 变成 `0.215861`，产生可重复的明显色差。

诊断输出包含每一层的：

- 最大绝对线性误差和平均 RGB 误差。
- 变化纹素数量、比例和包围盒。
- 最大误差纹素的坐标、通道、字节偏移和前后线性值。
- 该层差异图的 SHA256 指纹。
- `first_failing_level` 与 `worst_level`，可直接定位到具体 mip 层级。

当前程序化样例的三层 `512 / 256 / 128` 都会稳定检出，首个失配层级为 level `0`；元数据表中每一行也带有对应层的误差和拼图矩形。

## TXCON1 容器布局

所有整数字段使用小端序。扩展名为 `.ktxc`，它是 KTX 思想的自定义容器，不是官方 KTX 文件。

固定头为 64 字节：

```text
偏移  长度  字段
0     8    Magic = "TXCON1\\0\\0"
8     4    Version = 1
12    4    HeaderSize = 64
16    4    Format = 1 (RGBA8)
20    4    BaseWidth
24    4    BaseHeight
28    4    MipLevelCount
32    4    MetadataLength
36    4    IndexLength
40    8    PixelSectionLength
48    1    ColorSpace (0=linear, 1=sRGB)
49    1    Flags
50    14   Reserved
```

头后依次是排序键值元数据、40 字节一项的 mip 索引和 4 字节对齐的像素段。索引项记录 level、宽高、相对像素段偏移、大小、CRC32 与保留字段。解码时会校验 magic、长度、mip 序号、RGBA8 载荷长度和每级 CRC32。

## 代码结构

- `colorspace/`：sRGB 传递函数与线性采样。
- `texcontainer/`：TXCON1 编码、解码、校验和布局检查。
- `mipchain/`：程序化反照率/法线、线性光降采样、法线向量归一化。
- `material/`：材质重载、线性采样和逐层指纹。
- `diagnose/`：逐层误标诊断与 Markdown 报告。
- `atlas/`：mip 拼图和 PNG 输出。
- `catalog/`：CSV、JSON、Markdown 元数据表。
- `workflow/`：完整确定性生成流程。
- `cli/`、`main.go`、`cmd/texcontainer/`：命令行入口。
