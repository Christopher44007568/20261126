# texaudit

一个只使用 Go 标准库的纹理色彩空间审计工具。它从 JSON 清单读取
albedo、normal、roughness 三种纹理，按清单里的 `color_space` 解释数据，
渲染正确的 PBR 材质球，并生成“强制把 albedo 当 sRGB”时的对照图、
局部误差图和统计报告。

文件名和扩展名不参与角色或色彩空间判断。三个输入故意命名为
`surface-a.png`、`surface-b.png`、`surface-c.png`，实际语义只来自清单。
清单声明是唯一权威，工具不会用 PNG 内嵌标记或文件名来改写它。

## 快速开始

```bash
go run . -manifest artifacts/demo/manifest.json -out artifacts/demo

# 或重新生成内置 demo 输入后审计
go run . -demo -out artifacts/demo
go test ./...
```

内置 demo 会生成线性编码的 albedo、线性法线和线性粗糙度，然后输出：

| 文件 | 含义 |
| --- | --- |
| `correct.png` | 按声明解释后的正确材质球 |
| `forced-srgb-albedo.png` | 只把 albedo 错误解释为 sRGB 后的材质球 |
| `comparison.png` | 正确/误读并排对照图 |
| `local-error.png` | 显示空间局部误差热图，默认放大 2 倍 |
| `report.json` | 纹理声明、SHA-256、误差统计和线性数据不变量 |

![正确材质球](artifacts/demo/correct.png)

![并排对照](artifacts/demo/comparison.png)

![局部误差](artifacts/demo/local-error.png)

## 真实场景

```bash
go run . -manifest scene.json -out audit-output -error-gain 2
```

`scene.json` 示例：

```json
{
  "title": "material checks",
  "render": {
    "width": 720,
    "height": 720,
    "samples": 3,
    "exposure": 1.0
  },
  "material": {
    "metallic": 0.0,
    "specular": 0.5,
    "normal_strength": 1.0
  },
  "light": {
    "azimuth_degrees": -35,
    "elevation_degrees": 42,
    "intensity": 4.2,
    "ambient": 0.1
  },
  "textures": {
    "albedo": {
      "path": "textures/color.png",
      "color_space": "srgb",
      "channel": "rgb"
    },
    "normal": {
      "path": "textures/vector.png",
      "color_space": "linear",
      "channel": "rgb"
    },
    "roughness": {
      "path": "textures/gloss.png",
      "color_space": "linear",
      "channel": "r"
    }
  }
}
```

纹理路径相对清单位置解析。输入支持 Go 标准库可解码的 PNG/JPEG；
输出为 16-bit PNG，避免误差图和渐变出现额外量化。

## 色彩空间规则

- `albedo` 可声明 `srgb` 或 `linear`。声明 `srgb` 时按
  IEC 61966-2-1 分段函数解码到线性光；声明 `linear` 时直接使用原值。
- `normal` 和 `roughness` 只能声明 `linear`。清单校验会拒绝 `srgb`，
  因为这两种纹理是数据而不是显示颜色。
- 对照变体 `forced-srgb-albedo.png` 只重新解释 albedo。法线、粗糙度在
  正确与误读路径中使用完全相同的原始线性采样，不存在传递函数入口。
- 因此不会对法线或粗糙度做“颜色修正”，也不会因此出现假高光。
  若 albedo 本来就声明为 `srgb`，误读变体与正确图相同，报告中的
  `misread_has_effect` 为 `false`。

法线贴图按切线空间 OpenGL 约定解释（绿色通道向上）。粗糙度支持
`r`、`g`、`b`、`a` 或线性亮度 `luma`。

## 误差定义

`local-error.png` 比较两张最终显示编码图。每个像素的局部误差为 RGB
三个通道绝对差的平均值，再乘以 `-error-gain` 后映射到热图。JSON 报告保留
未经放大的 `mean_absolute_error`、`rmse`、`p95`、`p99`、`max` 和变化像素数。

报告还会记录每个纹理的声明色彩空间、实际传递函数、尺寸和 SHA-256，便于
把一次审计结果绑定到具体输入。

## 验证

```bash
go test ./...
go vet ./...
```

测试覆盖 sRGB 编解码、声明优先于文件名、法线/粗糙度零传递、中性法线、
误差图数值、并排图尺寸，以及完整 CLI 审计输出。
