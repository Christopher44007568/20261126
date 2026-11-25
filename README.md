# DDS 子集预览器

纯 Go、仅标准库实现的 DDS 解析与 CPU 预览工具。不依赖 OpenGL、Vulkan、DirectX
或任何图像/图形 SDK。

## 构建与运行

```bash
go test ./...
go build -o dds-preview .

./dds-preview -out preview path/to/texture.dds
./dds-preview path/to/texture.dds preview
```

也可以一次传入多个文件，后一个文件只有在完整解析成功时才替换当前预览：

```bash
./dds-preview -out preview good.dds truncated.dds
```

## 支持的 DDS 子集

- 4 字节 `DDS ` magic，124 字节旧版头；`DX10` fourCC 的可选 20 字节扩展头。
- 2D 纹理与六面 cubemap（`+X,-X,+Y,-Y,+Z,-Z`），数据按 **face 优先、每面
  mip 链连续** 的旧版 DDS 顺序读取。
- 1..N 级 mip；每级尺寸从基础尺寸逐级减半，最小为 1。
- 未压缩 8/16/24/32 位、由 mask 定义的 RGBA/BGR/luminance 数据，因此
  `RGBA8`、`BGRA8`、`BGRX8` 等通道顺序由 mask 正确解码。
- DX10 未压缩格式：`28,29,87,88,91,24,61,65,49`。
- `DDSD_PITCH`：基础层使用 `dwPitchOrLinearSize` 作为含行尾填充的行宽
  读取；`DDSD_LINEARSIZE`：由顶层 linear size / height 反推行宽。后续 mip
  按紧凑行宽读取。

`DXT/BC` 压缩、volume texture、非 1 或 6 的 texture array 会返回明确的
字段错误，而不是猜测解码。

## 输出

单面纹理：

```text
preview/
  mip_0.ppm
  mip_1.ppm
  ...
  sphere.ppm
  report.json
```

cubemap 每个面的每个 mip：

```text
preview/
  face_0_mip_0.ppm
  face_0_mip_1.ppm
  ...
  face_5_mip_N.ppm
  sphere.ppm
  report.json
```

- `mip_*.ppm` / `face_*_mip_*.ppm` 是该 mip 原始尺寸的二进制 P6 PPM。
- `sphere.ppm` 是软件光线求值的材质球，尺寸等于基础 mip 的
  `width x height`；普通纹理使用球面 UV，cubemap 使用方向到 cube face 的采样。
- `report.json` 包含纹理摘要、原始像素格式 mask、所有产物路径，以及
  `field/offset/message/expected/actual` 字段级错误。

## 截断与非法尺寸

解析采用“全有或全无”提交：

1. 先校验 magic、头字段、格式、face 数、mip 链长度和每级数据范围。
2. 只有所有 face/mip 的全部行都能落在文件内，才生成 `Texture`。
3. `Previewer.Load` 失败时保留上一张有效纹理；截断或非法尺寸不会生成任何
   半截 PPM/材质球，也不会覆盖旧图像。
4. 失败时只更新 `report.json`，其中 `keptPreviousTexture=true` 表示旧纹理
   仍在使用。

典型 stderr：

```text
broken.dds: invalid DDS; no texture committed
  kept previous texture from good.dds
  data.face[2].mip[1] @312: texture data is truncated; the last valid texture must be kept
    (expected complete 2x2 mip with pitch 8, ending at byte 328, got file ends at byte 324)
```

## 代码结构

```text
main.go                       CLI 与多文件加载
internal/dds/header.go        旧版/DX10 头校验
internal/dds/format.go        mask、DXGI 与 pitch 布局
internal/dds/decode.go        全量预校验、RGBA 解码
internal/preview/state.go     上一有效纹理的状态机
internal/preview/sphere.go    纯 CPU 材质球
internal/preview/output.go    PPM、原子文件写入与报告
```
