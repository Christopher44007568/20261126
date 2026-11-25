# AtlasPad：Go 标准库图集 Padding / 旋转装箱 / Mip 采样验证

本项目完全使用 Go 标准库实现，不依赖图像、字体、数学或装箱第三方包。运行示例会生成一套故意紧贴的图集和一套 mip-safe padding 图集，并用 CPU 双线性/三线性采样逐层验证串色。

## 运行

```bash
go test ./...
go run . -out artifacts/generated
```

也可以显式写 `demo` 子命令：

```bash
go run . demo -out artifacts/generated
```

## 生成产物

输出目录默认是 `artifacts/generated/`：

- `atlas_layout.png`：atlas 布局图；青框是子纹理 UV/Frame，黄框是包含 padding 的装箱 Cell，角标对应报告中的放置顺序。
- `bleed_highlight.png`：紧贴、无 padding 图集的串色高亮；红圈来自运行期采样误差最大的位置。
- `bleed_highlight_safe.png`：padding 修复后的采样校验图，绿框表示所有 mip 均通过。
- `bleed_comparison.png`：修复前/后并排诊断图。
- `mip_padding.png`：各级 mip 以及随层级换算的 padding 框。
- `utilization.png`、`utilization.json`：纹理面积、padding 面积、空闲面积、Cell 覆盖率和装箱有效率。
- `validation_tight.json`、`validation_safe.json`：每个采样点的 UV、实际色、期望色和误差。
- `layout.json`：矩形、旋转状态、UV、padding 策略和利用率输入。
- `report.md`：上述结果的文字汇总。

## Padding 与 Mip

基础双线性采样在 UV 边界会读到边界外的 1 个 texel。mip level `l` 的一个纹素覆盖 `2^l` 个 level-0 纹素，因此本实现使用：

```text
RequiredBasePadding(l) = Padding << l
PackPadding            = Padding << (MipLevels - 1)
```

装箱时一次性预留最深 mip 所需的最大间距；level 0 做边缘像素挤出，之后每降一级都重新绘制/挤出该层对应的 gutter，而不是只修最高分辨率。为了让 mip 2×2 块相位对齐，构建器默认把 Cell 原点对齐到 `1 << (MipLevels-1)`，示例图中的尺寸也都是 mip 对齐尺寸。

## 核心 API

```go
policy := atlas.PaddingPolicy{Padding: 1, MipLevels: 4}

built, err := atlas.BuildAtlas(sprites, atlas.BuildOptions{
    MipLevels: 4,
    Extrude:   true,
    Pack: atlas.PackOptions{
        Width: 512, Height: 384,
        AllowRotation: true,
        Policy: policy,
        // Alignment 留 0 时，BuildAtlas 会按最深 mip 自动对齐。
    },
})

report := atlas.ValidateSampling(built, "my-atlas", atlas.ValidateOptions{
    Grid: 11,
    ColorTolerance: 24,
})

rt := atlas.NewRuntimeAtlas(built)
c, _, err := rt.SampleMip("tex-01-grid", 0.0, 0.5, 3)
c, err = rt.SampleLOD("tex-01-grid", 0.0, 0.5, 2.5)
```

装箱器是确定性的 MaxRects / best-short-side-fit，支持 90° 旋转候选、固定尺寸或二次幂方形自动扩尺寸。旋转 UV 的正反映射由 `SpriteToAtlasUV` 与 `AtlasToSpriteUV` 提供。

## 当前示例结果

示例为 `512×384`、4 层 mip：

- 无 padding 图集在 mip `0/1/2/3` 都能检出边界串色。
- mip-expanded padding 图集的 4840 个逐层采样全部通过，最大 RGBA 距离约 1.7。
- 原始纹理利用率约 `45.6%`；包含 padding Cell 的覆盖率约 `63.0%`；Cell 内有效率约 `72.3%`。

这些数字来自确定性生成的示例纹理；替换成自己的 `image.Image` 后，以 `validation_*.json` 和 `report.md` 的实际结果为准。
