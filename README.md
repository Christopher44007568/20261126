# patchcheck

纯 Go 标准库的色块检查器。它按清单解析色块区域，检查亮度、CIEDE2000 色差、灰阶单调性和 sRGB 色域裁切，并始终生成一张可查看的原图 / 目标图 / 误差放大三栏对照图。

![对照图示例](artifacts/comparison.png)

示例刻意包含“平均值通过、局部失败”的情况：平均 ΔE 低于阈值，但高光 `HL-01`、肤色 `SKIN-03` 和灰阶 `N-01/N-02` 分别触发局部规则。最终状态仍为 `FAIL`，原图与目标图完整保留，失败区域用颜色和嵌套边框标出。

## 运行

要求 Go 1.22+，不安装任何第三方模块。

```sh
go build -o patchcheck .
./patchcheck -manifest examples/manifest.json -out artifacts/comparison.png
```

生成：

- `artifacts/comparison.png`：三栏对照图。
- `artifacts/comparison.report.json`：逐色块指标、失败原因、平均值和分组单调性结果。

可选参数：

```text
-source PATH       原图，支持 PNG/JPEG/GIF
-target PATH       目标图，格式同上；必须与 -source 同时提供
-report PATH       报告路径，默认 <输出文件名>.report.json
-fail-on-error     存在局部失败时以退出码 2 结束
```

不提供 `-source/-target` 时，程序会按清单中的 `source/target` 十六进制颜色生成两张色块图，因此示例可直接运行。外部图片大于清单画布时按左上角裁到画布，小于画布则报错。

## 清单格式

```json
{
  "title": "LOCAL COLOR GATE",
  "canvas": { "width": 720, "height": 540 },
  "output": { "amplification": 10 },
  "groups": [
    { "id": "ramp", "name": "NEUTRAL RAMP", "direction": "increasing" }
  ],
  "patches": [
    {
      "id": "SKIN-03",
      "kind": "skin",
      "rect": { "x": 366, "y": 24, "w": 156, "h": 156 },
      "source": "#B97B63",
      "target": "#C88870"
    }
  ]
}
```

关键字段：

- `patches[].rect`：色块区域，单位为像素。
- `patches[].kind`：`patch`、`skin` 或 `highlight`，后两类可套用更严格的局部阈值。
- `patches[].source/target`：未提供外部图片时用于生成原图与目标图。
- `patches[].group/order`：同一亮度序列的分组及顺序；`direction` 支持 `increasing`、`decreasing`、`auto`、`none`。
- `output.amplification`：误差图的通道差放大倍数。

## 判定规则

- 亮度：比较色块区域线性光均值转换后的 CIELAB `L*`，使用 `|ΔL*|`。
- 色差：使用 CIEDE2000 `ΔE00`。
- 单调性：按 `group/order` 检查目标 `L*` 是否保持指定方向。`auto` 从原图首尾亮度推断方向，`none` 跳过。
- 色域裁切：统计目标色块中任一通道达到 `clipHigh` 或低于 `clipLow`、且原图同一像素未裁切的占比，即 `newClipRatio`。
- 局部失败：任一色块超过色差、亮度或裁切阈值，或落入单调性违规，即失败。
- 平均值通过：仅表示 `meanDeltaE <= deltaE` 且 `meanAbsDeltaL <= brightness`，不会覆盖任何局部失败。
- 最终状态：只有所有局部规则和分组规则都通过才是 `pass`。

`highlightDeltaE`、`skinDeltaE` 等分类阈值会与全局阈值比较并取更严格者。完整默认值见 `manifest.go`。

## 可视化颜色

- 品红：色差或亮度超限。
- 青色：新增色域裁切。
- 琥珀：单调性违规。
- 绿色：通过。

一个色块同时命中多类规则时，外框使用第一类颜色，内层增加其他类别的彩色边框；小标签右侧也显示对应色标。误差图灰度越亮表示原图与目标图的通道差越大，色块覆盖层仍然标出所有超限区域。

## 验证

```sh
go test ./...
go vet ./...
```

测试覆盖 CIEDE2000 参考值、线性裁切检测、灰阶断点、平均通过但局部失败的保留画面，以及 PNG/JSON 输出。

对照图内文字使用内置 5×7 点阵字体，因此不依赖系统字体或外部组件。非 ASCII 标签仍完整保存在 JSON 报告中，图中使用色块 ID 标识。
